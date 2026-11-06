package cmd

import (
	"strings"
	"testing"

	"register/api/services/sheets_service"
	"register/api/services/update_service"
	"register/pkg/banking"
	cfg "register/pkg/config"
	"register/pkg/models"

	"github.com/plaid/plaid-go/v15/plaid"
)

func TestNeedsNameForRegister(t *testing.T) {
	tests := []struct {
		name string
		t    *models.Transaction
		want bool
	}{
		{"empty name needs one", &models.Transaction{BankName: "ACME MARKET", Name: ""}, true},
		{"bank name as placeholder with no column needs one", &models.Transaction{BankName: "ACME MARKET", Name: "ACME MARKET", ColumnIndex: 0}, true},
		{"bank name as placeholder with a column does not", &models.Transaction{BankName: "ACME MARKET", Name: "ACME MARKET", ColumnIndex: 11}, false},
		{"a real name does not", &models.Transaction{BankName: "ACME MARKET", Name: "Acme", ColumnIndex: 11}, false},
		{"a real name with no column does not", &models.Transaction{BankName: "ACME MARKET", Name: "Acme", ColumnIndex: 0}, false},
		{"check with empty name takes the check path", &models.Transaction{BankName: "CHECK # 1234", Name: ""}, false},
		{"check with placeholder name takes the check path", &models.Transaction{BankName: "CHECK # 1234", Name: "CHECK # 1234", ColumnIndex: 0}, false},
		{"check substring anywhere in the bank name", &models.Transaction{BankName: "BANK CHECK # 99", Name: ""}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := needsNameForRegister(tc.t); got != tc.want {
				t.Errorf("needsNameForRegister(%+v) = %v, want %v", tc.t, got, tc.want)
			}
		})
	}
}

func TestNeedUserInfoForTransactions(t *testing.T) {
	tests := []struct {
		name  string
		trans []*models.Transaction
		want  bool
	}{
		{"nil", nil, false},
		{"empty", []*models.Transaction{}, false},
		{"all named", []*models.Transaction{
			{BankName: "ACME", Name: "Acme", ColumnIndex: 11},
			{BankName: "SHOP", Name: "Shop", ColumnIndex: 12},
		}, false},
		{"one unnamed", []*models.Transaction{
			{BankName: "ACME", Name: "Acme", ColumnIndex: 11},
			{BankName: "MYSTERY", Name: ""},
		}, true},
		{"placeholder name with no column", []*models.Transaction{
			{BankName: "MYSTERY", Name: "MYSTERY", ColumnIndex: 0},
		}, true},
		{"checks do not count", []*models.Transaction{
			{BankName: "CHECK # 1", Name: ""},
		}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := needUserInfoForTransactions(tc.trans); got != tc.want {
				t.Errorf("needUserInfoForTransactions = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestCategoryColumns(t *testing.T) {
	tests := []struct {
		name    string
		columns []models.Column
		want    []string // names of the kept columns, in order
	}{
		{"nil", nil, nil},
		{"empty", []models.Column{}, nil},
		{"mixed keeps only categories in order", []models.Column{
			{ID: 1, Name: "Recon", ColumnIndex: 1},
			{ID: 11, Name: "Grocery", ColumnIndex: 11},
			{ID: 2, Name: "Date", ColumnIndex: 3},
			{ID: 12, Name: "Rent", ColumnIndex: 12},
		}, []string{"Grocery", "Rent"}},
		{"no categories", []models.Column{
			{ID: 1, Name: "Recon", ColumnIndex: 1},
			{ID: 2, Name: "Date", ColumnIndex: 3},
		}, nil},
		{"first category index counts", []models.Column{
			{ID: 10, Name: "LastFixed", ColumnIndex: 10},
			{ID: 11, Name: "Grocery", ColumnIndex: 11},
		}, []string{"Grocery"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := categoryColumns(tc.columns)
			if len(got) != len(tc.want) {
				t.Fatalf("categoryColumns returned %d columns, want %d: %v", len(got), len(tc.want), got)
			}
			for i, name := range tc.want {
				if got[i].Name != name {
					t.Errorf("categoryColumns()[%d].Name = %q, want %q", i, got[i].Name, name)
				}
			}
		})
	}
}

func TestGetAmount(t *testing.T) {
	tests := []struct {
		name                    string
		source                  string
		deposit, withdrawal, cp float64
		want                    float64
	}{
		{"WellsFargo deposit", "WellsFargo", 100, 0, 0, 100},
		{"WellsFargo withdrawal", "WellsFargo", 0, 40, 0, 40},
		{"WellsFargo deposit wins over withdrawal", "WellsFargo", 100, 40, 0, 100},
		{"WellsFargo all zero", "WellsFargo", 0, 0, 99, 0},
		{"Chase uses credit purchases", "Chase", 100, 50, 25, 25},
		{"Fidelity uses credit purchases", "Fidelity", 0, 0, 7.5, 7.5},
		{"unknown source uses credit purchases", "Other", 100, 50, 3, 3},
		{"empty source uses credit purchases", "", 100, 50, 3, 3},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := getAmount(tc.source, tc.deposit, tc.withdrawal, tc.cp); got != tc.want {
				t.Errorf("getAmount(%q, %v, %v, %v) = %v, want %v",
					tc.source, tc.deposit, tc.withdrawal, tc.cp, got, tc.want)
			}
		})
	}
}

func TestDashes(t *testing.T) {
	tests := []struct {
		count int
		want  string
	}{
		{0, ""},
		{1, "-"},
		{5, "-----"},
		{10, "----------"},
	}
	for _, tc := range tests {
		if got := dashes(tc.count); got != tc.want {
			t.Errorf("dashes(%d) = %q, want %q", tc.count, got, tc.want)
		}
	}
}

func TestReadString(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"plain line", "hello\n", "hello"},
		{"EOF without newline", "hello", "hello"},
		{"empty line", "\n", ""},
		{"empty input", "", ""},
		{"only the first line is read", "first\nsecond\n", "first"},
		{"spaces are kept", "  padded  \n", "  padded  "},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var got string
			captureStdout(t, func() {
				got = readString(newReader(tc.input), "Prompt: ")
			})
			if got != tc.want {
				t.Errorf("readString(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestReadInt(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    int
		wantErr string // substring of the error; "" means no error
	}{
		{"positive", "42\n", 42, ""},
		{"negative", "-3\n", -3, ""},
		{"zero", "0\n", 0, ""},
		{"EOF without newline", "7", 7, ""},
		{"not a number", "abc\n", 0, "could not convert abc to an int"},
		{"decimal", "4.5\n", 0, "could not convert 4.5 to an int"},
		{"leading space", " 42\n", 0, "could not convert  42 to an int"},
		{"empty", "\n", 0, "could not convert  to an int"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var got int
			var err error
			captureStdout(t, func() {
				got, err = readInt(newReader(tc.input), "Column ID: ")
			})
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("readInt(%q) returned unexpected error: %v", tc.input, err)
				}
				if got != tc.want {
					t.Errorf("readInt(%q) = %d, want %d", tc.input, got, tc.want)
				}
				return
			}
			if err == nil {
				t.Fatalf("readInt(%q) = %d, want error containing %q", tc.input, got, tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("readInt(%q) error = %q, want it to contain %q", tc.input, err.Error(), tc.wantErr)
			}
		})
	}
}

func TestGetTransactionNotes(t *testing.T) {
	trans := []*models.Transaction{
		{BankName: "CHK", Name: models.CheckRegisterName, Source: "WellsFargo", Date: "2026-11-01", Amount: 25},
		{BankName: "AMZN", Name: "Amazon", Source: "Chase", Date: "2026-11-01", Amount: 9.99},
		{BankName: "AMZN", Name: "Amazon Marketplace", Source: "Chase", Date: "2026-11-02", Amount: 14.5},
		{BankName: "ACME", Name: "Acme", Source: "Chase", Date: "2026-11-02", Amount: 5},
		{BankName: "ACME", Name: "amazon", Source: "Chase", Date: "2026-11-02", Amount: 5},
	}
	var got []*models.Transaction
	captureStdout(t, func() {
		got = getTransactionNotes(newReader("plumber\nbook\nheadphones\n"), trans)
	})
	wantNotes := []string{"plumber", "book", "headphones", "", ""}
	for i, want := range wantNotes {
		if got[i].Note != want {
			t.Errorf("trans[%d] (%q).Note = %q, want %q", i, got[i].Name, got[i].Note, want)
		}
	}

	// No note-worthy transaction consumes no input and changes nothing.
	plain := []*models.Transaction{{BankName: "ACME", Name: "Acme"}}
	got = getTransactionNotes(newReader("unused\n"), plain)
	if got[0].Note != "" {
		t.Errorf("plain transaction got Note %q, want empty", got[0].Note)
	}
}

func TestPrintColumns(t *testing.T) {
	columns := []models.Column{
		{ID: 1, Name: "Recon", ColumnIndex: 1},
		{ID: 2, Name: "Date", ColumnIndex: 3},
		{ID: 11, Name: "Grocery", ColumnIndex: 11},
		{ID: 12, Name: "Rent", ColumnIndex: 12},
		{ID: 13, Name: "Dining", ColumnIndex: 13},
		{ID: 14, Name: "Travel", ColumnIndex: 14},
		{ID: 15, Name: "Fun", ColumnIndex: 15},
		{ID: 16, Name: "Clothes", ColumnIndex: 16},
		{ID: 17, Name: "Gifts", ColumnIndex: 17},
	}

	t.Run("six categories print as two rows of three", func(t *testing.T) {
		repo := &fakeQueryRepo{columns: columns[:8]} // Grocery..Fun: 6 categories
		var err error
		out := captureStdout(t, func() { err = printColumns(repo) })
		if err != nil {
			t.Fatalf("printColumns returned error: %v", err)
		}
		for _, name := range []string{"Grocery", "Rent", "Dining", "Travel", "Fun"} {
			if !strings.Contains(out, name) {
				t.Errorf("output missing %q:\n%s", name, out)
			}
		}
		// Non-categories are filtered out.
		for _, name := range []string{"Recon", "Date"} {
			if strings.Contains(out, name) {
				t.Errorf("output should not contain non-category %q:\n%s", name, out)
			}
		}
		if lines := strings.Count(out, "\n"); lines != 2 {
			t.Errorf("output has %d lines, want 2:\n%s", lines, out)
		}
	})

	t.Run("seven categories print two full rows plus a remainder row", func(t *testing.T) {
		repo := &fakeQueryRepo{columns: columns} // 7 categories
		var err error
		out := captureStdout(t, func() { err = printColumns(repo) })
		if err != nil {
			t.Fatalf("printColumns returned error: %v", err)
		}
		if lines := strings.Count(out, "\n"); lines != 3 {
			t.Errorf("output has %d lines, want 3:\n%s", lines, out)
		}
		if !strings.Contains(out, "Gifts") {
			t.Errorf("output missing remainder row %q:\n%s", "Gifts", out)
		}
	})

	t.Run("one category prints a single remainder row", func(t *testing.T) {
		repo := &fakeQueryRepo{columns: []models.Column{{ID: 11, Name: "Grocery", ColumnIndex: 11}}}
		var err error
		out := captureStdout(t, func() { err = printColumns(repo) })
		if err != nil {
			t.Fatalf("printColumns returned error: %v", err)
		}
		if lines := strings.Count(out, "\n"); lines != 1 {
			t.Errorf("output has %d lines, want 1:\n%s", lines, out)
		}
	})

	t.Run("no categories prints nothing", func(t *testing.T) {
		repo := &fakeQueryRepo{columns: []models.Column{{ID: 1, Name: "Recon", ColumnIndex: 1}}}
		var err error
		out := captureStdout(t, func() { err = printColumns(repo) })
		if err != nil {
			t.Fatalf("printColumns returned error: %v", err)
		}
		if out != "" {
			t.Errorf("output = %q, want empty", out)
		}
	})

	t.Run("repo error is returned", func(t *testing.T) {
		repo := &fakeQueryRepo{columnsErr: errTestBoom}
		var err error
		captureStdout(t, func() { err = printColumns(repo) })
		if err == nil {
			t.Fatal("printColumns returned nil error, want the repo error")
		}
		if !strings.Contains(err.Error(), "boom") {
			t.Errorf("printColumns error = %q, want it to contain %q", err.Error(), "boom")
		}
	})
}

func TestPrintTransactions(t *testing.T) {
	trans := []*models.Transaction{
		{TransactionID: "tx-new", Key: "k-new", Source: "Chase", Date: "2026-11-01", Name: "Acme", CreditPurchase: 7.5},
	}
	pending := []*models.Transaction{
		{TransactionID: "tx-pend", Key: "k-pend", Source: "WellsFargo", Date: "2026-11-01", Name: "Shop", Deposit: 50},
	}
	corrections := []*update_service.DateCorrection{
		{NewDate: "2026-11-02", Entry: &sheets_service.RegisterEntry{
			TransactionID: "tx-corr", Key: "k-corr", Source: "Chase",
			Date: "2026-11-01", Name: "Acme", CreditPurchase: 12.34,
		}},
	}

	out := captureStdout(t, func() {
		printTransactions(trans, pending, corrections)
	})
	// printTransactions no longer prints Transaction ID / Key — each row
	// is identified by its amount.
	for _, want := range []string{
		"Updated Transactions:", "12.34", "2026-11-02",
		"Pending Transactions:", "50.00", "Shop",
		"New Transactions:", "7.50", "Acme",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}

	// Nothing to print: only the leading blank line.
	out = captureStdout(t, func() {
		printTransactions(nil, nil, nil)
	})
	if out != "\n" {
		t.Errorf("empty printTransactions output = %q, want %q", out, "\n")
	}
}

func TestReadFromUser(t *testing.T) {
	columns := []models.Column{
		{ID: 11, Name: "Grocery", ColumnIndex: 11},
		{ID: 12, Name: "Rent", ColumnIndex: 12},
	}

	t.Run("exact match saves a rule and fills the transaction", func(t *testing.T) {
		repo := &fakeQueryRepo{columns: columns, maxPriority: 5}
		trans := []*models.Transaction{{BankName: "ACME MARKET", Amount: 12.34, Date: "2026-11-01"}}
		var updated bool
		var got []*models.Transaction
		var err error
		out := captureStdout(t, func() {
			updated, got, err = readFromUser(newReader("Acme Store\n11\nweekly shop\ne\n"), repo, trans)
		})
		if err != nil {
			t.Fatalf("readFromUser returned error: %v", err)
		}
		if !updated {
			t.Error("readFromUser updated = false, want true")
		}
		if got[0].Name != "Acme Store" {
			t.Errorf("Name = %q, want %q", got[0].Name, "Acme Store")
		}
		if got[0].Note != "weekly shop" {
			t.Errorf("Note = %q, want %q", got[0].Note, "weekly shop")
		}
		if got[0].ColumnIndex != 11 {
			t.Errorf("ColumnIndex = %d, want 11 (position of ID 11)", got[0].ColumnIndex)
		}
		if !strings.Contains(out, "Rule saved (priority 0)") {
			t.Errorf("output missing exact-rule priority line:\n%s", out)
		}
	})

	t.Run("bad column ID retries, then substring with a custom pattern", func(t *testing.T) {
		repo := &fakeQueryRepo{columns: columns, maxPriority: 10}
		trans := []*models.Transaction{{BankName: "ACME MARKET", Amount: 12.34, Date: "2026-11-01"}}
		var updated bool
		var err error
		out := captureStdout(t, func() {
			updated, _, err = readFromUser(newReader("Acme Store\nabc\n11\n\ns\nACME\n"), repo, trans)
		})
		if err != nil {
			t.Fatalf("readFromUser returned error: %v", err)
		}
		if !updated {
			t.Error("readFromUser updated = false, want true")
		}
		if !strings.Contains(out, "could not convert abc to an int") {
			t.Errorf("output missing the retry message for the bad column ID:\n%s", out)
		}
		if !strings.Contains(out, "Rule saved (priority 20)") {
			t.Errorf("output missing substring priority (max 10 + 10):\n%s", out)
		}
	})

	t.Run("substring with an empty pattern keeps the bank name", func(t *testing.T) {
		repo := &fakeQueryRepo{columns: columns, maxPriority: 0}
		trans := []*models.Transaction{{BankName: "ACME MARKET", Amount: 12.34, Date: "2026-11-01"}}
		var err error
		out := captureStdout(t, func() {
			_, _, err = readFromUser(newReader("Acme Store\n12\n\ns\n\n"), repo, trans)
		})
		if err != nil {
			t.Fatalf("readFromUser returned error: %v", err)
		}
		if trans[0].ColumnIndex != 12 {
			t.Errorf("ColumnIndex = %d, want 12", trans[0].ColumnIndex)
		}
		if !strings.Contains(out, "Rule saved (priority 10)") {
			t.Errorf("output missing substring priority line:\n%s", out)
		}
	})

	t.Run("unknown column ID leaves the column index unset", func(t *testing.T) {
		repo := &fakeQueryRepo{columns: columns}
		trans := []*models.Transaction{{BankName: "ACME MARKET", Amount: 12.34, Date: "2026-11-01"}}
		var err error
		captureStdout(t, func() {
			_, _, err = readFromUser(newReader("Acme Store\n99\n\n\n"), repo, trans)
		})
		if err != nil {
			t.Fatalf("readFromUser returned error: %v", err)
		}
		if trans[0].ColumnIndex != 0 {
			t.Errorf("ColumnIndex = %d, want 0 (ID 99 is not a column)", trans[0].ColumnIndex)
		}
	})

	t.Run("no transaction needs a name", func(t *testing.T) {
		repo := &fakeQueryRepo{columns: columns}
		trans := []*models.Transaction{{BankName: "ACME MARKET", Name: "Acme", ColumnIndex: 11}}
		var updated bool
		var err error
		captureStdout(t, func() {
			updated, _, err = readFromUser(newReader(""), repo, trans)
		})
		if err != nil {
			t.Fatalf("readFromUser returned error: %v", err)
		}
		if updated {
			t.Error("readFromUser updated = true, want false")
		}
	})

	t.Run("checks are skipped", func(t *testing.T) {
		repo := &fakeQueryRepo{columns: columns}
		trans := []*models.Transaction{{BankName: "CHECK # 1234", Name: ""}}
		var updated bool
		var err error
		captureStdout(t, func() {
			updated, _, err = readFromUser(newReader(""), repo, trans)
		})
		if err != nil {
			t.Fatalf("readFromUser returned error: %v", err)
		}
		if updated {
			t.Error("readFromUser updated = true, want false (checks take the note path)")
		}
	})

	t.Run("priority lookup error is returned", func(t *testing.T) {
		repo := &fakeQueryRepo{columns: columns, maxPriorityErr: errTestBoom}
		trans := []*models.Transaction{{BankName: "ACME MARKET", Amount: 12.34, Date: "2026-11-01"}}
		var err error
		captureStdout(t, func() {
			_, _, err = readFromUser(newReader("Acme Store\n11\n\ns\n\n"), repo, trans)
		})
		if err == nil {
			t.Fatal("readFromUser returned nil error, want the priority lookup error")
		}
		if !strings.Contains(err.Error(), "boom") {
			t.Errorf("readFromUser error = %q, want it to contain %q", err.Error(), "boom")
		}
	})

	t.Run("merchant save error is returned", func(t *testing.T) {
		repo := &fakeQueryRepo{columns: columns, createMerchantErr: errTestBoom}
		trans := []*models.Transaction{{BankName: "ACME MARKET", Amount: 12.34, Date: "2026-11-01"}}
		var err error
		captureStdout(t, func() {
			_, _, err = readFromUser(newReader("Acme Store\n11\n\ne\n"), repo, trans)
		})
		if err == nil {
			t.Fatal("readFromUser returned nil error, want the merchant save error")
		}
		if !strings.Contains(err.Error(), "boom") {
			t.Errorf("readFromUser error = %q, want it to contain %q", err.Error(), "boom")
		}
	})

	t.Run("column list error is returned", func(t *testing.T) {
		repo := &fakeQueryRepo{columnsErr: errTestBoom}
		trans := []*models.Transaction{{BankName: "ACME MARKET", Amount: 12.34, Date: "2026-11-01"}}
		var err error
		captureStdout(t, func() {
			_, _, err = readFromUser(newReader("Acme Store\n11\n\ne\n"), repo, trans)
		})
		if err == nil {
			t.Fatal("readFromUser returned nil error, want the column list error")
		}
		if !strings.Contains(err.Error(), "boom") {
			t.Errorf("readFromUser error = %q, want it to contain %q", err.Error(), "boom")
		}
	})
}

func TestGetUserInput(t *testing.T) {
	columns := []models.Column{
		{ID: 11, Name: "Grocery", ColumnIndex: 11, Color: models.ColorGreen},
	}
	lookup := []*models.DataRow{
		{BankName: "ACME MARKET", Name: "Acme", MatchType: "substring",
			ColumnIndex: 11, Color: models.ColorGreen, IsCategory: true},
	}
	bankClient := banking.NewClient("", nil, nil, "", "", plaid.Development, func(_ string) {})

	repo := &fakeQueryRepo{columns: columns, lookupData: lookup, maxPriority: 5}
	trans := []*models.Transaction{{BankName: "ACME MARKET", Amount: 12.34, Date: "2026-11-01"}}
	var got []*models.Transaction
	var err error
	out := captureStdout(t, func() {
		got, err = getUserInput(newReader("Acme Store\n11\n\ns\n\n"), bankClient, repo, trans)
	})
	if err != nil {
		t.Fatalf("getUserInput returned error: %v", err)
	}
	// The typed name is replaced by the lookup rule's name on the second pass.
	if got[0].Name != "Acme" {
		t.Errorf("Name = %q, want %q (lookup rule name)", got[0].Name, "Acme")
	}
	if got[0].ColumnIndex != 11 {
		t.Errorf("ColumnIndex = %d, want 11", got[0].ColumnIndex)
	}
	if got[0].Color != models.ColorGreen {
		t.Errorf("Color = %q, want %q", got[0].Color, models.ColorGreen)
	}
	if !strings.Contains(out, "Rule saved (priority 15)") {
		t.Errorf("output missing the saved-rule line:\n%s", out)
	}

	// A lookup error aborts the loop after the prompt pass.
	repo.lookupErr = errTestBoom
	trans = []*models.Transaction{{BankName: "ACME MARKET", Amount: 12.34, Date: "2026-11-01"}}
	captureStdout(t, func() {
		_, err = getUserInput(newReader("Acme Store\n11\n\ne\n"), bankClient, repo, trans)
	})
	if err == nil {
		t.Fatal("getUserInput returned nil error, want the lookup error")
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Errorf("getUserInput error = %q, want it to contain %q", err.Error(), "boom")
	}

	// A column-list error aborts after the matcher is built. The fake fails
	// GetColumns on the second call: the first (inside readFromUser)
	// succeeds, so the error surfaces from getUserInput's own call.
	repo.lookupErr = nil
	repo.lookupData = lookup
	repo.columnsErr = errTestBoom
	repo.columnsCalls = 0
	repo.columnsErrOnCall = 2
	trans = []*models.Transaction{{BankName: "ACME MARKET", Amount: 12.34, Date: "2026-11-01"}}
	captureStdout(t, func() {
		_, err = getUserInput(newReader("Acme Store\n11\n\ne\n"), bankClient, repo, trans)
	})
	if err == nil {
		t.Fatal("getUserInput returned nil error, want the column list error")
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Errorf("getUserInput error = %q, want it to contain %q", err.Error(), "boom")
	}
	repo.columnsErr = nil
	repo.columnsErrOnCall = 0

	// A bad regex in the lookup data fails the matcher build.
	repo.columnsErr = nil
	repo.lookupData = []*models.DataRow{
		{BankName: "X", Pattern: "[invalid", MatchType: "regex"},
	}
	trans = []*models.Transaction{{BankName: "ACME MARKET", Amount: 12.34, Date: "2026-11-01"}}
	captureStdout(t, func() {
		_, err = getUserInput(newReader("Acme Store\n11\n\ne\n"), bankClient, repo, trans)
	})
	if err == nil {
		t.Fatal("getUserInput returned nil error, want the bad-regex error")
	}
	if !strings.Contains(err.Error(), "bad regex") {
		t.Errorf("getUserInput error = %q, want it to contain %q", err.Error(), "bad regex")
	}

	// A bad regex in an income source fails the income matcher build.
	badIncomeClient := banking.NewClient("", nil,
		[]cfg.IncomeSource{{Name: "X", MatchPattern: "[invalid", MatchType: "regex"}},
		"", "", plaid.Development, func(_ string) {})
	repo.lookupData = lookup
	trans = []*models.Transaction{{BankName: "ACME MARKET", Amount: 12.34, Date: "2026-11-01"}}
	captureStdout(t, func() {
		_, err = getUserInput(newReader("Acme Store\n11\n\ne\n"), badIncomeClient, repo, trans)
	})
	if err == nil {
		t.Fatal("getUserInput returned nil error, want the bad income-regex error")
	}
	if !strings.Contains(err.Error(), "bad regex") {
		t.Errorf("getUserInput error = %q, want it to contain %q", err.Error(), "bad regex")
	}

	// A prompt-pass error (merchant save fails) is returned as-is.
	repo.lookupData = lookup
	repo.createMerchantErr = errTestBoom
	trans = []*models.Transaction{{BankName: "ACME MARKET", Amount: 12.34, Date: "2026-11-01"}}
	captureStdout(t, func() {
		_, err = getUserInput(newReader("Acme Store\n11\n\ne\n"), bankClient, repo, trans)
	})
	if err == nil {
		t.Fatal("getUserInput returned nil error, want the merchant save error")
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Errorf("getUserInput error = %q, want it to contain %q", err.Error(), "boom")
	}
}

func TestGatherTransactionInfo(t *testing.T) {
	columns := []models.Column{
		{ID: 11, Name: "Grocery", ColumnIndex: 11, Color: models.ColorGreen},
	}
	lookup := []*models.DataRow{
		{BankName: "ACME MARKET", Name: "Acme", MatchType: "substring",
			ColumnIndex: 11, Color: models.ColorGreen, IsCategory: true},
	}
	bankClient := banking.NewClient("", nil, nil, "", "", plaid.Development, func(_ string) {})

	t.Run("no info needed, only a note is prompted", func(t *testing.T) {
		trans := []*models.Transaction{
			{BankName: "AMZN", Name: "Amazon", Source: "Chase", Date: "2026-11-01", Amount: 9.99},
		}
		var got []*models.Transaction
		var err error
		captureStdout(t, func() {
			got, err = gatherTransactionInfo(newReader("bought a book\n"), nil, nil, trans)
		})
		if err != nil {
			t.Fatalf("gatherTransactionInfo returned error: %v", err)
		}
		if got[0].Note != "bought a book" {
			t.Errorf("Note = %q, want %q", got[0].Note, "bought a book")
		}
	})

	t.Run("info needed prompts for the merchant and applies the rule", func(t *testing.T) {
		repo := &fakeQueryRepo{columns: columns, lookupData: lookup, maxPriority: 5}
		trans := []*models.Transaction{
			{BankName: "ACME MARKET", Amount: 12.34, Date: "2026-11-01"},
		}
		var got []*models.Transaction
		var err error
		out := captureStdout(t, func() {
			got, err = gatherTransactionInfo(newReader("Acme Store\n11\n\ns\n\n"), bankClient, repo, trans)
		})
		if err != nil {
			t.Fatalf("gatherTransactionInfo returned error: %v", err)
		}
		if got[0].Name != "Acme" {
			t.Errorf("Name = %q, want %q", got[0].Name, "Acme")
		}
		if !strings.Contains(out, "Info needed...") {
			t.Errorf("output missing the info-needed banner:\n%s", out)
		}
	})

	t.Run("column list error is returned", func(t *testing.T) {
		repo := &fakeQueryRepo{columnsErr: errTestBoom}
		trans := []*models.Transaction{
			{BankName: "MYSTERY", Amount: 1, Date: "2026-11-01"},
		}
		var err error
		captureStdout(t, func() {
			_, err = gatherTransactionInfo(newReader(""), bankClient, repo, trans)
		})
		if err == nil {
			t.Fatal("gatherTransactionInfo returned nil error, want the column list error")
		}
		if !strings.Contains(err.Error(), "boom") {
			t.Errorf("gatherTransactionInfo error = %q, want it to contain %q", err.Error(), "boom")
		}
	})

	t.Run("user-input error is returned", func(t *testing.T) {
		repo := &fakeQueryRepo{columns: columns, lookupErr: errTestBoom}
		trans := []*models.Transaction{
			{BankName: "MYSTERY", Amount: 1, Date: "2026-11-01"},
		}
		var err error
		captureStdout(t, func() {
			_, err = gatherTransactionInfo(newReader("Mystery Store\n11\n\ne\n"), bankClient, repo, trans)
		})
		if err == nil {
			t.Fatal("gatherTransactionInfo returned nil error, want the lookup error")
		}
		if !strings.Contains(err.Error(), "boom") {
			t.Errorf("gatherTransactionInfo error = %q, want it to contain %q", err.Error(), "boom")
		}
	})
}
