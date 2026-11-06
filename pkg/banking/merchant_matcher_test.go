package banking

import (
	"testing"

	"register/pkg/config"
	"register/pkg/models"
)

func rule(id int, bankName, name string, priority int, matchType string) *models.DataRow {
	return &models.DataRow{ID: id, BankName: bankName, Name: name, Priority: priority, MatchType: matchType}
}

func TestMerchantMatcher_PriorityFirstWins(t *testing.T) {
	m, err := NewMerchantMatcher([]*models.DataRow{
		rule(1, "AMAZON", "Legacy Amazon", 0, ""),
		rule(2, "Amazon.com", "Amazon", 100, "substring"),
	})
	if err != nil {
		t.Fatal(err)
	}
	got, ok := m.Match("AMAZON.COM*AMZN MKTP")
	if !ok || got.Name != "Amazon" {
		t.Errorf("got %v, %v; want the priority-100 rule", got, ok)
	}
}

func TestMerchantMatcher_TieBreakLowestID(t *testing.T) {
	m, err := NewMerchantMatcher([]*models.DataRow{
		rule(9, "AMAZON", "Higher ID", 0, ""),
		rule(3, "AMAZON", "Lower ID", 0, ""),
	})
	if err != nil {
		t.Fatal(err)
	}
	got, ok := m.Match("AMAZON MKTPL")
	if !ok || got.Name != "Lower ID" {
		t.Errorf("got %v, %v; want lowest-ID rule on tie", got, ok)
	}
}

func TestMerchantMatcher_ExactFoldsCase(t *testing.T) {
	m, err := NewMerchantMatcher([]*models.DataRow{
		rule(1, "PAYROLL", "Paycheck", 0, "exact"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := m.Match("NOVA BEER LLC PAYROLL"); ok {
		t.Error("exact rule matched a longer bank name")
	}
	if _, ok := m.Match("payroll"); !ok {
		t.Error("exact rule should fold case")
	}
}

func TestMerchantMatcher_Regex(t *testing.T) {
	m, err := NewMerchantMatcher([]*models.DataRow{
		rule(1, "^AMZN", "Amazon", 0, "regex"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := m.Match("AMZN MKTP US"); !ok {
		t.Error("regex rule should match")
	}
	if _, ok := m.Match("amzn mktp us"); ok {
		t.Error("regex rules are case-sensitive unless the pattern says (?i)")
	}
}

func TestMerchantMatcher_NoMatch(t *testing.T) {
	m, err := NewMerchantMatcher([]*models.DataRow{
		rule(1, "AMAZON", "Amazon", 0, ""),
	})
	if err != nil {
		t.Fatal(err)
	}
	if got, ok := m.Match("LOCAL BAKERY"); ok || got != nil {
		t.Errorf("got %v, %v; want no match", got, ok)
	}
}

func TestMerchantMatcher_FindByBankPattern(t *testing.T) {
	tests := []struct {
		name   string
		rules  []*models.DataRow
		query  string
		wantOK bool
		wantID int // 0 when no match
	}{
		{
			name:   "match on BankName when Pattern empty",
			rules:  []*models.DataRow{rule(1, "AMAZON", "Amazon", 0, "substring")},
			query:  "AMAZON",
			wantOK: true, wantID: 1,
		},
		{
			name:   "match on Pattern when set",
			rules:  []*models.DataRow{{ID: 2, BankName: "AMAZON", Pattern: "AMZN MKTP US", Name: "Amazon", MatchType: "substring"}},
			query:  "AMZN MKTP US",
			wantOK: true, wantID: 2,
		},
		{
			name:   "Pattern replaces BankName as the match key",
			rules:  []*models.DataRow{{ID: 2, BankName: "AMAZON", Pattern: "AMZN MKTP US", Name: "Amazon", MatchType: "substring"}},
			query:  "AMAZON",
			wantOK: false,
		},
		{
			name:   "case folds query lower",
			rules:  []*models.DataRow{rule(1, "AMAZON", "Amazon", 0, "substring")},
			query:  "amazon",
			wantOK: true, wantID: 1,
		},
		{
			name:   "case folds rule lower",
			rules:  []*models.DataRow{rule(1, "amazon", "Amazon", 0, "substring")},
			query:  "AMAZON",
			wantOK: true, wantID: 1,
		},
		{
			name:   "case folds Pattern",
			rules:  []*models.DataRow{{ID: 2, BankName: "AMAZON", Pattern: "Amzn Mktp", Name: "Amazon", MatchType: "substring"}},
			query:  "AMZN MKTP",
			wantOK: true, wantID: 2,
		},
		{
			name:   "no match",
			rules:  []*models.DataRow{rule(1, "AMAZON", "Amazon", 0, "substring")},
			query:  "TARGET",
			wantOK: false,
		},
		{
			name:   "substring of the pattern is not a match",
			rules:  []*models.DataRow{rule(1, "AMAZON", "Amazon", 0, "substring")},
			query:  "AMAZ",
			wantOK: false,
		},
		{
			name:   "pattern as substring of the query is not a match",
			rules:  []*models.DataRow{rule(1, "AMAZON", "Amazon", 0, "substring")},
			query:  "AMAZON.COM*AMZN MKTP",
			wantOK: false,
		},
		{
			name:   "regex rule found by its pattern string",
			rules:  []*models.DataRow{rule(1, "^AMZN", "Amazon", 0, "regex")},
			query:  "^AMZN",
			wantOK: true, wantID: 1,
		},
		{
			name:   "regex rule not found by a string it would match",
			rules:  []*models.DataRow{rule(1, "^AMZN", "Amazon", 0, "regex")},
			query:  "AMZN MKTP US",
			wantOK: false,
		},
		{
			name:   "empty matcher",
			rules:  nil,
			query:  "AMAZON",
			wantOK: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := NewMerchantMatcher(tt.rules)
			if err != nil {
				t.Fatal(err)
			}
			got, ok := m.FindByBankPattern(tt.query)
			if ok != tt.wantOK {
				t.Fatalf("FindByBankPattern(%q) ok = %v, want %v", tt.query, ok, tt.wantOK)
			}
			if !tt.wantOK {
				if got != nil {
					t.Errorf("FindByBankPattern(%q) = %v, want nil", tt.query, got)
				}
				return
			}
			if got == nil || got.ID != tt.wantID {
				t.Errorf("FindByBankPattern(%q) = %v, want rule id %d", tt.query, got, tt.wantID)
			}
		})
	}
}

// FindByBankPattern returns a rule whether or not it would win a Match:
// the broad high-priority rule wins the match, but the narrow rule is
// still found by its own pattern.
func TestMerchantMatcher_FindByBankPattern_FindsNonWinningRule(t *testing.T) {
	m, err := NewMerchantMatcher([]*models.DataRow{
		rule(1, "AMAZON.COM", "Exact Amazon", 0, "substring"),
		rule(2, "AMAZON", "Broad Amazon", 100, "substring"),
	})
	if err != nil {
		t.Fatal(err)
	}
	got, ok := m.Match("AMAZON.COM*123")
	if !ok || got.Name != "Broad Amazon" {
		t.Fatalf("Match = %v, %v; want the priority-100 rule", got, ok)
	}
	got, ok = m.FindByBankPattern("AMAZON.COM")
	if !ok || got.Name != "Exact Amazon" {
		t.Errorf("FindByBankPattern = %v, %v; want the low-priority rule", got, ok)
	}
}

func TestNewMerchantMatcher_BadRegex(t *testing.T) {
	if _, err := NewMerchantMatcher([]*models.DataRow{
		rule(1, "GLO FIBER (", "Broken", 0, "regex"),
	}); err == nil {
		t.Error("expected an error for a bad regex pattern")
	}
}

func TestFormatMerchantNames_AppliesRules(t *testing.T) {
	matcher, err := NewMerchantMatcher([]*models.DataRow{
		{ID: 1, BankName: "AMAZON", Name: "Amazon", ColumnIndex: 11, Color: models.ColorBlack, IsCategory: true, TaxDeductible: true},
		{ID: 2, BankName: models.CheckTransactionName, Name: models.CheckRegisterName, ColumnIndex: 10, Color: models.ColorWhite},
	})
	if err != nil {
		t.Fatal(err)
	}
	columns := []models.Column{{Name: "Emergency Fund", ColumnIndex: 42}}

	c := &Client{IncomeSources: []config.IncomeSource{
		{Name: "50/50 Taphouse Paycheck", MatchPattern: "NOVA BEER LLC", MatchType: "substring", Frequency: "weekly"},
	}}
	income, err := c.IncomeMatcher()
	if err != nil {
		t.Fatal(err)
	}

	trans := []*models.Transaction{
		{BankName: "AMAZON.COM*AMZN MKTP"},
		{Name: models.CheckTransactionName, BankName: "CHECK #1042"},
		{BankName: "NOVA BEER LLC PAYROLL"},
		{BankName: "LOCAL BAKERY"},
	}
	got := FormatMerchantNames(trans, matcher, income, columns)

	if got[0].Name != "Amazon" || got[0].ColumnIndex != 11 || !got[0].IsCategory || !got[0].TaxDeductible {
		t.Errorf("matcher rule not applied: %+v", got[0])
	}
	if got[1].ColumnIndex != 10 || got[1].Color != models.ColorWhite {
		t.Errorf("CHECK special not applied: %+v", got[1])
	}
	if got[2].Name != "50/50 Taphouse Paycheck" || got[2].ColumnIndex != 42 {
		t.Errorf("income source not applied: %+v", got[2])
	}
	if got[3].Name != "" {
		t.Errorf("unmatched transaction renamed: %+v", got[3])
	}
}

// The CHECK special in FormatMerchantNames only fires for transactions
// already named models.CheckRegisterName ("Check"), and its column comes
// from the rule whose bank pattern is "Check" — not from the substring
// matcher, which would rename the transaction instead.
func TestFormatMerchantNames_CheckSpecial(t *testing.T) {
	income, err := NewMerchantMatcher(nil)
	if err != nil {
		t.Fatal(err)
	}
	columns := []models.Column{{Name: "Emergency Fund", ColumnIndex: 42}}

	t.Run("check rule sets the column", func(t *testing.T) {
		matcher, err := NewMerchantMatcher([]*models.DataRow{
			{ID: 1, BankName: models.CheckRegisterName, Name: models.CheckRegisterName, ColumnIndex: 10, Color: models.ColorWhite},
			{ID: 2, BankName: "CHECK", Name: "Checky", ColumnIndex: 99, MatchType: "substring"},
		})
		if err != nil {
			t.Fatal(err)
		}
		trans := []*models.Transaction{
			{Name: models.CheckRegisterName, BankName: "CHECK #1042", IsCategory: true, TaxDeductible: true},
		}
		got := FormatMerchantNames(trans, matcher, income, columns)
		if got[0].Name != models.CheckRegisterName {
			t.Errorf("Name = %q, want the check branch to leave it alone", got[0].Name)
		}
		if got[0].Color != models.ColorWhite {
			t.Errorf("Color = %q, want white", got[0].Color)
		}
		if got[0].ColumnIndex != 10 {
			t.Errorf("ColumnIndex = %d, want 10 from the Check rule", got[0].ColumnIndex)
		}
		if got[0].IsCategory || got[0].TaxDeductible {
			t.Errorf("IsCategory = %v, TaxDeductible = %v; want both false", got[0].IsCategory, got[0].TaxDeductible)
		}
	})

	t.Run("no check rule leaves the column alone", func(t *testing.T) {
		matcher, err := NewMerchantMatcher([]*models.DataRow{
			{ID: 1, BankName: "AMAZON", Name: "Amazon", ColumnIndex: 11, MatchType: "substring"},
		})
		if err != nil {
			t.Fatal(err)
		}
		trans := []*models.Transaction{
			{Name: models.CheckRegisterName, BankName: "CHECK #1042", IsCategory: true, TaxDeductible: true},
		}
		got := FormatMerchantNames(trans, matcher, income, columns)
		if got[0].ColumnIndex != 0 {
			t.Errorf("ColumnIndex = %d, want 0 with no Check rule", got[0].ColumnIndex)
		}
		if got[0].Color != models.ColorWhite || got[0].IsCategory || got[0].TaxDeductible {
			t.Errorf("check flags = %+v, want white / not category / not deductible", got[0])
		}
	})
}
