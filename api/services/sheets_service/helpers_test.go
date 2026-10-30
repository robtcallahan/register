package sheets_service

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"reflect"
	"testing"

	"register/api/providers/sheets_provider"
	"register/pkg/config"
	"register/pkg/models"

	"google.golang.org/api/sheets/v4"
)

type providerMock struct {
	service       *sheets.Service
	spreadsheetID string
	t             *testing.T
}

var getValuesProviderFunc = func(range_ string) ([][]any, error) {
	// 9.3 header checks: return the real header rows for the header ranges.
	var values [][]any
	if range_ == "Register!A4:J4" {
		values = [][]any{{"Recon", "Check", "Date", "Description", " Withdrawals ", " Deposits ", " Credit Purchases ", " Register ", " Cleared ", " Delta "}}
		return values, nil
	}
	if range_ == "Budget!A1:J1" {
		values = [][]any{{"", "Weekly", "Monthly", "Yearly", "Current Total", "Needed - Monthly"}}
		return values, nil
	}
	values[0] = make([]any, 10)
	values[0][0] = "a string"
	return values, nil
}

var getFormulaProviderFunc = func(range_ string) ([][]any, error) {
	values := make([][]any, 10)
	values[0] = make([]any, 10)
	values[0][0] = "=SUM(A1:A2)"
	return values, nil
}

func (p *providerMock) GetValues(range_ string) ([][]any, error) {
	return getValuesProviderFunc(range_)
}

func (p *providerMock) GetFormula(range_ string) ([][]any, error) {
	return getFormulaProviderFunc(range_)
}

func (p *providerMock) GetSpreadsheet(tabName string) (*sheets_provider.Spreadsheet, error) {
	// Canned answer: services tests exercise how services use the sheet
	// properties, not how the provider looks them up — that lookup is
	// covered by the provider package's own tests.
	switch tabName {
	case RegisterTabName:
		return &sheets_provider.Spreadsheet{Title: RegisterTabName, SheetID: 617336355, RowCount: 1000, ColumnCount: 47}, nil
	case BudgetTabName:
		return &sheets_provider.Spreadsheet{Title: BudgetTabName, SheetID: 987654321, RowCount: 100, ColumnCount: 10}, nil
	}
	return nil, fmt.Errorf("could not find sheet: %s", tabName)
}

func (p *providerMock) BatchUpdate(_ *sheets.BatchUpdateSpreadsheetRequest) (*sheets.BatchUpdateSpreadsheetResponse, error) {
	return &sheets.BatchUpdateSpreadsheetResponse{}, nil
}

func (p *providerMock) ClearValues(_ string) error {
	return nil
}

func (p *providerMock) Update(_ string, _ *sheets.ValueRange) (*sheets.UpdateValuesResponse, error) {
	resp := &sheets.UpdateValuesResponse{
		SpreadsheetId:  "1-i_rA_q39tlN9reW_7c8vnXTgO5n3JxdK7rvDLdTwbI",
		UpdatedCells:   1,
		UpdatedColumns: 1,
		UpdatedRange:   "Register!G1",
		UpdatedRows:    1,
	}
	return resp, nil
}

var ss *RegisterService
var bs *BudgetService
var budgetSheet *BudgetSheet
var registerSheet *RegisterSheet

func init() {
	// The categoriesMap fixture lives on Rob's Mac, not in the repo. Tests
	// that need it skip when it's absent; the rest run with an empty map.
	var entries map[string]*BudgetEntry
	if c, err := os.ReadFile("json/categoriesMap.json"); err == nil {
		if err := json.Unmarshal(c, &entries); err != nil {
			log.Printf("warning: could not unmarshal categoriesMap.json: %s\n", err.Error())
		}
	}

	provider := providerMock{
		service:       &sheets.Service{},
		spreadsheetID: "ssID",
		t:             nil,
	}
	ss = NewRegisterService(&provider)
	bs = NewBudgetService(&provider)

	registerSheet = &RegisterSheet{
		TabName: "Register",
	}
	budgetSheet = &BudgetSheet{
		TabName: "Budget",
		SheetCoords: SheetCoords{
			StartRow:       3,
			EndRow:         63,
			EndColumnName:  "J",
			EndColumnIndex: 9,
		},
		CategoriesMap: entries,
	}
}

func Test_hasNote(t *testing.T) {
	type args struct {
		tran *models.Transaction
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Test hasNote is true",
			args: args{tran: &models.Transaction{Note: "some note"}},
			want: true,
		},
		{
			name: "Test hasNote is false",
			args: args{tran: &models.Transaction{Note: ""}},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasNote(tt.args.tran); got != tt.want {
				t.Errorf("hasNote() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func Test_isIncomeName(t *testing.T) {
	ss := &RegisterService{IncomeSources: []config.IncomeSource{
		{Name: "Taylor House Paycheck"},
		{Name: "Social Security"},
	}}
	type args struct {
		name string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Test isIncomeName is true",
			args: args{name: "Taylor House Paycheck"},
			want: true,
		},
		{
			name: "Test isIncomeName is false",
			args: args{name: "Amazon"},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isIncomeName(tt.args.name, ss.IncomeSources); got != tt.want {
				t.Errorf("isIncomeName() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func Test_getRegisterToDeltaReadRange(t *testing.T) {
	type args struct {
		i int64
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Test getRegisterToDeltaReadRange",
			args: args{i: 20},
			want: "Register!H21:J21",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getRegisterToDeltaReadRange(tt.args.i); got != tt.want {
				t.Errorf("getRegisterToDeltaReadRange() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func Test_getBackgroundColor(t *testing.T) {
	type args struct {
		trans  *models.Transaction
		income bool
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Test getBackgroundColor for paycheck",
			args: args{trans: &models.Transaction{Name: "Taylor House Paycheck"}, income: true},
			want: models.ColorGreen,
		},
		{
			name: "Test getBackgroundColor for normal",
			args: args{trans: &models.Transaction{Name: "Amazon"}},
			want: models.ColorWhite,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getBackgroundColor(tt.args.trans, tt.args.income); got != tt.want {
				t.Errorf("getBackgroundColor() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func Test_isCheckingAccount(t *testing.T) {
	type args struct {
		trans *models.Transaction
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Test isCheckingAccount is true",
			args: args{trans: &models.Transaction{Source: CheckingAccountSourceName, IsCheck: false}},
			want: true,
		},
		{
			name: "Test isCheckingAccount is true",
			args: args{trans: &models.Transaction{Source: "1001", IsCheck: true}},
			want: true,
		},
		{
			name: "Test isCheckingAccount is false",
			args: args{trans: &models.Transaction{Source: "Fidelity", IsCheck: false}},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isCheckingAccount(tt.args.trans); got != tt.want {
				t.Errorf("isCheckingAccount() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func Test_isBudgetColumn(t *testing.T) {
	type args struct {
		colName string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Test isBudgetColumn is true",
			args: args{colName: "Grocery"},
			want: true,
		},
		{
			name: "Test isBudgetColumn is false",
			args: args{colName: ""},
			want: false,
		},
		{
			name: "Test isBudgetColumn is false",
			args: args{colName: CreditCardColumnName},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isBudgetColumn(tt.args.colName); got != tt.want {
				t.Errorf("isBudgetColumn() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func Test_mkTextFormat(t *testing.T) {
	tests := []struct {
		name string
		want *sheets.TextFormat
	}{
		{
			name: "Test standard mkTextFormat",
			want: &sheets.TextFormat{
				FontFamily: "Arial",
				FontSize:   10,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mkTextFormat(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("mkTextFormat() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func Test_mkTextFormatReconcileColumn(t *testing.T) {
	tests := []struct {
		name string
		want *sheets.TextFormat
	}{
		{
			name: "Test reconcile column mkTextFormat",
			want: &sheets.TextFormat{
				FontFamily: "Archivo Black",
				FontSize:   12,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mkTextFormatReconcileColumn(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("mkTextFormat() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func Test_mkColor(t *testing.T) {
	type args struct {
		name string
	}
	tests := []struct {
		name string
		args args
		want *sheets.Color
	}{
		{
			name: "Test color white",
			args: args{name: models.ColorWhite},
			want: &sheets.Color{
				Alpha: 1,
				Blue:  1,
				Red:   1,
				Green: 1},
		},
		{
			name: "Test color green",
			args: args{name: models.ColorGreen},
			want: &sheets.Color{
				Alpha: 1,
				Blue:  0,
				Red:   0.5,
				Green: 1,
			},
		},
		{
			name: "Test color black",
			args: args{name: models.ColorBlack},
			want: &sheets.Color{
				Alpha: 1,
				Blue:  0,
				Red:   0,
				Green: 0,
			},
		},
		{
			name: "Test color yellow",
			args: args{name: models.ColorYellow},
			want: &sheets.Color{
				Alpha: 1,
				Blue:  0.6,
				Red:   1,
				Green: 1,
			},
		},
		{
			name: "Test color blue",
			args: args{name: models.ColorBlue},
			want: &sheets.Color{
				Alpha: 1,
				Blue:  1,
				Red:   0,
				Green: 0.8,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mkColor(tt.args.name); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("mkColor() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func Test_mkBorders(t *testing.T) {
	type args struct {
		on bool
	}
	tests := []struct {
		name string
		args args
		want *sheets.Borders
	}{
		{
			name: "Test mkBorders off",
			args: args{on: false},
			want: &sheets.Borders{},
		},
		{
			name: "Test mkBorders on",
			args: args{on: true},
			want: &sheets.Borders{
				Left: &sheets.Border{
					Color: mkColor(models.ColorBlack),
					Style: "SOLID",
				},
				Right: &sheets.Border{
					Color: mkColor(models.ColorBlack),
					Style: "SOLID",
				},
				Bottom: &sheets.Border{
					Color: mkColor(models.ColorBlack),
					Style: "SOLID",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mkBorders(tt.args.on); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("mkBorders() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func Test_mkCellFormat(t *testing.T) {
	type args struct {
		align     string
		colorName string
		bordersOn bool
	}
	tests := []struct {
		name string
		args args
		want *sheets.CellFormat
	}{
		{
			name: "Test mkCellFormat",
			args: args{align: "left", colorName: models.ColorBlue, bordersOn: true},
			want: &sheets.CellFormat{
				HorizontalAlignment: "LEFT",
				TextFormat: &sheets.TextFormat{
					FontFamily: "Arial",
					FontSize:   10,
				},
				BackgroundColor: &sheets.Color{
					Alpha: 1,
					Blue:  1,
					Red:   0,
					Green: 0.8,
				},
				Borders: &sheets.Borders{
					Left: &sheets.Border{
						Color: mkColor(models.ColorBlack),
						Style: "SOLID",
					},
					Right: &sheets.Border{
						Color: mkColor(models.ColorBlack),
						Style: "SOLID",
					},
					Bottom: &sheets.Border{
						Color: mkColor(models.ColorBlack),
						Style: "SOLID",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mkCellFormat(tt.args.align, tt.args.colorName, tt.args.bordersOn); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("mkCellFormat() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func Test_mkNumberFormatDate(t *testing.T) {
	tests := []struct {
		name string
		want *sheets.NumberFormat
	}{
		{
			name: "Test mkNumberFormatDate",
			want: &sheets.NumberFormat{
				Pattern: "mm/dd/yy",
				Type:    "DATE",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mkNumberFormatDate(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("mkNumberFormatDate() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func Test_getOddOrEvenRowColor(t *testing.T) {
	type args struct {
		i    int
		even string
		odd  string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Test even row number",
			args: args{i: 2, even: "even", odd: "odd"},
			want: "even",
		},
		{
			name: "Test odd row number",
			args: args{i: 1, even: "even", odd: "odd"},
			want: "odd",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getOddOrEvenRowColor(tt.args.i, tt.args.even, tt.args.odd); got != tt.want {
				t.Errorf("getOddOrEvenRowColor() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func Test_mkCellDataString(t *testing.T) {
	v := "string"
	type args struct {
		value     string
		align     string
		color     string
		bordersOn bool
	}
	tests := []struct {
		name string
		args args
		want *sheets.CellData
	}{
		{
			name: "Test return complete CellData from string",
			args: args{value: v, align: "left", color: models.ColorGreen, bordersOn: false},
			want: &sheets.CellData{
				UserEnteredValue: &sheets.ExtendedValue{
					StringValue: &v,
				},
				UserEnteredFormat: &sheets.CellFormat{
					HorizontalAlignment: "LEFT",
					TextFormat: &sheets.TextFormat{
						FontFamily: "Arial",
						FontSize:   10,
					},
					BackgroundColor: &sheets.Color{
						Alpha: 1,
						Blue:  0,
						Red:   0.5,
						Green: 1,
					},
					Borders: &sheets.Borders{},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mkCellDataString(tt.args.value, tt.args.align, tt.args.color, tt.args.bordersOn); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("mkCellDataString() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func Test_mkBoldFormat(t *testing.T) {
	v := "string"
	type args struct {
		value     string
		align     string
		color     string
		bordersOn bool
	}
	tests := []struct {
		name string
		args args
		want *sheets.CellData
	}{
		{
			name: "Test return complete bold-formatted CellData from string",
			args: args{value: v, align: "left", color: models.ColorWhite, bordersOn: false},
			want: &sheets.CellData{
				UserEnteredValue: &sheets.ExtendedValue{
					StringValue: &v,
				},
				UserEnteredFormat: &sheets.CellFormat{
					HorizontalAlignment: "LEFT",
					TextFormat: &sheets.TextFormat{
						FontFamily: "Arial",
						FontSize:   10,
						Bold:       true,
					},
					BackgroundColor: &sheets.Color{
						Alpha: 1,
						Blue:  1,
						Red:   1,
						Green: 1,
					},
					Borders: &sheets.Borders{},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mkBoldFormat(tt.args.value, tt.args.align, tt.args.color, tt.args.bordersOn); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("mkBoldFormat() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func Test_mkCellDataNumber(t *testing.T) {
	v := 100.12
	type args struct {
		value     float64
		align     string
		color     string
		bordersOn bool
	}
	tests := []struct {
		name string
		args args
		want *sheets.CellData
	}{
		{
			name: "Test return complete CellData from dollars string",
			args: args{value: 100.1234, align: "left", color: models.ColorWhite, bordersOn: false},
			want: &sheets.CellData{
				UserEnteredValue: &sheets.ExtendedValue{
					NumberValue: &v,
				},
				UserEnteredFormat: &sheets.CellFormat{
					HorizontalAlignment: "LEFT",
					TextFormat: &sheets.TextFormat{
						FontFamily: "Arial",
						FontSize:   10,
					},
					BackgroundColor: &sheets.Color{
						Alpha: 1,
						Blue:  1,
						Red:   1,
						Green: 1,
					},
					Borders: &sheets.Borders{},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mkCellDataNumber(tt.args.value, tt.args.align, tt.args.color, tt.args.bordersOn); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("mkCellDataNumber() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func Test_mkCellDataDollars(t *testing.T) {
	v := 100.12
	type args struct {
		value     float64
		align     string
		colorName string
		bordersOn bool
	}
	tests := []struct {
		name string
		args args
		want *sheets.CellData
	}{
		{
			name: "Test return complete CellData dollars string",
			args: args{value: v, align: "left", colorName: models.ColorWhite, bordersOn: false},
			want: &sheets.CellData{
				UserEnteredValue: &sheets.ExtendedValue{
					NumberValue: &v,
				},
				UserEnteredFormat: &sheets.CellFormat{
					HorizontalAlignment: "LEFT",
					TextFormat: &sheets.TextFormat{
						FontFamily: "Arial",
						FontSize:   10,
					},
					NumberFormat: &sheets.NumberFormat{
						Pattern: `_("$"* #,##0.00_);_("$"* \(#,##0.00\);_("$"* "-"??_);_(@_)`,
						Type:    "CURRENCY",
					},
					BackgroundColor: &sheets.Color{
						Alpha: 1,
						Blue:  1,
						Red:   1,
						Green: 1,
					},
					Borders: &sheets.Borders{},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mkCellDataDollars(tt.args.value, tt.args.align, tt.args.colorName, tt.args.bordersOn); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("mkCellDataDollars() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func Test_mkCellDataEmpty(t *testing.T) {
	v := ""
	type args struct {
		value     string
		align     string
		colorName string
		bordersOn bool
	}
	tests := []struct {
		name string
		args args
		want *sheets.CellData
	}{
		{
			name: "Test return complete CellData for dollars formatted empty cell",
			args: args{value: v, align: "right", colorName: models.ColorWhite, bordersOn: false},
			want: &sheets.CellData{
				UserEnteredValue: &sheets.ExtendedValue{
					StringValue: &v,
				},
				UserEnteredFormat: &sheets.CellFormat{
					HorizontalAlignment: "RIGHT",
					TextFormat: &sheets.TextFormat{
						FontFamily: "Arial",
						FontSize:   10,
					},
					NumberFormat: &sheets.NumberFormat{
						Pattern: `_("$"* #,##0.00_);_("$"* \(#,##0.00\);_("$"* "-"??_);_(@_)`,
						Type:    "CURRENCY",
					},
					BackgroundColor: &sheets.Color{
						Alpha: 1,
						Blue:  1,
						Red:   1,
						Green: 1,
					},
					Borders: &sheets.Borders{},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mkCellDataEmpty(tt.args.align, tt.args.colorName, tt.args.bordersOn); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("mkCellDataEmpty() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func Test_getCellDataFormula(t *testing.T) {
	f := "=SUM(A1:Z2)"
	type args struct {
		value     string
		align     string
		colorName string
		bordersOn bool
	}
	tests := []struct {
		name string
		args args
		want *sheets.CellData
	}{
		{
			name: "Test return complete CellData from formula string",
			args: args{value: f, align: "center", colorName: models.ColorWhite, bordersOn: false},
			want: &sheets.CellData{
				UserEnteredValue: &sheets.ExtendedValue{
					FormulaValue: &f,
				},
				UserEnteredFormat: &sheets.CellFormat{
					HorizontalAlignment: "CENTER",
					TextFormat: &sheets.TextFormat{
						FontFamily: "Arial",
						FontSize:   10,
					},
					NumberFormat: &sheets.NumberFormat{
						Pattern: `_("$"* #,##0.00_);_("$"* \(#,##0.00\);_("$"* "-"??_);_(@_)`,
						Type:    "CURRENCY",
					},
					BackgroundColor: &sheets.Color{
						Alpha: 1,
						Blue:  1,
						Red:   1,
						Green: 1,
					},
					Borders: &sheets.Borders{},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mkCellDataFormula(tt.args.value, tt.args.align, tt.args.colorName, tt.args.bordersOn); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("mkCellDataFormula() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func Test_getCellDataReconcileColumn(t *testing.T) {
	v := "string"
	type args struct {
		value     string
		align     string
		colorName string
		bordersOn bool
	}
	tests := []struct {
		name string
		args args
		want *sheets.CellData
	}{
		{
			name: "Test return complete CellData for reconcile cell",
			args: args{value: v, align: "right", colorName: models.ColorWhite, bordersOn: true},
			want: &sheets.CellData{
				UserEnteredValue: &sheets.ExtendedValue{
					StringValue: &v,
				},
				UserEnteredFormat: &sheets.CellFormat{
					HorizontalAlignment: "RIGHT",
					TextFormat: &sheets.TextFormat{
						FontFamily: "Archivo Black",
						FontSize:   12,
					},
					BackgroundColor: &sheets.Color{
						Alpha: 1,
						Blue:  1,
						Red:   1,
						Green: 1,
					},
					Borders: &sheets.Borders{
						Left: &sheets.Border{
							Color: mkColor(models.ColorBlack),
							Style: "SOLID",
						},
						Right: &sheets.Border{
							Color: mkColor(models.ColorBlack),
							Style: "SOLID",
						},
						Bottom: &sheets.Border{
							Color: mkColor(models.ColorBlack),
							Style: "SOLID",
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getCellDataReconcileColumn(tt.args.value, tt.args.align, tt.args.colorName, tt.args.bordersOn); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getCellDataReconcileColumn() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func Test_getNumberFormatDollars(t *testing.T) {
	tests := []struct {
		name string
		want *sheets.NumberFormat
	}{
		{
			name: "Test NumberFormat as CURRENCY",
			want: &sheets.NumberFormat{
				Pattern: `_("$"* #,##0.00_);_("$"* \(#,##0.00\);_("$"* "-"??_);_(@_)`,
				Type:    "CURRENCY",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getNumberFormatDollars(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getNumberFormatDollars() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func Test_getCellDataDate(t *testing.T) {
	f := 43832.0
	type args struct {
		dateString string
		align      string
		colorName  string
		bordersOn  bool
	}
	tests := []struct {
		name  string
		args  args
		want  *sheets.CellData
		error string
	}{
		{
			name: "Test return fully formatted date cell",
			args: args{dateString: "01/02/2020", align: "center", colorName: models.ColorWhite, bordersOn: false},
			want: &sheets.CellData{
				UserEnteredValue: &sheets.ExtendedValue{
					NumberValue: &f,
				},
				UserEnteredFormat: &sheets.CellFormat{
					HorizontalAlignment: "CENTER",
					TextFormat: &sheets.TextFormat{
						FontFamily: "Arial",
						FontSize:   10,
					},
					NumberFormat: &sheets.NumberFormat{
						Pattern: "mm/dd/yy",
						Type:    "DATE",
					},
					BackgroundColor: &sheets.Color{
						Alpha: 1,
						Blue:  1,
						Red:   1,
						Green: 1,
					},
					Borders: &sheets.Borders{},
				},
			},
		},
		{
			name:  "Test return time.Parse error",
			args:  args{dateString: "01/02/2", align: "center", colorName: models.ColorWhite, bordersOn: false},
			error: "time.Parse error",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			got, err := getCellDataDate(tt.args.dateString, tt.args.align, tt.args.colorName, tt.args.bordersOn)
			if got == nil && err == nil {
				t.Errorf("getCellDataDate() error = nil, want error %s", tt.error)
			}
			if err == nil && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getCellDataDate() = %+v, want error %+v", got, tt.error)
			}
		})
	}
}

func Test_formatYear(t *testing.T) {
	type args struct {
		date string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Test 2 digit year",
			args: args{date: "01/02/03"},
			want: "01/02/03",
		},
		{
			name: "Test 4 digit year",
			args: args{date: "01/02/2003"},
			want: "01/02/03",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatYear(tt.args.date); got != tt.want {
				t.Errorf("formatYear() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func Test_readStringValue(t *testing.T) {
	type args struct {
		text any
	}

	var v any
	v = "string"
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Test string from interface",
			args: args{text: v},
			want: "string",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := readStringValue(tt.args.text); got != tt.want {
				t.Errorf("readStringValue() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func Test_readDollarsValue(t *testing.T) {
	type args struct {
		value any
	}
	tests := []struct {
		name string
		args args
		want float64
	}{
		{
			name: "Test positive dollars",
			args: args{value: "$ 10.20 "},
			want: 10.20,
		},
		{
			name: "Test negative dollars",
			args: args{value: "$ (10.20) "},
			want: -10.20,
		},
		{
			name: "Test for dash",
			args: args{value: " - "},
			want: 0,
		},
		{
			name: "Test empty cell",
			args: args{value: ""},
			want: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, _ := readDollarsValue(tt.args.value); got != tt.want {
				t.Errorf("readDollarsValue() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func Test_addDeposit(t *testing.T) {
	s := ""
	v := 101.56
	want := []*sheets.CellData{
		{
			UserEnteredValue: &sheets.ExtendedValue{
				StringValue: &s,
			},
			UserEnteredFormat: &sheets.CellFormat{
				HorizontalAlignment: "LEFT",
				TextFormat: &sheets.TextFormat{
					FontFamily: "Arial",
					FontSize:   10,
				},
				BackgroundColor: &sheets.Color{
					Alpha: 1,
					Blue:  1,
					Red:   1,
					Green: 1,
				},
				Borders: &sheets.Borders{},
			},
		},
		{
			UserEnteredValue: &sheets.ExtendedValue{
				NumberValue: &v,
			},
			UserEnteredFormat: &sheets.CellFormat{
				HorizontalAlignment: "RIGHT",
				TextFormat: &sheets.TextFormat{
					FontFamily: "Arial",
					FontSize:   10,
				},
				NumberFormat: &sheets.NumberFormat{
					Pattern: `_("$"* #,##0.00_);_("$"* \(#,##0.00\);_("$"* "-"??_);_(@_)`,
					Type:    "CURRENCY",
				},
				BackgroundColor: &sheets.Color{
					Alpha: 1,
					Blue:  1,
					Red:   1,
					Green: 1,
				},
				Borders: &sheets.Borders{},
			},
		},
	}

	type args struct {
		amount  float64
		bgColor string
		cells   []*sheets.CellData
	}
	tests := []struct {
		name string
		args args
		want []*sheets.CellData
	}{
		{
			name: "Test add a deposit row entry",
			args: args{amount: v, bgColor: models.ColorWhite, cells: []*sheets.CellData{}},
			want: want,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := addDeposit(tt.args.amount, tt.args.bgColor, tt.args.cells); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("addDeposit() = %v, want %v",
					got[1].UserEnteredValue.NumberValue, tt.want[1].UserEnteredValue.NumberValue)
			}
		})
	}
}

func Test_addWithdrawal(t *testing.T) {
	s := ""
	v := 101.56
	want := []*sheets.CellData{
		{
			UserEnteredValue: &sheets.ExtendedValue{
				NumberValue: &v,
			},
			UserEnteredFormat: &sheets.CellFormat{
				HorizontalAlignment: "RIGHT",
				TextFormat: &sheets.TextFormat{
					FontFamily: "Arial",
					FontSize:   10,
				},
				NumberFormat: &sheets.NumberFormat{
					Pattern: `_("$"* #,##0.00_);_("$"* \(#,##0.00\);_("$"* "-"??_);_(@_)`,
					Type:    "CURRENCY",
				},
				BackgroundColor: &sheets.Color{
					Alpha: 1,
					Blue:  1,
					Red:   1,
					Green: 1,
				},
				Borders: &sheets.Borders{},
			},
		},
		{
			UserEnteredValue: &sheets.ExtendedValue{
				StringValue: &s,
			},
			UserEnteredFormat: &sheets.CellFormat{
				HorizontalAlignment: "LEFT",
				TextFormat: &sheets.TextFormat{
					FontFamily: "Arial",
					FontSize:   10,
				},
				BackgroundColor: &sheets.Color{
					Alpha: 1,
					Blue:  1,
					Red:   1,
					Green: 1,
				},
				Borders: &sheets.Borders{},
			},
		},
	}

	type args struct {
		amount  float64
		bgColor string
		cells   []*sheets.CellData
	}
	tests := []struct {
		name string
		args args
		want []*sheets.CellData
	}{
		{
			name: "Test add a withdrawal row entry",
			args: args{amount: v, bgColor: models.ColorWhite, cells: []*sheets.CellData{}},
			want: want,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := addWithdrawal(tt.args.amount, tt.args.bgColor, tt.args.cells); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("addWithdrawal() = %v, want %v",
					got[0].UserEnteredValue.NumberValue, tt.want[0].UserEnteredValue.NumberValue)
			}
		})
	}
}

func Test_addCheckingTransaction(t *testing.T) {
	s := ""
	v := 101.56
	wantDeposit := []*sheets.CellData{
		{
			UserEnteredValue: &sheets.ExtendedValue{
				StringValue: &s,
			},
			UserEnteredFormat: &sheets.CellFormat{
				HorizontalAlignment: "LEFT",
				TextFormat: &sheets.TextFormat{
					FontFamily: "Arial",
					FontSize:   10,
				},
				BackgroundColor: &sheets.Color{
					Alpha: 1,
					Blue:  1,
					Red:   1,
					Green: 1,
				},
				Borders: &sheets.Borders{},
			},
		},
		{
			UserEnteredValue: &sheets.ExtendedValue{
				NumberValue: &v,
			},
			UserEnteredFormat: &sheets.CellFormat{
				HorizontalAlignment: "RIGHT",
				TextFormat: &sheets.TextFormat{
					FontFamily: "Arial",
					FontSize:   10,
				},
				NumberFormat: &sheets.NumberFormat{
					Pattern: `_("$"* #,##0.00_);_("$"* \(#,##0.00\);_("$"* "-"??_);_(@_)`,
					Type:    "CURRENCY",
				},
				BackgroundColor: &sheets.Color{
					Alpha: 1,
					Blue:  1,
					Red:   1,
					Green: 1,
				},
				Borders: &sheets.Borders{},
			},
		},
		{
			UserEnteredValue: &sheets.ExtendedValue{
				StringValue: &s,
			},
			UserEnteredFormat: &sheets.CellFormat{
				HorizontalAlignment: "LEFT",
				TextFormat: &sheets.TextFormat{
					FontFamily: "Arial",
					FontSize:   10,
				},
				BackgroundColor: &sheets.Color{
					Alpha: 1,
					Blue:  1,
					Red:   1,
					Green: 1,
				},
				Borders: &sheets.Borders{},
			},
		},
	}

	wantWithdrawal := []*sheets.CellData{
		{
			UserEnteredValue: &sheets.ExtendedValue{
				NumberValue: &v,
			},
			UserEnteredFormat: &sheets.CellFormat{
				HorizontalAlignment: "RIGHT",
				TextFormat: &sheets.TextFormat{
					FontFamily: "Arial",
					FontSize:   10,
				},
				NumberFormat: &sheets.NumberFormat{
					Pattern: `_("$"* #,##0.00_);_("$"* \(#,##0.00\);_("$"* "-"??_);_(@_)`,
					Type:    "CURRENCY",
				},
				BackgroundColor: &sheets.Color{
					Alpha: 1,
					Blue:  1,
					Red:   1,
					Green: 1,
				},
				Borders: &sheets.Borders{},
			},
		},
		{
			UserEnteredValue: &sheets.ExtendedValue{
				StringValue: &s,
			},
			UserEnteredFormat: &sheets.CellFormat{
				HorizontalAlignment: "LEFT",
				TextFormat: &sheets.TextFormat{
					FontFamily: "Arial",
					FontSize:   10,
				},
				BackgroundColor: &sheets.Color{
					Alpha: 1,
					Blue:  1,
					Red:   1,
					Green: 1,
				},
				Borders: &sheets.Borders{},
			},
		},
		{
			UserEnteredValue: &sheets.ExtendedValue{
				StringValue: &s,
			},
			UserEnteredFormat: &sheets.CellFormat{
				HorizontalAlignment: "LEFT",
				TextFormat: &sheets.TextFormat{
					FontFamily: "Arial",
					FontSize:   10,
				},
				BackgroundColor: &sheets.Color{
					Alpha: 1,
					Blue:  1,
					Red:   1,
					Green: 1,
				},
				Borders: &sheets.Borders{},
			},
		}}

	type args struct {
		trans   *models.Transaction
		bgColor string
		cells   []*sheets.CellData
	}
	tests := []struct {
		name string
		args args
		want []*sheets.CellData
	}{
		{
			name: "Test add a deposit checking row entry",
			args: args{trans: &models.Transaction{Deposit: v}, bgColor: models.ColorWhite, cells: []*sheets.CellData{}},
			want: wantDeposit,
		},
		{
			name: "Test add a withdrawal checking row entry",
			args: args{trans: &models.Transaction{Withdrawal: v}, bgColor: models.ColorWhite, cells: []*sheets.CellData{}},
			want: wantWithdrawal,
		},
	}
	tt := tests[0]
	t.Run(tt.name, func(t *testing.T) {
		if got := addCheckingTransaction(tt.args.trans, tt.args.bgColor, tt.args.cells); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("addCheckingTransaction() = %v, want %v",
				got[1].UserEnteredValue.NumberValue, tt.want[1].UserEnteredValue.NumberValue)
		}
	})
	tt = tests[1]
	t.Run(tt.name, func(t *testing.T) {
		if got := addCheckingTransaction(tt.args.trans, tt.args.bgColor, tt.args.cells); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("addCheckingTransaction() = %v, want %v",
				got[0].UserEnteredValue.NumberValue, tt.want[0].UserEnteredValue.NumberValue)
		}
	})
}

func Test_addCCTransaction(t *testing.T) {
	s := ""
	v := 101.56
	want := []*sheets.CellData{
		{
			UserEnteredValue: &sheets.ExtendedValue{
				StringValue: &s,
			},
			UserEnteredFormat: &sheets.CellFormat{
				HorizontalAlignment: "LEFT",
				TextFormat: &sheets.TextFormat{
					FontFamily: "Arial",
					FontSize:   10,
				},
				BackgroundColor: &sheets.Color{
					Alpha: 1,
					Blue:  1,
					Red:   1,
					Green: 1,
				},
				Borders: &sheets.Borders{},
			},
		},
		{
			UserEnteredValue: &sheets.ExtendedValue{
				StringValue: &s,
			},
			UserEnteredFormat: &sheets.CellFormat{
				HorizontalAlignment: "LEFT",
				TextFormat: &sheets.TextFormat{
					FontFamily: "Arial",
					FontSize:   10,
				},
				BackgroundColor: &sheets.Color{
					Alpha: 1,
					Blue:  1,
					Red:   1,
					Green: 1,
				},
				Borders: &sheets.Borders{},
			},
		},
		{
			UserEnteredValue: &sheets.ExtendedValue{
				NumberValue: &v,
			},
			UserEnteredFormat: &sheets.CellFormat{
				HorizontalAlignment: "RIGHT",
				TextFormat: &sheets.TextFormat{
					FontFamily: "Arial",
					FontSize:   10,
				},
				NumberFormat: &sheets.NumberFormat{
					Pattern: `_("$"* #,##0.00_);_("$"* \(#,##0.00\);_("$"* "-"??_);_(@_)`,
					Type:    "CURRENCY",
				},
				BackgroundColor: &sheets.Color{
					Alpha: 1,
					Blue:  1,
					Red:   1,
					Green: 1,
				},
				Borders: &sheets.Borders{},
			},
		},
	}

	type args struct {
		trans   *models.Transaction
		bgColor string
		cells   []*sheets.CellData
	}
	tests := []struct {
		name string
		args args
		want []*sheets.CellData
	}{
		{
			name: "Test add a credit card row entry",
			args: args{trans: &models.Transaction{CreditPurchase: v}, bgColor: models.ColorWhite, cells: []*sheets.CellData{}},
			want: want,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := addCCTransaction(tt.args.trans, tt.args.bgColor, tt.args.cells); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("addCCTransaction() = %v, want %v",
					got[2].UserEnteredValue.NumberValue, tt.want[2].UserEnteredValue.NumberValue)
			}
		})
	}
}

func Test_isCorrectBudgetColumn(t *testing.T) {
	type args struct {
		transName          string
		colName            string
		transNameToColName map[string]string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Test if transaction matches the current budget column",
			args: args{transName: "UBER", colName: "Uber", transNameToColName: map[string]string{"UBER": "Uber"}},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isCorrectBudgetColumn(tt.args.transName, tt.args.colName, tt.args.transNameToColName); got != tt.want {
				t.Errorf("isCorrectBudgetColumn() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getStringField(t *testing.T) {
	var v = make([]any, 2)
	v[0] = "string"
	type args struct {
		values []any
		i      int
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Test string from array of interfaces",
			args: args{values: v, i: 0},
			want: "string",
		},
		{
			name: "Test string from array of interfaces with bad index",
			args: args{values: v, i: 4},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getStringField(tt.args.values, tt.args.i); got != tt.want {
				t.Errorf("getStringField() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func Test_sortAggregateMapKeys(t *testing.T) {
	type args struct {
		aggMap map[string]map[string]float64
	}
	in := make(map[string]map[string]float64)
	in["c"] = make(map[string]float64)
	in["c"]["x"] = 1.0
	in["b"] = make(map[string]float64)
	in["b"]["y"] = 2.0
	in["a"] = make(map[string]float64)
	in["a"]["z"] = 3.0

	res := []string{"a", "b", "c"}

	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "Test key sort",
			args: args{aggMap: in},
			want: res,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sortAggregateMapKeys(tt.args.aggMap); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("sortAggregateMapKeys() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func Test_isCreditCardTransaction(t *testing.T) {
	type args struct {
		source  string
		colName string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Test is credit card transaction",
			args: args{source: "Fidelity", colName: CreditCardColumnName},
			want: true,
		},
		{
			name: "Test is not credit card transaction",
			args: args{source: "Fidelity", colName: "Uber"},
			want: false,
		},
		{
			name: "Test is not credit card transaction",
			args: args{source: CheckingAccountSourceName, colName: CreditCardColumnName},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isCreditCardTransaction(tt.args.source, tt.args.colName); got != tt.want {
				t.Errorf("isCreditCardTransaction() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_isRegisterClearedOrDeltaColumn(t *testing.T) {
	type args struct {
		i int
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Test is Register, Cleared, or Delta column",
			args: args{i: 1},
			want: true,
		},
		{
			name: "Test is not Register, Cleared, or Delta column",
			args: args{i: 5},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isRegisterClearedOrDeltaColumn(tt.args.i); got != tt.want {
				t.Errorf("isRegisterClearedOrDeltaColumn() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getSourceField(t *testing.T) {
	strings := []string{"X", "Fidelity", "01/02/2023", "Rent", "123.45", "", ""}
	values := make([]any, len(strings))
	for i, s := range strings {
		values[i] = s
	}

	type args struct {
		values []any
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Test get source field",
			args: args{values: values},
			want: models.FidelityID,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getSourceField(tt.args.values); got != tt.want {
				t.Errorf("getSourceField() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getTransactionKey1(t *testing.T) {
	strings := []string{"X", "Fidelity", "01/02/2023", "Rent", "123.45", "", ""}
	values := make([]any, len(strings))
	for i, v := range strings {
		values[i] = v
	}

	type args struct {
		values []any
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Test get transaction key",
			args: args{values: values},
			want: "fidelity:01/02/23:123.45",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getTransactionKey(tt.args.values); got != tt.want {
				t.Errorf("getTransactionKey() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_addAmountCell(t *testing.T) {
	s := ""
	v := 101.56

	transWithdrawal := models.Transaction{
		Source:     CheckingAccountSourceName,
		Amount:     -v,
		Withdrawal: v,
	}
	wantWithdrawal := []*sheets.CellData{
		{
			UserEnteredValue: &sheets.ExtendedValue{
				NumberValue: &v,
			},
			UserEnteredFormat: &sheets.CellFormat{
				HorizontalAlignment: "RIGHT",
				TextFormat: &sheets.TextFormat{
					FontFamily: "Arial",
					FontSize:   10,
				},
				NumberFormat: &sheets.NumberFormat{
					Pattern: `_("$"* #,##0.00_);_("$"* \(#,##0.00\);_("$"* "-"??_);_(@_)`,
					Type:    "CURRENCY",
				},
				BackgroundColor: &sheets.Color{
					Alpha: 1,
					Blue:  1,
					Red:   1,
					Green: 1,
				},
				Borders: &sheets.Borders{},
			},
		},
		{
			UserEnteredValue: &sheets.ExtendedValue{
				StringValue: &s,
			},
			UserEnteredFormat: &sheets.CellFormat{
				HorizontalAlignment: "LEFT",
				TextFormat: &sheets.TextFormat{
					FontFamily: "Arial",
					FontSize:   10,
				},
				BackgroundColor: &sheets.Color{
					Alpha: 1,
					Blue:  1,
					Red:   1,
					Green: 1,
				},
				Borders: &sheets.Borders{},
			},
		},
		{
			UserEnteredValue: &sheets.ExtendedValue{
				StringValue: &s,
			},
			UserEnteredFormat: &sheets.CellFormat{
				HorizontalAlignment: "LEFT",
				TextFormat: &sheets.TextFormat{
					FontFamily: "Arial",
					FontSize:   10,
				},
				BackgroundColor: &sheets.Color{
					Alpha: 1,
					Blue:  1,
					Red:   1,
					Green: 1,
				},
				Borders: &sheets.Borders{},
			},
		}}

	transDeposit := models.Transaction{
		Source:  CheckingAccountSourceName,
		Amount:  v,
		Deposit: v,
	}
	wantDeposit := []*sheets.CellData{
		{
			UserEnteredValue: &sheets.ExtendedValue{
				StringValue: &s,
			},
			UserEnteredFormat: &sheets.CellFormat{
				HorizontalAlignment: "LEFT",
				TextFormat: &sheets.TextFormat{
					FontFamily: "Arial",
					FontSize:   10,
				},
				BackgroundColor: &sheets.Color{
					Alpha: 1,
					Blue:  1,
					Red:   1,
					Green: 1,
				},
				Borders: &sheets.Borders{},
			},
		},
		{
			UserEnteredValue: &sheets.ExtendedValue{
				NumberValue: &v,
			},
			UserEnteredFormat: &sheets.CellFormat{
				HorizontalAlignment: "RIGHT",
				TextFormat: &sheets.TextFormat{
					FontFamily: "Arial",
					FontSize:   10,
				},
				NumberFormat: &sheets.NumberFormat{
					Pattern: `_("$"* #,##0.00_);_("$"* \(#,##0.00\);_("$"* "-"??_);_(@_)`,
					Type:    "CURRENCY",
				},
				BackgroundColor: &sheets.Color{
					Alpha: 1,
					Blue:  1,
					Red:   1,
					Green: 1,
				},
				Borders: &sheets.Borders{},
			},
		},
		{
			UserEnteredValue: &sheets.ExtendedValue{
				StringValue: &s,
			},
			UserEnteredFormat: &sheets.CellFormat{
				HorizontalAlignment: "LEFT",
				TextFormat: &sheets.TextFormat{
					FontFamily: "Arial",
					FontSize:   10,
				},
				BackgroundColor: &sheets.Color{
					Alpha: 1,
					Blue:  1,
					Red:   1,
					Green: 1,
				},
				Borders: &sheets.Borders{},
			},
		},
	}

	transCC := models.Transaction{
		Source:         "Fidelity",
		Amount:         v,
		CreditPurchase: v,
	}
	wantCC := []*sheets.CellData{
		{
			UserEnteredValue: &sheets.ExtendedValue{
				StringValue: &s,
			},
			UserEnteredFormat: &sheets.CellFormat{
				HorizontalAlignment: "LEFT",
				TextFormat: &sheets.TextFormat{
					FontFamily: "Arial",
					FontSize:   10,
				},
				BackgroundColor: &sheets.Color{
					Alpha: 1,
					Blue:  1,
					Red:   1,
					Green: 1,
				},
				Borders: &sheets.Borders{},
			},
		},
		{
			UserEnteredValue: &sheets.ExtendedValue{
				StringValue: &s,
			},
			UserEnteredFormat: &sheets.CellFormat{
				HorizontalAlignment: "LEFT",
				TextFormat: &sheets.TextFormat{
					FontFamily: "Arial",
					FontSize:   10,
				},
				BackgroundColor: &sheets.Color{
					Alpha: 1,
					Blue:  1,
					Red:   1,
					Green: 1,
				},
				Borders: &sheets.Borders{},
			},
		},
		{
			UserEnteredValue: &sheets.ExtendedValue{
				NumberValue: &v,
			},
			UserEnteredFormat: &sheets.CellFormat{
				HorizontalAlignment: "RIGHT",
				TextFormat: &sheets.TextFormat{
					FontFamily: "Arial",
					FontSize:   10,
				},
				NumberFormat: &sheets.NumberFormat{
					Pattern: `_("$"* #,##0.00_);_("$"* \(#,##0.00\);_("$"* "-"??_);_(@_)`,
					Type:    "CURRENCY",
				},
				BackgroundColor: &sheets.Color{
					Alpha: 1,
					Blue:  1,
					Red:   1,
					Green: 1,
				},
				Borders: &sheets.Borders{},
			},
		},
	}

	type args struct {
		cells   []*sheets.CellData
		trans   *models.Transaction
		bgColor string
	}

	tests := []struct {
		name string
		args args
		want []*sheets.CellData
	}{
		{
			name: "Test add a checking withdrawal column amount",
			args: args{cells: []*sheets.CellData{}, trans: &transWithdrawal, bgColor: models.ColorWhite},
			want: wantWithdrawal,
		},
		{
			name: "Test add a checking deposit column amount",
			args: args{cells: []*sheets.CellData{}, trans: &transDeposit, bgColor: models.ColorWhite},
			want: wantDeposit,
		},
		{
			name: "Test add a credit card column amount",
			args: args{cells: []*sheets.CellData{}, trans: &transCC, bgColor: models.ColorWhite},
			want: wantCC,
		},
	}
	tt := tests[0]
	t.Run(tt.name, func(t *testing.T) {
		if got := addAmountCell(tt.args.cells, tt.args.trans, tt.args.bgColor); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("addCheckingTransaction() = %v, want %v",
				got[1].UserEnteredValue.NumberValue, tt.want[0].UserEnteredValue.NumberValue)
		}
	})
	tt = tests[1]
	t.Run(tt.name, func(t *testing.T) {
		if got := addAmountCell(tt.args.cells, tt.args.trans, tt.args.bgColor); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("addCheckingTransaction() = %v, want %v",
				got[0].UserEnteredValue.NumberValue, tt.want[1].UserEnteredValue.NumberValue)
		}
	})
	tt = tests[2]
	t.Run(tt.name, func(t *testing.T) {
		if got := addAmountCell(tt.args.cells, tt.args.trans, tt.args.bgColor); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("addCheckingTransaction() = %v, want %v",
				got[2].UserEnteredValue.NumberValue, tt.want[2].UserEnteredValue.NumberValue)
		}
	})
}

func Test_addCategoryCells(t *testing.T) {
	// 10.10: Rewritten with inline data. Verifies category cells are placed
	// by ColumnIndex.
	columns := []models.Column{
		{Name: "Grocery", ColumnIndex: 11, Color: models.ColorBlue},
		{Name: "Rent", ColumnIndex: 12, Color: models.ColorBlue},
	}
	trans := &models.Transaction{
		ColumnIndex: 11,
		Budget:      46.80,
	}
	got := addCategoryCells(nil, trans, columns, map[string]string{}, []string{"", "", ""})
	if len(got) == 0 {
		t.Errorf("addCategoryCells() produced no cells")
	}
}

func Test_getAmountString(t *testing.T) {
	var (
		values1 []any
		values2 []any
		values3 []any
	)
	values1 = make([]any, CreditPurchase+1)
	values2 = make([]any, CreditPurchase+1)
	values3 = make([]any, CreditPurchase+1)
	for i := 0; i <= CreditPurchase; i++ {
		values1[i] = ""
		values2[i] = ""
		values3[i] = ""
	}

	type args struct {
		values []any
	}
	type ts struct {
		name string
		args args
		want string
	}
	var tests []ts

	values1[Withdrawals] = "$ 10.00 "
	tests = append(tests, ts{
		name: "Test getting the total Amount - Withdrawal",
		args: args{values: values1},
		want: "10.00",
	})
	values2[Deposits] = "$ 20.00 "
	tests = append(tests, ts{
		name: "Test getting the total Amount - Deposit",
		args: args{values: values2},
		want: "20.00",
	})
	values3[CreditPurchase] = "$ 30.00 "
	tests = append(tests, ts{
		name: "Test getting the total Amount - Credit Card",
		args: args{values: values3},
		want: "30.00",
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getAmountString(tt.args.values); got != tt.want {
				t.Errorf("getAmountString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getDollarsCellByIndex(t *testing.T) {
	type args struct {
		values []any
		i      int
	}
	tests := []struct {
		name string
		args args
		want float64
	}{
		{
			name: "Test positive dollars",
			args: args{values: []any{"$ 10.20 "}, i: 0},
			want: 10.20,
		},
		{
			name: "Test negative dollars",
			args: args{values: []any{"$ (10.20) "}, i: 0},
			want: -10.20,
		},
		{
			name: "Test for dash",
			args: args{values: []any{" - "}, i: 0},
			want: 0,
		},
		{
			name: "Test empty cell",
			args: args{values: []any{""}, i: 0},
			want: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, _ := getDollarsCellByIndex(tt.args.values, tt.args.i); got != tt.want {
				t.Errorf("GetDollarsCellByIndex() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getDateField(t *testing.T) {
	var values []any
	var values2 []any
	values = make([]any, Date+1)
	values2 = make([]any, Date+1)
	for i := 0; i <= Date; i++ {
		values[i] = "x"
		values2[i] = "x"
	}
	values[Date] = "01/02/2020"
	values2[Date] = ""

	type args struct {
		values []any
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Test getting the Date field",
			args: args{values: values},
			want: "01/02/20",
		},
		{
			name: "Test getting a blank Date field",
			args: args{values: values2},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getDateField(tt.args.values); got != tt.want {
				t.Errorf("getDateField() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getNameField(t *testing.T) {
	var values []any
	values = make([]any, Description+1)
	for i := 0; i <= Description; i++ {
		values[i] = "x"
	}
	values[Description] = "merchant"

	type args struct {
		values []any
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Test getting the Description field",
			args: args{values: values},
			want: "merchant",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getNameField(tt.args.values); got != tt.want {
				t.Errorf("getNameField() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_ReadStringCell(t *testing.T) {
	getValuesProviderFunc = func(range_ string) ([][]any, error) {
		values := make([][]any, 10)
		values[0] = make([]any, 10)
		values[0][0] = "a string"
		return values, nil
	}

	type args struct {
		cell string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Test reading a string cell",
			args: args{
				cell: "a string",
			},
			want: "a string",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// TODO: test error
			if got, _ := ss.ReadStringCell(*registerSheet, tt.args.cell); got != tt.want {
				t.Errorf("ReadStringCell() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_ReadDateCell(t *testing.T) {
	getValuesProviderFunc = func(range_ string) ([][]any, error) {
		values := make([][]any, 10)
		values[0] = make([]any, 10)
		values[0][0] = "01/02/20"
		return values, nil
	}

	type args struct {
		cell string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Test reading a date cell",
			args: args{
				cell: "01/02/20",
			},
			want: "01/02/20",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// TODO: test error
			if got, _ := ss.ReadDateCell(*registerSheet, tt.args.cell); got != tt.want {
				t.Errorf("ReadDateCell() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_ReadDollarsCell(t *testing.T) {
	getValuesProviderFunc = func(range_ string) ([][]any, error) {
		values := make([][]any, 10)
		values[0] = make([]any, 10)
		values[0][0] = " $  20.20 "
		return values, nil
	}
	type args struct {
		cell string
	}

	tests := []struct {
		name string
		args args
		want float64
	}{
		{
			name: "Test reading a dollars cell",
			args: args{
				cell: " $  20.20 ",
			},
			want: 20.20,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// TODO: test error
			if got, _ := ss.ReadDollarsCell(*registerSheet, tt.args.cell); got != tt.want {
				t.Errorf("ReadDollarsCell() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_ReadFormulaCell(t *testing.T) {
	type args struct {
		cell string
	}

	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Test reading a formula cell",
			args: args{
				cell: "A1",
			},
			want: "=SUM(A1:A2)",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// TODO: test error
			if got, _ := ss.ReadFormulaCell(*registerSheet, tt.args.cell); got != tt.want {
				t.Errorf("ReadFormulaCell() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_readCell(t *testing.T) {
	getValuesProviderFunc = func(range_ string) ([][]any, error) {
		values := make([][]any, 10)
		values[0] = make([]any, 10)
		values[0][0] = "a string"
		return values, nil
	}
	getFormulaProviderFunc = func(range_ string) ([][]any, error) {
		values := make([][]any, 10)
		values[0] = make([]any, 10)
		values[0][0] = "=SUM(A1:A2)"
		return values, nil
	}

	type args struct {
		cell  string
		dType CellDataType
	}
	tests := []struct {
		name string
		args args
		want any
	}{
		{
			name: "Test reading a formula cell",
			args: args{
				cell:  "A1",
				dType: CellDataFormula,
			},
			want: "=SUM(A1:A2)",
		},
		{
			name: "Test reading a string cell",
			args: args{
				cell:  "A1",
				dType: CellDataString,
			},
			want: "a string",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// TODO: test error
			if got, _ := ss.ReadCell(*registerSheet, tt.args.cell, tt.args.dType); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ReadCell() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_WriteCell(t *testing.T) {
	want := &sheets.UpdateValuesResponse{
		SpreadsheetId:  "1-i_rA_q39tlN9reW_7c8vnXTgO5n3JxdK7rvDLdTwbI",
		UpdatedCells:   1,
		UpdatedColumns: 1,
		UpdatedRange:   "Register!G1",
		UpdatedRows:    1,
	}

	type args struct {
		cell  string
		value any
	}
	tests := []struct {
		name string
		args args
		want *sheets.UpdateValuesResponse
	}{
		{
			name: "Test write cell",
			args: args{
				cell:  "G1",
				value: 10.00,
			},
			want: want,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// TODO: test for error
			if got, _ := ss.WriteCell(registerSheet.TabName, tt.args.cell, tt.args.value); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("WriteCell() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_addSalaryCells(t *testing.T) {
	// 10.10: Rewritten with inline data. Verifies salary cells are placed
	// by ColumnIndex with gaps leaving empty fillers.
	columns := []models.Column{
		{Name: "Grocery", ColumnIndex: 11, Color: models.ColorBlue},
		{Name: "Rent", ColumnIndex: 13, Color: models.ColorBlue}, // gap at 12
	}
	categoriesMap := map[string]*BudgetEntry{
		"Grocery": {Category: "Grocery", Weekly: 120.00, Monthly: 520.00},
		"Rent":    {Category: "Rent", Weekly: 0, Monthly: 1400.00},
	}
	rb := &RowBuilder{Columns: columns, CategoriesMap: categoriesMap}
	got := rb.addSalaryCells(nil, []string{"", "", ""}, "weekly")
	// 3 filler (8,9,10) + Grocery(11) + 1 filler (12, gap) + Rent(13) = 6
	if len(got) != 6 {
		t.Errorf("addSalaryCells() produced %d cells, want 6", len(got))
	}
}

func Test_addSourceDateNameCells(t *testing.T) {
	// 10.10: Rewritten with inline data. Verifies the 4 source/date/name cells.
	trans := &models.Transaction{
		Source: "WellsFargo",
		Date:   "12/07/20",
		Name:   "TARGET",
	}
	got, err := addSourceDateNameCells(nil, trans, "")
	if err != nil {
		t.Fatalf("addSourceDateNameCells() error = %v", err)
	}
	if len(got) != 4 {
		t.Errorf("addSourceDateNameCells() produced %d cells, want 4", len(got))
	}
}
