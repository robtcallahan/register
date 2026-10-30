package sheets_service

import (
	"reflect"
	"strings"
	"testing"

	"register/pkg/config"
	"register/pkg/models"
)

func Test_budgetService_ReadBudgetSheet(t *testing.T) {
	// 10.10: Rewritten to test 9.3 header validation without fixtures.
	// The original compared against a stale JSON golden.
	getValuesProviderFunc = func(range_ string) ([][]any, error) {
		var values [][]any
		if range_ == "Budget!A1:J1" {
			values = [][]any{{"", "Weekly", "Weekly (view only)", "Bi-Weekly", "Monthly", "Monthly (view only)", "Yearly", "Current Total", "Needed - Monthly"}}
			return values, nil
		}
		// Return minimal budget data
		values = [][]any{
			{"", "Weekly", "Weekly (view only)", "Bi-Weekly", "Monthly", "Monthly (view only)", "Yearly", "Current Total", "Needed - Monthly"},
			{"Grocery", "120", "240", "520", "520", "6240", "0", "0"},
		}
		return values, nil
	}

	got, err := bs.ReadBudgetSheet(budgetSheet)
	if err != nil {
		t.Fatalf("ReadBudgetSheet() error = %v", err)
	}
	if got == nil {
		t.Fatal("ReadBudgetSheet() returned nil")
	}
	// Verify the budget entries were parsed
	if len(got.CategoriesMap) == 0 {
		t.Errorf("ReadBudgetSheet() returned empty CategoriesMap")
	}
}

func Test_budgetService_BudgetRows(t *testing.T) {
	getValuesProviderFunc = func(rng string) ([][]any, error) {
		var ret = make([][]any, 0, 3)
		ret = [][]any{{"Cat 1", "$22.50"}, {"Cat 2", "10.00"}, {"Cat 3", "100.00"}}
		return ret, nil
	}

	var want = map[string]int{
		"Cat 1": 3,
		"Cat 2": 4,
		"Cat 3": 5,
	}

	tests := []struct {
		name string
		want map[string]int
	}{
		{
			name: "Test 2 digit year",
			want: want,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := bs.BudgetRows(*budgetSheet)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("BudgetRows = %+v, want %+v", got, tt.want)
			}
		})
	}

}

func Test_budgetService_NewBudgetSheet(t *testing.T) {
	cfg := config.Config{BudgetStartRow: 3, BudgetEndRow: 63}
	tests := []struct {
		name      string
		headers   [][]any
		wantIndex int64
		wantName  string
		wantErr   bool
	}{
		{
			name:      "headers with trailing empties",
			headers:   [][]any{{"Category", "Weekly", "Monthly", "", ""}},
			wantIndex: 2,
			wantName:  "B",
		},
		{
			name:      "all headers present",
			headers:   [][]any{{"", "Weekly", "Monthly", "Yearly", "Current Total", "Needed - Monthly"}},
			wantIndex: 5,
			wantName:  "E",
		},
		{
			name:    "no values",
			headers: [][]any{},
			wantErr: true,
		},
		{
			name:    "empty first row",
			headers: [][]any{{}},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			getValuesProviderFunc = func(range_ string) ([][]any, error) {
				if range_ == "Budget!A1:Z1" {
					return tt.headers, nil
				}
				t.Fatalf("unexpected range %q", range_)
				return nil, nil
			}
			got, err := bs.NewBudgetSheet(cfg)
			if tt.wantErr {
				if err == nil {
					t.Errorf("NewBudgetSheet() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("NewBudgetSheet() error = %v, want nil", err)
			}
			if got.TabName != BudgetTabName {
				t.Errorf("TabName = %q, want %q", got.TabName, BudgetTabName)
			}
			if got.ID != 987654321 {
				t.Errorf("ID = %d, want 987654321", got.ID)
			}
			if got.SheetCoords.StartRow != 3 || got.SheetCoords.EndRow != 63 {
				t.Errorf("rows = %d..%d, want 3..63", got.SheetCoords.StartRow, got.SheetCoords.EndRow)
			}
			if got.SheetCoords.EndColumnIndex != tt.wantIndex {
				t.Errorf("EndColumnIndex = %d, want %d", got.SheetCoords.EndColumnIndex, tt.wantIndex)
			}
			if got.SheetCoords.EndColumnName != tt.wantName {
				t.Errorf("EndColumnName = %q, want %q", got.SheetCoords.EndColumnName, tt.wantName)
			}
		})
	}
}

func Test_budgetService_CheckBudgetLayout(t *testing.T) {
	sheet := BudgetSheet{
		TabName:     BudgetTabName,
		SheetCoords: SheetCoords{EndColumnName: "J"},
	}
	tests := []struct {
		name    string
		headers []any
		wantOk  []bool
		wantErr bool
	}{
		{
			name:    "all required headers present",
			headers: []any{"", "Weekly", "Bi-Weekly", "Monthly"},
			wantOk:  []bool{true, true},
		},
		{
			name:    "monthly missing",
			headers: []any{"", "Weekly", "Bi-Weekly"},
			wantOk:  []bool{true, false},
		},
		{
			name:    "both missing",
			headers: []any{"", "Weekly"},
			wantOk:  []bool{false, false},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			getValuesProviderFunc = func(range_ string) ([][]any, error) {
				if range_ == "Budget!A1:J1" {
					return [][]any{tt.headers}, nil
				}
				t.Fatalf("unexpected range %q", range_)
				return nil, nil
			}
			got, err := bs.CheckBudgetLayout(sheet)
			if err != nil {
				t.Fatalf("CheckBudgetLayout() error = %v, want nil", err)
			}
			if len(got) != 2 {
				t.Fatalf("len(results) = %d, want 2", len(got))
			}
			for i, want := range tt.wantOk {
				if got[i].Ok != want {
					t.Errorf("result[%d].Ok = %v, want %v (message %q)", i, got[i].Ok, want, got[i].Message)
				}
			}
		})
	}
	t.Run("header read error", func(t *testing.T) {
		getValuesProviderFunc = func(range_ string) ([][]any, error) {
			return nil, errTestProvider
		}
		if _, err := bs.CheckBudgetLayout(sheet); err == nil {
			t.Errorf("CheckBudgetLayout() error = nil, want error")
		}
	})
}

func Test_budgetService_BudgetRowFor(t *testing.T) {
	sheet := BudgetSheet{
		TabName:     BudgetTabName,
		SheetCoords: SheetCoords{StartRow: 3, EndRow: 63},
	}
	columnA := [][]any{{"Grocery"}, {"Rent"}, {"Utilities"}}
	tests := []struct {
		name     string
		category string
		wantRow  int
		wantOk   bool
	}{
		{"first row", "Grocery", 3, true},
		{"middle row", "Rent", 4, true},
		{"last row", "Utilities", 5, true},
		{"missing category", "Phone", 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			getValuesProviderFunc = func(range_ string) ([][]any, error) {
				return columnA, nil
			}
			row, ok, err := bs.BudgetRowFor(sheet, tt.category)
			if err != nil {
				t.Fatalf("BudgetRowFor() error = %v, want nil", err)
			}
			if row != tt.wantRow || ok != tt.wantOk {
				t.Errorf("BudgetRowFor(%q) = (%d, %v), want (%d, %v)", tt.category, row, ok, tt.wantRow, tt.wantOk)
			}
		})
	}
	t.Run("provider error", func(t *testing.T) {
		getValuesProviderFunc = func(range_ string) ([][]any, error) {
			return nil, errTestProvider
		}
		if _, _, err := bs.BudgetRowFor(sheet, "Grocery"); err == nil {
			t.Errorf("BudgetRowFor() error = nil, want error")
		}
	})
}

func Test_budgetService_BudgetEntryFor(t *testing.T) {
	sheet := BudgetSheet{
		TabName:     BudgetTabName,
		SheetCoords: SheetCoords{StartRow: 3, EndRow: 63, EndColumnName: "J"},
	}
	header := []any{"", "Weekly", "Bi-Weekly", "Monthly"}
	setStubs := func(columnA, row [][]any) {
		getValuesProviderFunc = func(range_ string) ([][]any, error) {
			switch range_ {
			case "Budget!A3:A63":
				return columnA, nil
			case "Budget!A3:J3":
				return row, nil
			case "Budget!A1:J1":
				return [][]any{header}, nil
			}
			t.Fatalf("unexpected range %q", range_)
			return nil, nil
		}
	}
	t.Run("found", func(t *testing.T) {
		setStubs([][]any{{"Grocery"}, {"Rent"}}, [][]any{{"Grocery", "120", "240", "520"}})
		got, ok, err := bs.BudgetEntryFor(sheet, "Grocery")
		if err != nil {
			t.Fatalf("BudgetEntryFor() error = %v, want nil", err)
		}
		if !ok || got == nil {
			t.Fatalf("BudgetEntryFor() = (nil, %v), want entry", ok)
		}
		if got.Category != "Grocery" || got.Weekly != 120 || got.BiWeekly != 240 || got.Monthly != 520 {
			t.Errorf("BudgetEntryFor() = %+v, want Grocery 120/240/520", got)
		}
	})
	t.Run("not found", func(t *testing.T) {
		setStubs([][]any{{"Rent"}}, nil)
		got, ok, err := bs.BudgetEntryFor(sheet, "Grocery")
		if err != nil {
			t.Fatalf("BudgetEntryFor() error = %v, want nil", err)
		}
		if ok || got != nil {
			t.Errorf("BudgetEntryFor() = (%v, %v), want (nil, false)", got, ok)
		}
	})
	t.Run("empty row values", func(t *testing.T) {
		setStubs([][]any{{"Grocery"}}, [][]any{})
		got, ok, err := bs.BudgetEntryFor(sheet, "Grocery")
		if err != nil {
			t.Fatalf("BudgetEntryFor() error = %v, want nil", err)
		}
		if ok || got != nil {
			t.Errorf("BudgetEntryFor() = (%v, %v), want (nil, false)", got, ok)
		}
	})
	t.Run("provider error", func(t *testing.T) {
		getValuesProviderFunc = func(range_ string) ([][]any, error) {
			return nil, errTestProvider
		}
		if _, _, err := bs.BudgetEntryFor(sheet, "Grocery"); err == nil {
			t.Errorf("BudgetEntryFor() error = nil, want error")
		}
	})
}

func Test_budgetService_RenameBudgetCategory(t *testing.T) {
	sheet := BudgetSheet{
		TabName:     BudgetTabName,
		SheetCoords: SheetCoords{StartRow: 3, EndRow: 63},
	}
	t.Run("found", func(t *testing.T) {
		getValuesProviderFunc = func(range_ string) ([][]any, error) {
			return [][]any{{"Grocery"}, {"Rent"}}, nil
		}
		ok, err := bs.RenameBudgetCategory(sheet, "Grocery", "Food")
		if err != nil {
			t.Fatalf("RenameBudgetCategory() error = %v, want nil", err)
		}
		if !ok {
			t.Errorf("RenameBudgetCategory() = false, want true")
		}
	})
	t.Run("not found", func(t *testing.T) {
		getValuesProviderFunc = func(range_ string) ([][]any, error) {
			return [][]any{{"Rent"}}, nil
		}
		ok, err := bs.RenameBudgetCategory(sheet, "Grocery", "Food")
		if err != nil {
			t.Fatalf("RenameBudgetCategory() error = %v, want nil", err)
		}
		if ok {
			t.Errorf("RenameBudgetCategory() = true, want false")
		}
	})
	t.Run("provider error", func(t *testing.T) {
		getValuesProviderFunc = func(range_ string) ([][]any, error) {
			return nil, errTestProvider
		}
		if _, err := bs.RenameBudgetCategory(sheet, "Grocery", "Food"); err == nil {
			t.Errorf("RenameBudgetCategory() error = nil, want error")
		}
	})
}

func Test_budgetService_InsertBudgetCategory(t *testing.T) {
	sheet := BudgetSheet{
		ID:          987654321,
		TabName:     BudgetTabName,
		SheetCoords: SheetCoords{StartRow: 3, EndRow: 63},
	}
	t.Run("unknown color", func(t *testing.T) {
		if err := bs.InsertBudgetCategory(sheet, "NewCat", "red"); err == nil {
			t.Errorf("InsertBudgetCategory() error = nil, want error")
		}
	})
	t.Run("missing totals row", func(t *testing.T) {
		getValuesProviderFunc = func(range_ string) ([][]any, error) {
			return [][]any{{"Grocery"}}, nil
		}
		err := bs.InsertBudgetCategory(sheet, "NewCat", models.ColorGreen)
		if err == nil || !strings.Contains(err.Error(), "Discretionary") {
			t.Errorf("InsertBudgetCategory() error = %v, want missing totals row error", err)
		}
	})
	t.Run("happy path", func(t *testing.T) {
		getValuesProviderFunc = func(range_ string) ([][]any, error) {
			return [][]any{{"Discretionary"}, {"Grocery"}}, nil
		}
		getFormulaProviderFunc = func(range_ string) ([][]any, error) {
			return [][]any{{"=SUM(A1:A2)"}}, nil
		}
		if err := bs.InsertBudgetCategory(sheet, "NewCat", models.ColorGreen); err != nil {
			t.Errorf("InsertBudgetCategory() error = %v, want nil", err)
		}
	})
}

func Test_budgetService_DeleteBudgetCategory(t *testing.T) {
	sheet := BudgetSheet{
		TabName:     BudgetTabName,
		SheetCoords: SheetCoords{StartRow: 3, EndRow: 63},
	}
	t.Run("found", func(t *testing.T) {
		getValuesProviderFunc = func(range_ string) ([][]any, error) {
			return [][]any{{"Grocery"}, {"Rent"}}, nil
		}
		ok, err := bs.DeleteBudgetCategory(sheet, "Rent")
		if err != nil {
			t.Fatalf("DeleteBudgetCategory() error = %v, want nil", err)
		}
		if !ok {
			t.Errorf("DeleteBudgetCategory() = false, want true")
		}
	})
	t.Run("not found", func(t *testing.T) {
		getValuesProviderFunc = func(range_ string) ([][]any, error) {
			return [][]any{{"Grocery"}}, nil
		}
		ok, err := bs.DeleteBudgetCategory(sheet, "Rent")
		if err != nil {
			t.Fatalf("DeleteBudgetCategory() error = %v, want nil", err)
		}
		if ok {
			t.Errorf("DeleteBudgetCategory() = true, want false")
		}
	})
}

func Test_budgetService_MoveBudgetCategoryAfter(t *testing.T) {
	sheet := BudgetSheet{
		TabName:     BudgetTabName,
		SheetCoords: SheetCoords{StartRow: 3, EndRow: 63},
	}
	columnA := [][]any{{"Discretionary"}, {"Grocery"}, {"Rent"}}
	tests := []struct {
		name      string
		category  string
		afterName string
		wantErr   bool
	}{
		{"move", "Grocery", "Rent", false},
		{"already there", "Rent", "Grocery", false},
		{"missing category", "Phone", "Grocery", true},
		{"missing after", "Grocery", "Phone", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			getValuesProviderFunc = func(range_ string) ([][]any, error) {
				return columnA, nil
			}
			err := bs.MoveBudgetCategoryAfter(sheet, tt.category, tt.afterName)
			if tt.wantErr && err == nil {
				t.Errorf("MoveBudgetCategoryAfter() error = nil, want error")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("MoveBudgetCategoryAfter() error = %v, want nil", err)
			}
		})
	}
}

func Test_budgetService_MoveBudgetCategoryBefore(t *testing.T) {
	sheet := BudgetSheet{
		TabName:     BudgetTabName,
		SheetCoords: SheetCoords{StartRow: 3, EndRow: 63},
	}
	columnA := [][]any{{"Discretionary"}, {"Grocery"}, {"Rent"}}
	tests := []struct {
		name       string
		category   string
		beforeName string
		wantErr    bool
	}{
		{"move", "Rent", "Grocery", false},
		{"already there", "Grocery", "Rent", false},
		{"missing category", "Phone", "Grocery", true},
		{"missing before", "Grocery", "Phone", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			getValuesProviderFunc = func(range_ string) ([][]any, error) {
				return columnA, nil
			}
			err := bs.MoveBudgetCategoryBefore(sheet, tt.category, tt.beforeName)
			if tt.wantErr && err == nil {
				t.Errorf("MoveBudgetCategoryBefore() error = nil, want error")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("MoveBudgetCategoryBefore() error = %v, want nil", err)
			}
		})
	}
}

func Test_budgetService_moveBudgetRow(t *testing.T) {
	sheet := BudgetSheet{
		ID:      987654321,
		TabName: BudgetTabName,
	}
	if err := bs.moveBudgetRow(sheet, 4, 2); err != nil {
		t.Errorf("moveBudgetRow() error = %v, want nil", err)
	}
}
