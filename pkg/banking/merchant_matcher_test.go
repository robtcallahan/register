package banking

import (
	"testing"

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

func TestNewMerchantMatcher_BadRegex(t *testing.T) {
	if _, err := NewMerchantMatcher([]*models.DataRow{
		rule(1, "GLO FIBER (", "Broken", 0, "regex"),
	}); err == nil {
		t.Error("expected an error for a bad regex pattern")
	}
}

func TestFormatMerchantNames_AppliesRules(t *testing.T) {
	matcher, err := NewMerchantMatcher([]*models.DataRow{
		{ID: 1, BankName: "AMAZON", Name: "Amazon", ColumnIndex: 11, Color: "black", IsCategory: true, TaxDeductible: true},
		{ID: 2, BankName: "CHECK", Name: "Check", ColumnIndex: 10, Color: "white"},
	})
	if err != nil {
		t.Fatal(err)
	}
	columns := []models.Column{{Name: "Salary", ColumnIndex: 42}}

	trans := []*models.Transaction{
		{BankName: "AMAZON.COM*AMZN MKTP"},
		{Name: "CHECK", BankName: "CHECK #1042"},
		{BankName: "NOVA BEER LLC PAYROLL"},
		{BankName: "LOCAL BAKERY"},
	}
	got := (&Client{}).FormatMerchantNames(trans, matcher, columns)

	if got[0].Name != "Amazon" || got[0].ColumnIndex != 11 || !got[0].IsCategory || !got[0].TaxDeductible {
		t.Errorf("matcher rule not applied: %+v", got[0])
	}
	if got[1].ColumnIndex != 10 || got[1].Color != "white" {
		t.Errorf("CHECK special not applied: %+v", got[1])
	}
	if got[2].Name != PayCheckName || got[2].ColumnIndex != 42 {
		t.Errorf("paycheck special not applied: %+v", got[2])
	}
	if got[3].Name != "" {
		t.Errorf("unmatched transaction renamed: %+v", got[3])
	}
}
