package sheets_service

import (
	"strings"
	"testing"

	"register/pkg/models"
)

// fixed block (1-10), then green 11-14, yellow 15-17, black 18-19
// (the card payoff columns), blue 20-24 — Taxes is a savings column.
func moveTestColumns() []models.Column {
	mk := func(i int, name, color string) models.Column {
		return models.Column{ID: i, Name: name, Color: color, ColumnIndex: i}
	}
	cols := []models.Column{}
	for i := 1; i <= 10; i++ {
		cols = append(cols, mk(i, "", ""))
	}
	cols = append(cols,
		mk(11, "Cash", models.ColorGreen), mk(12, "Dining", models.ColorGreen),
		mk(13, "Grocery", models.ColorGreen), mk(14, "Misc", models.ColorGreen),
		mk(15, "Rent", models.ColorYellow), mk(16, "Electric", models.ColorYellow),
		mk(17, "ISP", models.ColorYellow),
		mk(18, "Credit Cards", models.ColorBlack), mk(19, "AppleCard", models.ColorBlack),
		mk(20, "Savings", models.ColorBlue), mk(21, "Gifts", models.ColorBlue),
		mk(22, "Medical", models.ColorBlue), mk(23, "Emergency Fund", models.ColorBlue),
		mk(24, "Taxes", models.ColorBlue),
	)
	return cols
}

func TestPlanColumnMove(t *testing.T) {
	cols := moveTestColumns()
	cases := []struct {
		name        string
		from, after int
		wantTo      int
		wantChanged bool
		wantErr     bool
	}{
		{"within group, moving left", 13, 11, 12, true, false},
		{"within group, moving right", 12, 14, 14, true, false},
		{"already after target", 13, 12, 13, false, false},
		{"move self is an error", 13, 13, 0, false, true},
		{"green into yellow refused", 13, 16, 0, false, true},
		{"yellow into green refused", 15, 12, 0, false, true},
		{"fixed block cannot move", 5, 13, 0, false, true},
		{"after a fixed column refused", 13, 9, 0, false, true},
		{"blue moves within savings", 24, 22, 23, true, false},
		{"black leaving the payoff block refused", 19, 0, 0, false, true},
		{"black splitting blue refused", 18, 21, 0, false, true},
		{"category to the end refused (leaves its group)", 13, 0, 0, false, true},
		{"unknown from index", 99, 11, 0, false, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			to, changed, err := PlanColumnMove(cols, tc.from, tc.after)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("PlanColumnMove(%d, %d) = (%d, %v, nil), want error", tc.from, tc.after, to, changed)
				}
				return
			}
			if err != nil {
				t.Fatalf("PlanColumnMove(%d, %d) error: %v", tc.from, tc.after, err)
			}
			if to != tc.wantTo || changed != tc.wantChanged {
				t.Errorf("PlanColumnMove(%d, %d) = (%d, %v), want (%d, %v)",
					tc.from, tc.after, to, changed, tc.wantTo, tc.wantChanged)
			}
		})
	}
}

func checkTestColumns() []models.Column {
	cols := []models.Column{}
	for i := 1; i <= 10; i++ {
		cols = append(cols, models.Column{ID: i, Name: "", ColumnIndex: i})
	}
	cols = append(cols,
		models.Column{ID: 11, Name: "Grocery", Color: models.ColorBlue, ColumnIndex: 11},
		models.Column{ID: 12, Name: "Rent", Color: models.ColorBlue, ColumnIndex: 12},
	)
	return cols
}

func Test_registerService_CheckRegisterColumns(t *testing.T) {
	sheet := RegisterSheet{
		TabName:     RegisterTabName,
		SheetCoords: SheetCoords{EndColumnName: "L", ColumnCount: 12},
	}
	header := func(grocery, rent string) []any {
		row := make([]any, 12)
		row[10] = grocery
		row[11] = rent
		return row
	}
	t.Run("clean", func(t *testing.T) {
		getValuesProviderFunc = func(range_ string) ([][]any, error) {
			if range_ == "Register!A4:L4" {
				return [][]any{header("Grocery", "Rent")}, nil
			}
			t.Fatalf("unexpected range %q", range_)
			return nil, nil
		}
		problems, err := ss.CheckRegisterColumns(sheet, checkTestColumns())
		if err != nil {
			t.Fatalf("CheckRegisterColumns() error = %v, want nil", err)
		}
		if len(problems) != 0 {
			t.Errorf("CheckRegisterColumns() problems = %v, want none", problems)
		}
	})
	t.Run("header drift", func(t *testing.T) {
		getValuesProviderFunc = func(range_ string) ([][]any, error) {
			return [][]any{header("Grocer", "Rent")}, nil
		}
		problems, err := ss.CheckRegisterColumns(sheet, checkTestColumns())
		if err != nil {
			t.Fatalf("CheckRegisterColumns() error = %v, want nil", err)
		}
		if len(problems) != 1 || !strings.Contains(problems[0], "Grocery") {
			t.Errorf("CheckRegisterColumns() problems = %v, want one mentioning Grocery", problems)
		}
	})
	t.Run("index gap", func(t *testing.T) {
		getValuesProviderFunc = func(range_ string) ([][]any, error) {
			return [][]any{header("Grocery", "Rent")}, nil
		}
		cols := checkTestColumns()
		cols = append(cols[:4], cols[5:]...) // drop index 5
		problems, err := ss.CheckRegisterColumns(sheet, cols)
		if err != nil {
			t.Fatalf("CheckRegisterColumns() error = %v, want nil", err)
		}
		found := false
		for _, p := range problems {
			if strings.Contains(p, "no column at index 5") {
				found = true
			}
		}
		if !found {
			t.Errorf("CheckRegisterColumns() problems = %v, want missing index 5", problems)
		}
	})
	t.Run("column count mismatch", func(t *testing.T) {
		getValuesProviderFunc = func(range_ string) ([][]any, error) {
			return [][]any{header("Grocery", "Rent")}, nil
		}
		wide := sheet
		wide.SheetCoords.ColumnCount = 13
		problems, err := ss.CheckRegisterColumns(wide, checkTestColumns())
		if err != nil {
			t.Fatalf("CheckRegisterColumns() error = %v, want nil", err)
		}
		found := false
		for _, p := range problems {
			if strings.Contains(p, "column count mismatch") {
				found = true
			}
		}
		if !found {
			t.Errorf("CheckRegisterColumns() problems = %v, want count mismatch", problems)
		}
	})
	t.Run("provider error", func(t *testing.T) {
		getValuesProviderFunc = func(range_ string) ([][]any, error) {
			return nil, errTestProvider
		}
		if _, err := ss.CheckRegisterColumns(sheet, checkTestColumns()); err == nil {
			t.Errorf("CheckRegisterColumns() error = nil, want error")
		}
	})
}

func Test_registerService_InsertRegisterColumn(t *testing.T) {
	sheet := RegisterSheet{
		ID:          617336355,
		TabName:     RegisterTabName,
		SheetCoords: SheetCoords{RowCount: 5},
	}
	t.Run("first column needs no neighbor read", func(t *testing.T) {
		if err := ss.InsertRegisterColumn(sheet, 1, "NewCol"); err != nil {
			t.Errorf("InsertRegisterColumn() error = %v, want nil", err)
		}
	})
	t.Run("neighbor formulas shifted", func(t *testing.T) {
		getFormulaProviderFunc = func(range_ string) ([][]any, error) {
			if range_ == "Register!B1:B5" {
				return [][]any{{""}, {""}, {""}, {""}, {"=B5"}}, nil
			}
			t.Fatalf("unexpected range %q", range_)
			return nil, nil
		}
		if err := ss.InsertRegisterColumn(sheet, 3, "NewCol"); err != nil {
			t.Errorf("InsertRegisterColumn() error = %v, want nil", err)
		}
	})
	t.Run("formula read error", func(t *testing.T) {
		getFormulaProviderFunc = func(range_ string) ([][]any, error) {
			return nil, errTestProvider
		}
		if err := ss.InsertRegisterColumn(sheet, 3, "NewCol"); err == nil {
			t.Errorf("InsertRegisterColumn() error = nil, want error")
		}
	})
}

func Test_registerService_DeleteRegisterColumn(t *testing.T) {
	sheet := RegisterSheet{ID: 617336355, TabName: RegisterTabName}
	if err := ss.DeleteRegisterColumn(sheet, 12); err != nil {
		t.Errorf("DeleteRegisterColumn() error = %v, want nil", err)
	}
}

func Test_registerService_MoveRegisterColumn(t *testing.T) {
	sheet := RegisterSheet{ID: 617336355, TabName: RegisterTabName}
	if err := ss.MoveRegisterColumn(sheet, 11, 12); err != nil {
		t.Errorf("MoveRegisterColumn() error = %v, want nil", err)
	}
}

func Test_registerService_RenameRegisterColumnHeader(t *testing.T) {
	sheet := RegisterSheet{ID: 617336355, TabName: RegisterTabName}
	if err := ss.RenameRegisterColumnHeader(sheet, 11, "Food"); err != nil {
		t.Errorf("RenameRegisterColumnHeader() error = %v, want nil", err)
	}
}

func Test_registerService_readColumnFormulas(t *testing.T) {
	sheet := RegisterSheet{ID: 617336355, TabName: RegisterTabName}
	t.Run("formulas and non-formulas", func(t *testing.T) {
		getFormulaProviderFunc = func(range_ string) ([][]any, error) {
			if range_ == "Register!A1:A4" {
				return [][]any{{"=A1"}, {"plain"}, {""}, {"=B2"}}, nil
			}
			t.Fatalf("unexpected range %q", range_)
			return nil, nil
		}
		got, err := ss.readColumnFormulas(sheet, "A", 4)
		if err != nil {
			t.Fatalf("readColumnFormulas() error = %v, want nil", err)
		}
		want := []string{"=A1", "", "", "=B2"}
		if len(got) != len(want) {
			t.Fatalf("len(formulas) = %d, want %d", len(got), len(want))
		}
		for i, w := range want {
			if got[i] != w {
				t.Errorf("formulas[%d] = %q, want %q", i, got[i], w)
			}
		}
	})
	t.Run("truncated to row count", func(t *testing.T) {
		getFormulaProviderFunc = func(range_ string) ([][]any, error) {
			return [][]any{{"=A1"}, {"=A2"}, {"=A3"}}, nil
		}
		got, err := ss.readColumnFormulas(sheet, "A", 2)
		if err != nil {
			t.Fatalf("readColumnFormulas() error = %v, want nil", err)
		}
		if len(got) != 2 || got[0] != "=A1" || got[1] != "=A2" {
			t.Errorf("readColumnFormulas() = %v, want [=A1 =A2]", got)
		}
	})
	t.Run("provider error", func(t *testing.T) {
		getFormulaProviderFunc = func(range_ string) ([][]any, error) {
			return nil, errTestProvider
		}
		if _, err := ss.readColumnFormulas(sheet, "A", 4); err == nil {
			t.Errorf("readColumnFormulas() error = nil, want error")
		}
	})
}
