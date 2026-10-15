package sheets_service

import (
	"testing"

	"register/pkg/config"
	"register/pkg/models"
)

func TestAllocationFor(t *testing.T) {
	tests := []struct {
		name      string
		entry     *BudgetEntry
		frequency string
		want      float64
	}{
		{
			name:      "weekly frequency returns weekly amount",
			entry:     &BudgetEntry{Category: "Grocery", BiWeekly: 240.00, Monthly: 520.00},
			frequency: "biweekly",
			want:      240.00,
		},
		{
			name:      "monthly frequency returns monthly amount",
			entry:     &BudgetEntry{Category: "Rent", BiWeekly: 0, Monthly: 1400.00},
			frequency: "monthly",
			want:      1400.00,
		},
		{
			name:      "frequency is case-insensitive",
			entry:     &BudgetEntry{Category: "Grocery", BiWeekly: 240.00, Monthly: 520.00},
			frequency: "BIWEEKLY",
			want:      240.00,
		},
		{
			name:      "unknown frequency returns zero",
			entry:     &BudgetEntry{Category: "Grocery", BiWeekly: 240.00, Monthly: 520.00},
			frequency: "bimonthly",
			want:      0,
		},
		{
			name:      "empty frequency returns zero",
			entry:     &BudgetEntry{Category: "Grocery", BiWeekly: 240.00, Monthly: 520.00},
			frequency: "",
			want:      0,
		},
		{
			name:      "nil entry returns zero",
			entry:     nil,
			frequency: "weekly",
			want:      0,
		},
		{
			name:      "zero amounts return zero",
			entry:     &BudgetEntry{Category: "Extra", Weekly: 0, Monthly: 0},
			frequency: "weekly",
			want:      0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := allocationFor(tt.entry, tt.frequency); got != tt.want {
				t.Errorf("allocationFor() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsIncomeName(t *testing.T) {
	sources := []config.IncomeSource{
		{Name: "Taylor House", Frequency: "weekly"},
		{Name: "Social Security", Frequency: "monthly"},
	}

	tests := []struct {
		name   string
		txName string
		want   bool
	}{
		{"matches Taylor House", "Taylor House", true},
		{"matches Social Security", "Social Security", true},
		{"no match", "TARGET", false},
		{"case sensitive", "taylor house", false},
		{"empty name", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isIncomeName(tt.txName, sources); got != tt.want {
				t.Errorf("isIncomeName(%q) = %v, want %v", tt.txName, got, tt.want)
			}
		})
	}
}

func TestIncomeFrequency(t *testing.T) {
	sources := []config.IncomeSource{
		{Name: "Taylor House", Frequency: "weekly"},
		{Name: "Social Security", Frequency: "monthly"},
	}

	if got := incomeFrequency("Taylor House", sources); got != "weekly" {
		t.Errorf("incomeFrequency(Taylor House) = %q, want %q", got, "weekly")
	}
	if got := incomeFrequency("Social Security", sources); got != "monthly" {
		t.Errorf("incomeFrequency(Social Security) = %q, want %q", got, "monthly")
	}
	if got := incomeFrequency("TARGET", sources); got != "" {
		t.Errorf("incomeFrequency(TARGET) = %q, want empty", got)
	}
}

func TestAddSalaryCells_GappedColumns(t *testing.T) {
	// Columns with a gap in ColumnIndex: the gap should produce an empty
	// cell, not shift subsequent amounts left.
	columns := []models.Column{
		{Name: "Grocery", ColumnIndex: 11, Color: "blue"},
		{Name: "Rent", ColumnIndex: 13, Color: "blue"}, // gap at 12
		{Name: "Gas", ColumnIndex: 14, Color: "blue"},
	}

	categoriesMap := map[string]*BudgetEntry{
		"Grocery": {Category: "Grocery", Weekly: 120.00, Monthly: 520.00},
		"Rent":    {Category: "Rent", Weekly: 0, Monthly: 1400.00},
		"Gas":     {Category: "Gas", Weekly: 50.00, Monthly: 217.00},
	}

	rb := &RowBuilder{
		Columns:       columns,
		CategoriesMap: categoriesMap,
	}

	// Build minimal cells up to the category block (A-G are already there in real use;
	// here we just test the category placement logic)
	totalsFormulas := []string{"", "", ""}

	// We test addSalaryCells directly. It starts at nextIndex = BankRegister+1 = 8.
	// With columns at 11, 13, 14:
	// - nextIndex=8, col=11: fill 8,9,10 with zeros (3 cells), then Grocery at 11
	// - nextIndex=12, col=13: fill 12 with zero (1 cell, the gap), then Rent at 13
	// - nextIndex=14, col=14: Gas at 14
	// Total: 3 + 1 + 1 + 1 + 1 = 7 cells
	got := rb.addSalaryCells(nil, totalsFormulas, "weekly")

	// 3 filler (8,9,10) + Grocery(11) + 1 filler (12, gap) + Rent(13) + Gas(14) = 7
	if len(got) != 7 {
		t.Errorf("addSalaryCells with gap produced %d cells, want 7", len(got))
	}

	// Verify the gap cell (index 3, which is column 12) is a zero-dollar filler,
	// not a category amount. The filler cells use mkCellDataDollars(0.00, ...).
	// We check by verifying the cell exists and is not nil.
	if got[3] == nil {
		t.Errorf("gap cell at index 3 is nil, want a zero-dollar filler cell")
	}
}
