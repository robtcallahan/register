package sheets_service

import (
	"testing"

	"register/pkg/models"
)

func Test_compareColumnHeaders(t *testing.T) {
	columns := []models.Column{
		{Name: "BankRegister", ColumnIndex: 8}, // fixed block: never compared
		{Name: "Cash", ColumnIndex: 11},
		{Name: "Dining Out", ColumnIndex: 12},
	}
	headerRow := []interface{}{"", "", "", "", "", "", "", " Register ", "", "", " Cash ", "Groceries"}
	if problems := compareColumnHeaders(columns, headerRow); len(problems) != 1 {
		t.Errorf("compareColumnHeaders() = %v; want exactly the Dining Out mismatch", problems)
	}
	if problems := compareColumnHeaders(columns[:2], headerRow); len(problems) != 0 {
		t.Errorf("compareColumnHeaders() = %v; want no problems", problems)
	}
}

func Test_shiftFormulaColumn(t *testing.T) {
	tests := []struct {
		formula string
		from    string
		to      string
		want    string
	}{
		{"=SUM(K5:K6)", "K", "L", "=SUM(L5:L6)"},
		{"=SUM(AA10:AA11)", "AA", "AB", "=SUM(AB10:AB11)"},
		{"=$K$5+K7", "K", "L", "=$L$5+L7"},
		{"=SUM(K5:K6)+Z2", "K", "L", "=SUM(L5:L6)+Z2"},
		{"Cash", "K", "L", "Cash"},
	}
	for _, tt := range tests {
		if got := shiftFormulaColumn(tt.formula, tt.from, tt.to); got != tt.want {
			t.Errorf("shiftFormulaColumn(%q, %q, %q) = %q; want %q", tt.formula, tt.from, tt.to, got, tt.want)
		}
	}
}
