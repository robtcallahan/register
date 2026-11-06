package csv

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"register/pkg/config"
	"register/pkg/models"
)

func writeCSVFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write fixture %s: %v", name, err)
	}
	return path
}

func TestNew(t *testing.T) {
	banks := map[string]config.Bank{
		models.FidelityID: {ID: models.FidelityID, CSVFileName: "fidelity.csv"},
	}
	progress := func(_ string) {}
	c := New("/tmp/fin", banks, []string{models.FidelityID}, progress)
	if c.FinanceDir != "/tmp/fin" {
		t.Errorf("FinanceDir = %q, want /tmp/fin", c.FinanceDir)
	}
	if len(c.Banks) != 1 || c.Banks[models.FidelityID].CSVFileName != "fidelity.csv" {
		t.Errorf("Banks = %v, want fidelity entry", c.Banks)
	}
	if len(c.BankIDs) != 1 || c.BankIDs[0] != models.FidelityID {
		t.Errorf("BankIDs = %v, want [fidelity]", c.BankIDs)
	}
	if c.Progress == nil {
		t.Error("Progress = nil, want non-nil func as passed")
	}
}

func Test_normalizeHeader(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"Date", "date"},
		{"  Transaction Date  ", "transaction date"},
		{"AMOUNT", "amount"},
		{"CHECK #", "check #"},
		{"", ""},
	}
	for _, tt := range tests {
		if got := normalizeHeader(tt.in); got != tt.want {
			t.Errorf("normalizeHeader(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

type csvHeadersSample struct {
	A string `csv:"Alpha"`
	B int    `csv:"-"`
	C string
	D string `csv:"Delta"`
}

func Test_csvHeaders(t *testing.T) {
	want := []string{"Alpha", "Delta"}
	for _, sample := range []any{csvHeadersSample{}, &csvHeadersSample{}} {
		got := csvHeaders(sample)
		if len(got) != len(want) {
			t.Fatalf("csvHeaders(%T) = %v, want %v", sample, got, want)
		}
		for i, w := range want {
			if got[i] != w {
				t.Errorf("csvHeaders(%T)[%d] = %q, want %q", sample, i, got[i], w)
			}
		}
	}
}

func Test_checkCSVHeaders(t *testing.T) {
	dir := t.TempDir()
	full := writeCSVFile(t, dir, "full.csv", `"Date","Transaction","Name","Memo","Amount"`+"\n")
	if err := checkCSVHeaders(full, FidelityVisa{}); err != nil {
		t.Errorf("checkCSVHeaders(full) = %v, want nil", err)
	}

	variant := writeCSVFile(t, dir, "variant.csv", `date, transaction ,NAME,memo,amount`+"\n")
	if err := checkCSVHeaders(variant, FidelityVisa{}); err != nil {
		t.Errorf("checkCSVHeaders(case/whitespace variant) = %v, want nil", err)
	}

	missing := writeCSVFile(t, dir, "missing.csv", `"Date","Transaction","Name","Amount"`+"\n")
	if err := checkCSVHeaders(missing, FidelityVisa{}); err == nil {
		t.Error("checkCSVHeaders(missing Memo) = nil, want error")
	} else if !strings.Contains(err.Error(), "Memo") {
		t.Errorf("checkCSVHeaders(missing Memo) = %v, want error naming Memo", err)
	}

	if err := checkCSVHeaders(filepath.Join(dir, "nope.csv"), FidelityVisa{}); err == nil {
		t.Error("checkCSVHeaders(nonexistent) = nil, want error")
	} else if !strings.Contains(err.Error(), "could not open file") {
		t.Errorf("checkCSVHeaders(nonexistent) = %v, want open error", err)
	}

	empty := writeCSVFile(t, dir, "empty.csv", "")
	if err := checkCSVHeaders(empty, FidelityVisa{}); err == nil {
		t.Error("checkCSVHeaders(empty) = nil, want error")
	}
}

func Test_readFileContents(t *testing.T) {
	dir := t.TempDir()
	path := writeCSVFile(t, dir, "data.csv", "hello")
	got, err := readFileContents(path)
	if err != nil {
		t.Fatalf("readFileContents = %v, want nil", err)
	}
	if string(got) != "hello" {
		t.Errorf("readFileContents = %q, want %q", got, "hello")
	}
	if got, err := readFileContents(filepath.Join(dir, "nope.csv")); err == nil {
		t.Error("readFileContents(missing file) = nil, want error")
	} else if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("readFileContents(missing file) = %v, want not-exist error", err)
	} else if got != nil {
		t.Errorf("readFileContents(missing file) = %q, want nil bytes", got)
	}
}

func Test_doWellsFargoHeadsExist(t *testing.T) {
	tests := []struct {
		name  string
		bytes []byte
		want  bool
	}{
		{"quoted DATE prefix", []byte(`"DATE","DESCRIPTION"`), true},
		{"unquoted DATE prefix", []byte(`DATE,DESCRIPTION`), false},
		{"other content", []byte(`12/07/2020,-46.80`), false},
		{"empty", []byte{}, false},
		{"nil", nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := doWellsFargoHeadsExist(tt.bytes); got != tt.want {
				t.Errorf("doWellsFargoHeadsExist = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_addWellsFargoHeads(t *testing.T) {
	dir := t.TempDir()
	path := writeCSVFile(t, dir, "wf.csv", "12/07/2020,-46.80\n")
	orig, _ := os.ReadFile(path)
	if err := addWellsFargoHeads(path, orig); err != nil {
		t.Fatalf("addWellsFargoHeads = %v, want nil", err)
	}
	got, _ := os.ReadFile(path)
	want := `"date","amount","dummy1","dummy2","name"` + "\n" + "12/07/2020,-46.80\n"
	if string(got) != want {
		t.Errorf("file after addWellsFargoHeads = %q, want %q", got, want)
	}
	if err := addWellsFargoHeads(filepath.Join(dir, "no", "wf.csv"), orig); err == nil {
		t.Error("addWellsFargoHeads(bad path) = nil, want error")
	}
}

const wellsFargoFixture = `"DATE","DESCRIPTION","AMOUNT","CHECK #","STATUS"
"12/07/2020","TARGET","-46.80","","posted"
"12/01/2020","DIRECT DEPOSIT","1500.00","","posted"
"12/15/2020","CHECK # 1234","-250.00","CHECK # 1234","posted"
`

const fidelityFixture = `"Date","Transaction","Name","Memo","Amount"
"2026-09-28","DEBIT","Netflix","memo123","-19.99"
"2026-09-29","DEBIT","PAYMENT - THANK YOU","memo456","-100.00"
`

const chaseFixture = `"Transaction Date","Post Date","Description","Category","Type","Amount"
"09/28/2026","09/29/2026","AMAZON","Shopping","Sale","-20.00"
"09/30/2026","10/01/2026","Payment Thank You","Payment","Payment","500.00"
`

const bankOfAmericaFixture = `"Posted Date","Reference Number","Payee","Address","Amount"
"09/28/2026","REF1","AMAZON","Seattle","19.99"
"09/29/2026","REF2","BA Electronic Payment","Somewhere","500.00"
`

func Test_readWellsFargoCSVRows(t *testing.T) {
	dir := t.TempDir()
	path := writeCSVFile(t, dir, "wf.csv", wellsFargoFixture)
	got, err := readWellsFargoCSVRows(path, models.WellsFargoID)
	if err != nil {
		t.Fatalf("readWellsFargoCSVRows = %v, want nil", err)
	}
	if len(got) != 3 {
		t.Fatalf("got %d transactions, want 3", len(got))
	}
	if got[0].Date != "12/07/20" || got[0].Amount != -46.80 || got[0].Withdrawal != 46.80 || got[0].Key != "wellsfargo:12/07/20:-46.80" {
		t.Errorf("row 0 = %+v, want TARGET withdrawal", got[0])
	}
	if got[1].Deposit != 1500.00 {
		t.Errorf("row 1 Deposit = %v, want 1500.00", got[1].Deposit)
	}
	if !got[2].IsCheck || got[2].Source != "1234" || got[2].Name != models.CheckRegisterName {
		t.Errorf("row 2 = %+v, want check 1234", got[2])
	}
	if _, err := readWellsFargoCSVRows(filepath.Join(dir, "nope.csv"), models.WellsFargoID); err == nil {
		t.Error("readWellsFargoCSVRows(nonexistent) = nil, want error")
	}
}

func Test_readFidelityCSVRows(t *testing.T) {
	dir := t.TempDir()
	path := writeCSVFile(t, dir, "fid.csv", fidelityFixture)
	got, err := readFidelityCSVRows(path, models.FidelityID)
	if err != nil {
		t.Fatalf("readFidelityCSVRows = %v, want nil", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d transactions, want 1 (PAYMENT row skipped)", len(got))
	}
	tx := got[0]
	if tx.TransactionID != "memo123" || tx.Date != "09/28/26" || tx.CreditPurchase != 19.99 || tx.Key != "fidelity:09/28/26:19.99" {
		t.Errorf("tx = %+v, want Netflix charge", tx)
	}

	noHeader := writeCSVFile(t, dir, "noheader.csv", `"Date","Transaction","Name","Amount"`+"\n")
	if _, err := readFidelityCSVRows(noHeader, models.FidelityID); err == nil {
		t.Error("readFidelityCSVRows(missing Memo header) = nil, want error")
	}
	if _, err := readFidelityCSVRows(filepath.Join(dir, "nope.csv"), models.FidelityID); err == nil {
		t.Error("readFidelityCSVRows(nonexistent) = nil, want error")
	}
}

func Test_readChaseCSVRows(t *testing.T) {
	dir := t.TempDir()
	path := writeCSVFile(t, dir, "chase.csv", chaseFixture)
	got, err := readChaseCSVRows(path, models.ChaseID)
	if err != nil {
		t.Fatalf("readChaseCSVRows = %v, want nil", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d transactions, want 1 (Payment row skipped)", len(got))
	}
	tx := got[0]
	if tx.Date != "09/28/26" || tx.Amount != -20.00 || tx.CreditPurchase != 20.00 || tx.Budget != -20.00 || tx.Key != "chase:09/28/26:20.00" {
		t.Errorf("tx = %+v, want AMAZON charge", tx)
	}

	noHeader := writeCSVFile(t, dir, "noheader.csv", `"Transaction Date","Description","Amount"`+"\n")
	if _, err := readChaseCSVRows(noHeader, models.ChaseID); err == nil {
		t.Error("readChaseCSVRows(missing headers) = nil, want error")
	}
}

func Test_readBankOfAmericaCSVRows(t *testing.T) {
	dir := t.TempDir()
	path := writeCSVFile(t, dir, "boa.csv", bankOfAmericaFixture)
	got, err := readBankOfAmericaCSVRows(path, models.BoAID)
	if err != nil {
		t.Fatalf("readBankOfAmericaCSVRows = %v, want nil", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d transactions, want 1 (electronic payment skipped)", len(got))
	}
	tx := got[0]
	if tx.Source != models.BoAID || tx.Date != "09/28/26" || tx.Amount != -19.99 || tx.CreditPurchase != 19.99 || tx.Key != "boa:09/28/26:19.99" {
		t.Errorf("tx = %+v, want AMAZON purchase", tx)
	}

	noHeader := writeCSVFile(t, dir, "noheader.csv", `"Posted Date","Payee","Amount"`+"\n")
	if _, err := readBankOfAmericaCSVRows(noHeader, models.BoAID); err == nil {
		t.Error("readBankOfAmericaCSVRows(missing headers) = nil, want error")
	}
}

func Test_readWellsFargoCSV(t *testing.T) {
	dir := t.TempDir()
	banks := map[string]config.Bank{
		models.WellsFargoID: {ID: models.WellsFargoID, CSVFileName: "wf.csv"},
	}
	c := New(dir, banks, []string{models.WellsFargoID}, func(_ string) {})

	withHeads := writeCSVFile(t, dir, "wf.csv", wellsFargoFixture)
	_ = withHeads
	got, err := c.readWellsFargoCSV()
	if err != nil {
		t.Fatalf("readWellsFargoCSV(with headers) = %v, want nil", err)
	}
	if len(got) != 3 {
		t.Fatalf("got %d transactions, want 3", len(got))
	}

	// Headerless file: the client prepends the legacy header and reads on.
	// gocsv matches headers to struct tags case-sensitively, so the
	// lowercase legacy header leaves the rows unmapped (zero values) —
	// assert the file mutation and the row count, which is what this
	// path actually guarantees.
	legacy := writeCSVFile(t, dir, "wf.csv", "12/07/2020,-46.80,foo,bar,baz\n")
	_ = legacy
	got, err = c.readWellsFargoCSV()
	if err != nil {
		t.Fatalf("readWellsFargoCSV(headerless) = %v, want nil", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d transactions, want 1", len(got))
	}
	contents, _ := os.ReadFile(filepath.Join(dir, "wf.csv"))
	if !strings.HasPrefix(string(contents), `"date","amount","dummy1","dummy2","name"`) {
		t.Errorf("headerless file after read = %q, want legacy header prepended", contents)
	}

	// Missing file: the read error propagates instead of producing an
	// empty list or a header-only file.
	missing := New(t.TempDir(), banks, []string{models.WellsFargoID}, func(_ string) {})
	if _, err := missing.readWellsFargoCSV(); err == nil {
		t.Error("readWellsFargoCSV(missing file) = nil, want error")
	} else if !strings.Contains(err.Error(), "could not read CSV file") {
		t.Errorf("readWellsFargoCSV(missing file) = %v, want wrapped read error", err)
	}
}

func Test_readWellsFargoCSV_MissingHeaders(t *testing.T) {
	dir := t.TempDir()
	banks := map[string]config.Bank{
		models.WellsFargoID: {ID: models.WellsFargoID, CSVFileName: "wf.csv"},
	}
	c := New(dir, banks, []string{models.WellsFargoID}, func(_ string) {})
	// Has the "DATE" prefix so no header is prepended, but is missing
	// the CHECK # and STATUS headers the WellsFargo struct expects.
	writeCSVFile(t, dir, "wf.csv", `"DATE","DESCRIPTION","AMOUNT"`+"\n"+`"12/07/2020","TARGET","-46.80"`+"\n")
	if _, err := c.readWellsFargoCSV(); err == nil {
		t.Error("readWellsFargoCSV(missing headers) = nil, want error")
	} else if !strings.Contains(err.Error(), "CHECK #") || !strings.Contains(err.Error(), "STATUS") {
		t.Errorf("readWellsFargoCSV(missing headers) = %v, want error naming CHECK # and STATUS", err)
	}
}

func Test_readWellsFargoCSV_WriteHeadsError(t *testing.T) {
	dir := t.TempDir()
	banks := map[string]config.Bank{
		models.WellsFargoID: {ID: models.WellsFargoID, CSVFileName: "missing-dir/wf.csv"},
	}
	c := New(dir, banks, []string{models.WellsFargoID}, func(_ string) {})
	// The read succeeds (stubbed headerless bytes) but the prepended
	// header cannot be written: the file lives in a directory that
	// does not exist.
	old := myReadFile
	myReadFile = func(_ string) ([]byte, error) {
		return []byte("12/07/2020,-46.80,foo,bar,baz\n"), nil
	}
	defer func() { myReadFile = old }()
	if _, err := c.readWellsFargoCSV(); err == nil {
		t.Error("readWellsFargoCSV(unwritable file) = nil, want error")
	} else if !strings.Contains(err.Error(), "could not write csv file") {
		t.Errorf("readWellsFargoCSV(unwritable file) = %v, want write error", err)
	}
}

func Test_readCSVRows_UnmarshalError(t *testing.T) {
	tests := []struct {
		name   string
		file   string
		bankID string
		read   func(string, string) ([]*models.Transaction, error)
	}{
		{
			name:   "wells fargo bad amount",
			file:   `"DATE","DESCRIPTION","AMOUNT","CHECK #","STATUS"` + "\n" + `"12/07/2020","TARGET","not-a-number","","posted"` + "\n",
			bankID: models.WellsFargoID,
			read:   readWellsFargoCSVRows,
		},
		{
			name:   "fidelity bad amount",
			file:   `"Date","Transaction","Name","Memo","Amount"` + "\n" + `"2026-09-28","DEBIT","Netflix","memo123","not-a-number"` + "\n",
			bankID: models.FidelityID,
			read:   readFidelityCSVRows,
		},
		{
			name:   "chase bad amount",
			file:   `"Transaction Date","Post Date","Description","Category","Type","Amount"` + "\n" + `"09/28/2026","09/29/2026","AMAZON","Shopping","Sale","not-a-number"` + "\n",
			bankID: models.ChaseID,
			read:   readChaseCSVRows,
		},
		{
			name:   "bank of america bad amount",
			file:   `"Posted Date","Reference Number","Payee","Address","Amount"` + "\n" + `"09/28/2026","REF1","AMAZON","Seattle","not-a-number"` + "\n",
			bankID: models.BoAID,
			read:   readBankOfAmericaCSVRows,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := writeCSVFile(t, dir, "bad.csv", tt.file)
			if _, err := tt.read(path, tt.bankID); err == nil {
				t.Error("read rows(bad amount) = nil, want error")
			} else if !strings.Contains(err.Error(), "could not unmarshal CSV file") {
				t.Errorf("read rows(bad amount) = %v, want unmarshal error", err)
			}
		})
	}
}

func TestClient_GetTransactions_BankErrors(t *testing.T) {
	tests := []struct {
		name   string
		bankID string
		file   string
	}{
		{"wells fargo missing file", models.WellsFargoID, "wf.csv"},
		{"chase missing file", models.ChaseID, "chase.csv"},
		{"bank of america missing file", models.BoAID, "boa.csv"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			banks := map[string]config.Bank{
				tt.bankID: {ID: tt.bankID, CSVFileName: tt.file},
			}
			c := New(dir, banks, []string{tt.bankID}, func(_ string) {})
			if _, err := c.GetTransactions(); err == nil {
				t.Error("GetTransactions(missing file) = nil, want error")
			} else if !strings.Contains(err.Error(), "could not read CSV file") {
				t.Errorf("GetTransactions(missing file) = %v, want wrapped read error", err)
			}
		})
	}
}

func TestClient_GetTransactions_AllBanks(t *testing.T) {
	dir := t.TempDir()
	writeCSVFile(t, dir, "fid.csv", fidelityFixture)
	writeCSVFile(t, dir, "chase.csv", chaseFixture)
	writeCSVFile(t, dir, "wf.csv", wellsFargoFixture)
	writeCSVFile(t, dir, "boa.csv", bankOfAmericaFixture)
	banks := map[string]config.Bank{
		models.FidelityID:   {ID: models.FidelityID, CSVFileName: "fid.csv"},
		models.ChaseID:      {ID: models.ChaseID, CSVFileName: "chase.csv"},
		models.WellsFargoID: {ID: models.WellsFargoID, CSVFileName: "wf.csv"},
		models.BoAID:        {ID: models.BoAID, CSVFileName: "boa.csv"},
	}
	var progress []string
	c := New(dir, banks, []string{models.FidelityID, models.ChaseID, models.WellsFargoID, models.BoAID}, func(s string) {
		progress = append(progress, s)
	})
	got, err := c.GetTransactions()
	if err != nil {
		t.Fatalf("GetTransactions = %v, want nil", err)
	}
	if len(got) != 6 {
		t.Fatalf("got %d transactions, want 6 (1 fidelity + 1 chase + 3 wells fargo + 1 bank of america)", len(got))
	}
	found := false
	for _, p := range progress {
		if strings.Contains(p, "Bank of America") {
			found = true
		}
	}
	if !found {
		t.Errorf("progress %v missing %q", progress, "Bank of America")
	}
}

func Test_readBankCSVClients(t *testing.T) {
	dir := t.TempDir()
	writeCSVFile(t, dir, "fid.csv", fidelityFixture)
	writeCSVFile(t, dir, "chase.csv", chaseFixture)
	writeCSVFile(t, dir, "boa.csv", bankOfAmericaFixture)
	banks := map[string]config.Bank{
		models.FidelityID: {ID: models.FidelityID, CSVFileName: "fid.csv"},
		models.ChaseID:    {ID: models.ChaseID, CSVFileName: "chase.csv"},
		models.BoAID:      {ID: models.BoAID, CSVFileName: "boa.csv"},
	}
	c := New(dir, banks, nil, func(_ string) {})

	if got, err := c.readFidelityCSV(); err != nil || len(got) != 1 {
		t.Errorf("readFidelityCSV = %d transactions, %v; want 1, nil", len(got), err)
	}
	if got, err := c.readChaseCSV(); err != nil || len(got) != 1 {
		t.Errorf("readChaseCSV = %d transactions, %v; want 1, nil", len(got), err)
	}
	if got, err := c.readBankOfAmericaCSV(); err != nil || len(got) != 1 {
		t.Errorf("readBankOfAmericaCSV = %d transactions, %v; want 1, nil", len(got), err)
	}
}

func TestClient_GetTransactions(t *testing.T) {
	dir := t.TempDir()
	writeCSVFile(t, dir, "fid.csv", fidelityFixture)
	writeCSVFile(t, dir, "chase.csv", chaseFixture)
	writeCSVFile(t, dir, "wf.csv", wellsFargoFixture)
	banks := map[string]config.Bank{
		models.FidelityID:   {ID: models.FidelityID, CSVFileName: "fid.csv"},
		models.ChaseID:      {ID: models.ChaseID, CSVFileName: "chase.csv"},
		models.WellsFargoID: {ID: models.WellsFargoID, CSVFileName: "wf.csv"},
	}
	var progress []string
	c := New(dir, banks, []string{models.FidelityID, models.ChaseID, models.WellsFargoID, "unknown"}, func(s string) {
		progress = append(progress, s)
	})
	got, err := c.GetTransactions()
	if err != nil {
		t.Fatalf("GetTransactions = %v, want nil", err)
	}
	if len(got) != 5 {
		t.Fatalf("got %d transactions, want 5 (1 fidelity + 1 chase + 3 wells fargo)", len(got))
	}
	for _, want := range []string{"Fidelity", "Chase", "Wells Fargo"} {
		found := false
		for _, p := range progress {
			if strings.Contains(p, want) {
				found = true
			}
		}
		if !found {
			t.Errorf("progress %v missing %q", progress, want)
		}
	}

	// A bank whose file is missing fails the whole run.
	bad := New(dir, map[string]config.Bank{
		models.FidelityID: {ID: models.FidelityID, CSVFileName: "nope.csv"},
	}, []string{models.FidelityID}, func(_ string) {})
	if _, err := bad.GetTransactions(); err == nil {
		t.Error("GetTransactions(missing file) = nil, want error")
	} else if !strings.Contains(err.Error(), "could not read CSV file") {
		t.Errorf("GetTransactions(missing file) = %v, want wrapped read error", err)
	}

	// No bank IDs means no transactions and no error.
	empty := New(dir, banks, nil, func(_ string) {})
	if got, err := empty.GetTransactions(); err != nil || len(got) != 0 {
		t.Errorf("GetTransactions(nil banks) = %d transactions, %v; want 0, nil", len(got), err)
	}
}
