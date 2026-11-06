package sheets_service

import (
	"errors"
	"strconv"
	"strings"
	"testing"

	"register/pkg/config"
	"register/pkg/models"
)

func budgetBulkSheet() BudgetSheet {
	return BudgetSheet{
		TabName:     BudgetTabName,
		SheetCoords: SheetCoords{StartRow: 3, EndRow: 63, EndColumnName: "J"},
	}
}

func Test_budgetService_NewBudgetSheet_Errors(t *testing.T) {
	cfg := config.Config{BudgetStartRow: 3, BudgetEndRow: 63}

	t.Run("spreadsheet error", func(t *testing.T) {
		svc := NewBudgetService(&stubSheetsProvider{spreadsheetErr: errTestProvider})
		if _, err := svc.NewBudgetSheet(cfg); !errors.Is(err, errTestProvider) {
			t.Errorf("NewBudgetSheet() error = %v, wantErr %v", err, errTestProvider)
		}
	})
	t.Run("header read error", func(t *testing.T) {
		stashProviderFuncs(t)
		getValuesProviderFunc = func(range_ string) ([][]any, error) {
			return nil, errTestProvider
		}
		if _, err := bs.NewBudgetSheet(cfg); err == nil || !strings.Contains(err.Error(), "could not read budget header row") {
			t.Errorf("NewBudgetSheet() error = %v, want header read error", err)
		}
	})
}

func Test_budgetService_ReadBudgetSheet_Errors(t *testing.T) {
	sheet := budgetBulkSheet()
	header := []any{"", "Weekly", "Bi-Weekly", "Monthly"}
	data := [][]any{
		{"", "Weekly", "Bi-Weekly", "Monthly"},
		{"Grocery", "120", "240", "520"},
	}
	tests := []struct {
		name    string
		stub    func(string) ([][]any, error)
		wantSub string
		wantErr error
	}{
		{
			name:    "values error",
			stub:    func(string) ([][]any, error) { return nil, errTestProvider },
			wantSub: "could not get sheet values",
		},
		{
			name:    "no values",
			stub:    func(string) ([][]any, error) { return [][]any{}, nil },
			wantSub: "no values found",
		},
		{
			name: "header read error",
			stub: func(range_ string) ([][]any, error) {
				if range_ == "Budget!A1:J1" {
					return nil, errTestProvider
				}
				return data, nil
			},
			wantSub: "could not read budget header row",
		},
		{
			name: "weekly header missing",
			stub: func(range_ string) ([][]any, error) {
				if range_ == "Budget!A1:J1" {
					return [][]any{{"", "Bi-Weekly", "Monthly"}}, nil
				}
				return data, nil
			},
			wantSub: `expected header "Weekly" not found`,
		},
		{
			name: "bi-weekly header missing",
			stub: func(range_ string) ([][]any, error) {
				if range_ == "Budget!A1:J1" {
					return [][]any{{"", "Weekly", "Monthly"}}, nil
				}
				return data, nil
			},
			wantSub: `expected header "Bi-Weekly" not found`,
		},
		{
			name: "monthly header missing",
			stub: func(range_ string) ([][]any, error) {
				if range_ == "Budget!A1:J1" {
					return [][]any{{"", "Weekly", "Bi-Weekly"}}, nil
				}
				return data, nil
			},
			wantSub: `expected header "Monthly" not found`,
		},
		{
			name: "bad weekly amount in a data row",
			stub: func(range_ string) ([][]any, error) {
				if range_ == "Budget!A1:J1" {
					return [][]any{header}, nil
				}
				return [][]any{
					{"", "Weekly", "Bi-Weekly", "Monthly"},
					{"Grocery", "xx", "240", "520"},
				}, nil
			},
			wantErr: strconv.ErrSyntax,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stashProviderFuncs(t)
			getValuesProviderFunc = tt.stub
			_, err := bs.ReadBudgetSheet(&sheet)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("ReadBudgetSheet() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantSub) {
				t.Errorf("ReadBudgetSheet() error = %v, want %q", err, tt.wantSub)
			}
		})
	}
}

func Test_budgetService_PopulateBudgetEntry_Errors(t *testing.T) {
	decoder, err := newRowDecoder([]any{"", "Weekly", "Bi-Weekly", "Monthly"})
	if err != nil {
		t.Fatalf("newRowDecoder() error = %v", err)
	}
	tests := []struct {
		name    string
		row     []any
		wantErr error
	}{
		{"bad weekly", []any{"Grocery", "xx", "240", "520"}, strconv.ErrSyntax},
		{"bad bi-weekly", []any{"Grocery", "120", "xx", "520"}, strconv.ErrSyntax},
		{"bad monthly", []any{"Grocery", "120", "240", "xx"}, strconv.ErrSyntax},
		{"good row", []any{"Grocery", "120", "240", "520"}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := bs.populateBudgetEntry(decoder, tt.row)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("populateBudgetEntry() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr != nil {
				return
			}
			if got.Category != "Grocery" || got.Weekly != 120 || got.BiWeekly != 240 || got.Monthly != 520 {
				t.Errorf("populateBudgetEntry() = %+v, want Grocery 120/240/520", got)
			}
		})
	}
}

func Test_budgetService_BudgetHeaderDecoder_NoHeader(t *testing.T) {
	stashProviderFuncs(t)
	getValuesProviderFunc = func(range_ string) ([][]any, error) {
		return [][]any{}, nil
	}
	if _, err := bs.budgetHeaderDecoder(budgetBulkSheet()); err == nil || !strings.Contains(err.Error(), "no header row found on Budget tab") {
		t.Errorf("budgetHeaderDecoder() error = %v, want no-header error", err)
	}
}

func Test_budgetService_BudgetRows_ProviderError(t *testing.T) {
	stashProviderFuncs(t)
	getValuesProviderFunc = func(range_ string) ([][]any, error) {
		return nil, errTestProvider
	}
	if _, err := bs.BudgetRows(budgetBulkSheet()); err == nil || !strings.Contains(err.Error(), "could not read budget categories") {
		t.Errorf("BudgetRows() error = %v, want categories read error", err)
	}
}

func Test_budgetService_BudgetEntryFor_Errors(t *testing.T) {
	sheet := budgetBulkSheet()
	columnA := [][]any{{"Grocery"}, {"Rent"}}
	header := []any{"", "Weekly", "Bi-Weekly", "Monthly"}
	row := [][]any{{"Grocery", "120", "240", "520"}}
	tests := []struct {
		name    string
		stub    func(string) ([][]any, error)
		wantSub string
	}{
		{
			name: "row read error",
			stub: func(range_ string) ([][]any, error) {
				if range_ == "Budget!A3:J3" {
					return nil, errTestProvider
				}
				return columnA, nil
			},
			wantSub: "could not read budget row 3",
		},
		{
			name: "header read error",
			stub: func(range_ string) ([][]any, error) {
				switch range_ {
				case "Budget!A3:A63":
					return columnA, nil
				case "Budget!A3:J3":
					return row, nil
				}
				return nil, errTestProvider
			},
			wantSub: "could not read budget header row",
		},
		{
			name: "bad row amount",
			stub: func(range_ string) ([][]any, error) {
				switch range_ {
				case "Budget!A3:A63":
					return columnA, nil
				case "Budget!A3:J3":
					return [][]any{{"Grocery", "xx", "240", "520"}}, nil
				case "Budget!A1:J1":
					return [][]any{header}, nil
				}
				return nil, nil
			},
			wantSub: "parseFloat error",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stashProviderFuncs(t)
			getValuesProviderFunc = tt.stub
			if _, _, err := bs.BudgetEntryFor(sheet, "Grocery"); err == nil || !strings.Contains(err.Error(), tt.wantSub) {
				t.Errorf("BudgetEntryFor() error = %v, want %q", err, tt.wantSub)
			}
		})
	}
}

func Test_budgetService_RenameBudgetCategory_UpdateError(t *testing.T) {
	stub := &stubSheetsProvider{
		getValuesFunc: func(string) ([][]any, error) {
			return [][]any{{"Grocery"}, {"Rent"}}, nil
		},
		updateErr: errTestProvider,
	}
	svc := NewBudgetService(stub)
	ok, err := svc.RenameBudgetCategory(budgetBulkSheet(), "Grocery", "Food")
	if err == nil || !strings.Contains(err.Error(), "could not rename budget row 3") {
		t.Errorf("RenameBudgetCategory() error = %v, want rename error", err)
	}
	if ok {
		t.Errorf("RenameBudgetCategory() = true, want false")
	}
}

func Test_budgetService_InsertBudgetCategory_Errors(t *testing.T) {
	sheet := BudgetSheet{
		ID:          987654321,
		TabName:     BudgetTabName,
		SheetCoords: SheetCoords{StartRow: 3, EndRow: 63},
	}
	columnA := func(string) ([][]any, error) {
		return [][]any{{"Discretionary"}, {"Grocery"}}, nil
	}

	t.Run("totals row lookup error", func(t *testing.T) {
		stashProviderFuncs(t)
		getValuesProviderFunc = func(range_ string) ([][]any, error) {
			return nil, errTestProvider
		}
		if err := bs.InsertBudgetCategory(sheet, "NewCat", models.ColorGreen); !errors.Is(err, errTestProvider) {
			t.Errorf("InsertBudgetCategory() error = %v, wantErr %v", err, errTestProvider)
		}
	})
	t.Run("batch update error", func(t *testing.T) {
		svc := NewBudgetService(&stubSheetsProvider{getValuesFunc: columnA, batchUpdateErr: errTestProvider})
		if err := svc.InsertBudgetCategory(sheet, "NewCat", models.ColorGreen); err == nil || !strings.Contains(err.Error(), "could not insert budget row") {
			t.Errorf("InsertBudgetCategory() error = %v, want insert error", err)
		}
	})
	t.Run("label update error", func(t *testing.T) {
		svc := NewBudgetService(&stubSheetsProvider{getValuesFunc: columnA, updateErr: errTestProvider})
		if err := svc.InsertBudgetCategory(sheet, "NewCat", models.ColorGreen); err == nil || !strings.Contains(err.Error(), "could not label budget row 3") {
			t.Errorf("InsertBudgetCategory() error = %v, want label error", err)
		}
	})
	t.Run("extend formulas error", func(t *testing.T) {
		svc := NewBudgetService(&stubSheetsProvider{getValuesFunc: columnA, getFormulaErr: errTestProvider})
		if err := svc.InsertBudgetCategory(sheet, "NewCat", models.ColorGreen); err == nil || !strings.Contains(err.Error(), "budget totals formulas extended failed") {
			t.Errorf("InsertBudgetCategory() error = %v, want extend error", err)
		}
	})
}

func Test_budgetService_DeleteBudgetCategory_BatchUpdateError(t *testing.T) {
	stub := &stubSheetsProvider{
		getValuesFunc: func(string) ([][]any, error) {
			return [][]any{{"Grocery"}, {"Rent"}}, nil
		},
		batchUpdateErr: errTestProvider,
	}
	svc := NewBudgetService(stub)
	if _, err := svc.DeleteBudgetCategory(budgetBulkSheet(), "Rent"); err == nil || !strings.Contains(err.Error(), "could not delete budget row 4") {
		t.Errorf("DeleteBudgetCategory() error = %v, want delete error", err)
	}
}

func Test_budgetService_MoveBudgetCategory_LookupErrors(t *testing.T) {
	sheet := budgetBulkSheet()
	columnA := [][]any{{"Discretionary"}, {"Grocery"}, {"Rent"}}
	// BudgetRowFor reads the same range twice (once per name), so the
	// stub fails on the Nth GetValues call to pick which lookup errors.
	// NOTE: BudgetRowFor reports a lookup failure as (0, false, err),
	// and the move functions check !found first, so a provider error
	// surfaces as "no budget row for ..." — the bare `return err`
	// after that check is unreachable.
	stub := func(failOn int) func(string) ([][]any, error) {
		calls := 0
		return func(string) ([][]any, error) {
			calls++
			if calls == failOn {
				return nil, errTestProvider
			}
			return columnA, nil
		}
	}
	tests := []struct {
		name    string
		failOn  int
		call    func() error
		wantSub string
	}{
		{"after: name lookup error", 1, func() error { return bs.MoveBudgetCategoryAfter(sheet, "Grocery", "Rent") }, `no budget row for "Grocery"`},
		{"after: after-name lookup error", 2, func() error { return bs.MoveBudgetCategoryAfter(sheet, "Grocery", "Rent") }, `no budget row for "Rent"`},
		{"before: name lookup error", 1, func() error { return bs.MoveBudgetCategoryBefore(sheet, "Rent", "Grocery") }, `no budget row for "Rent"`},
		{"before: before-name lookup error", 2, func() error { return bs.MoveBudgetCategoryBefore(sheet, "Rent", "Grocery") }, `no budget row for "Grocery"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stashProviderFuncs(t)
			getValuesProviderFunc = stub(tt.failOn)
			if err := tt.call(); err == nil || !strings.Contains(err.Error(), tt.wantSub) {
				t.Errorf("move error = %v, want %q", err, tt.wantSub)
			}
		})
	}
}

func Test_budgetService_MoveBudgetRow_BatchUpdateError(t *testing.T) {
	svc := NewBudgetService(&stubSheetsProvider{batchUpdateErr: errTestProvider})
	sheet := BudgetSheet{ID: 987654321, TabName: BudgetTabName}
	if err := svc.moveBudgetRow(sheet, 4, 2); err == nil || !strings.Contains(err.Error(), "could not move budget row 4") {
		t.Errorf("moveBudgetRow() error = %v, want move error", err)
	}
}
