package sheets_service

import (
	"errors"
	"reflect"
	"testing"

	"register/api/providers/sheets_provider"
	"register/pkg/config"
)

// houseBudgetHeaders are the Budget tab's real headers (row 1): the first
// cell is empty, the category name sits in column A.
var houseBudgetHeaders = []any{"", "Weekly", "Weekly (view only)", "Bi-Weekly", "Monthly", "Monthly (view only)", "Yearly", "Current Total", "Needed - Monthly"}

// houseBudgetRow builds one House Budget data row: name in column A, the
// Monthly amount in the Monthly column (index 4).
func houseBudgetRow(name, monthly string) []any {
	return []any{name, "0", "0", "0", monthly, "0", "0", "0"}
}

func Test_budgetService_ReadHousingSheet(t *testing.T) {
	cfg := config.Config{BudgetStartRow: 3, BudgetEndRow: 63}
	sheet := &sheets_provider.Spreadsheet{Title: "House Budget", SheetID: 123, RowCount: 100, ColumnCount: 10}

	newStub := func(t *testing.T, valuesFn func(string) ([][]any, error)) *stubSheetsProvider {
		t.Helper()
		return &stubSheetsProvider{
			spreadsheet:   sheet,
			getValuesFunc: valuesFn,
		}
	}
	happyValues := func(range_ string) ([][]any, error) {
		switch range_ {
		case "House Budget!A1:Z1", "House Budget!A1:H1":
			return [][]any{houseBudgetHeaders}, nil
		case "House Budget!A3:H100":
			return [][]any{
				houseBudgetRow("mortgage", "1450"),
				houseBudgetRow("mortgage insurance", "95"),
				houseBudgetRow("home insurance", "180"),
				houseBudgetRow("property tax", "325"),
				houseBudgetRow("home-expenses savings", "50"),
				// A category added after the build: read, not hardcoded.
				houseBudgetRow("HOA dues", "75"),
				// Group-totals rows: SUMs of the categories above them.
				// Counting one would double-count the group.
				houseBudgetRow("Non-Discretionary", "2050"),
				houseBudgetRow("Savings Categories", "125"),
				houseBudgetRow("Total", "2175"),
				{""},
			}, nil
		}
		t.Fatalf("unexpected range %q", range_)
		return nil, nil
	}

	tests := []struct {
		name    string
		stub    *stubSheetsProvider
		want    []HousingCategory
		wantErr bool
	}{
		{
			name: "reads every category row and skips the totals rows",
			stub: newStub(t, happyValues),
			want: []HousingCategory{
				{Name: "mortgage", Monthly: 1450},
				{Name: "mortgage insurance", Monthly: 95},
				{Name: "home insurance", Monthly: 180},
				{Name: "property tax", Monthly: 325},
				{Name: "home-expenses savings", Monthly: 50},
				{Name: "HOA dues", Monthly: 75},
			},
		},
		{
			name: "missing tab is an error",
			stub: &stubSheetsProvider{
				spreadsheetErr: errors.New("could not find sheet: House Budget"),
				getValuesFunc:  happyValues,
			},
			wantErr: true,
		},
		{
			name: "tab without a Monthly header is an error",
			stub: newStub(t, func(range_ string) ([][]any, error) {
				switch range_ {
				case "House Budget!A1:Z1", "House Budget!A1:B1":
					return [][]any{{"", "Weekly", "Bi-Weekly"}}, nil
				case "House Budget!A3:B100":
					return [][]any{houseBudgetRow("mortgage", "1450")}, nil
				}
				t.Fatalf("unexpected range %q", range_)
				return nil, nil
			}),
			wantErr: true,
		},
		{
			name: "empty tab is an error",
			stub: newStub(t, func(range_ string) ([][]any, error) {
				switch range_ {
				case "House Budget!A1:Z1", "House Budget!A1:H1":
					return [][]any{houseBudgetHeaders}, nil
				case "House Budget!A3:H100":
					return [][]any{}, nil
				}
				t.Fatalf("unexpected range %q", range_)
				return nil, nil
			}),
			wantErr: true,
		},
		{
			name: "provider error on the data range is an error",
			stub: newStub(t, func(range_ string) ([][]any, error) {
				switch range_ {
				case "House Budget!A1:Z1", "House Budget!A1:H1":
					return [][]any{houseBudgetHeaders}, nil
				case "House Budget!A3:H100":
					return nil, errTestProvider
				}
				t.Fatalf("unexpected range %q", range_)
				return nil, nil
			}),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bs := NewBudgetService(tt.stub)
			got, err := bs.ReadHousingSheet("House Budget", cfg)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ReadHousingSheet() error = nil, want an error")
				}
				return
			}
			if err != nil {
				t.Fatalf("ReadHousingSheet() error = %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ReadHousingSheet() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func Test_budgetService_NewBudgetSheetForTab(t *testing.T) {
	cfg := config.Config{BudgetStartRow: 3, BudgetEndRow: 63}
	sheet := &sheets_provider.Spreadsheet{Title: "House Budget", SheetID: 123, RowCount: 100, ColumnCount: 10}

	tests := []struct {
		name       string
		headers    [][]any
		wantEndRow int64
		wantEndCol string
		wantErr    bool
	}{
		{
			name:       "reads to the end of the tab's grid, not BudgetEndRow",
			headers:    [][]any{houseBudgetHeaders},
			wantEndRow: 100,
			wantEndCol: "H",
		},
		{
			name:    "no header row is an error",
			headers: [][]any{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stub := &stubSheetsProvider{
				spreadsheet: sheet,
				getValuesFunc: func(range_ string) ([][]any, error) {
					if range_ == "House Budget!A1:Z1" {
						return tt.headers, nil
					}
					t.Fatalf("unexpected range %q", range_)
					return nil, nil
				},
			}
			bs := NewBudgetService(stub)
			got, err := bs.NewBudgetSheetForTab(cfg, "House Budget")
			if tt.wantErr {
				if err == nil {
					t.Errorf("NewBudgetSheetForTab() error = nil, want an error")
				}
				return
			}
			if err != nil {
				t.Fatalf("NewBudgetSheetForTab() error = %v", err)
			}
			if got.TabName != "House Budget" {
				t.Errorf("TabName = %q, want %q", got.TabName, "House Budget")
			}
			if got.SheetCoords.EndRow != tt.wantEndRow {
				t.Errorf("EndRow = %d, want %d", got.SheetCoords.EndRow, tt.wantEndRow)
			}
			if got.SheetCoords.EndColumnName != tt.wantEndCol {
				t.Errorf("EndColumnName = %q, want %q", got.SheetCoords.EndColumnName, tt.wantEndCol)
			}
		})
	}
}
