package sheets_service

import (
	"encoding/json"
	"os"
	"reflect"
	"register/pkg/models"
	"testing"

	"google.golang.org/api/sheets/v4"
)

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

func Test_isPaycheck(t *testing.T) {
	type args struct {
		name string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Test isPaycheck is true",
			args: args{name: PayCheckName},
			want: true,
		},
		{
			name: "Test isPaycheck is false",
			args: args{name: "Amazon"},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isPaycheck(tt.args.name); got != tt.want {
				t.Errorf("isPaycheck() = %+v, want %+v", got, tt.want)
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
		trans *models.Transaction
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Test getBackgroundColor for paycheck",
			args: args{trans: &models.Transaction{Name: PayCheckName}},
			want: "green",
		},
		{
			name: "Test getBackgroundColor for normal",
			args: args{trans: &models.Transaction{Name: "Amazon"}},
			want: "white",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getBackgroundColor(tt.args.trans); got != tt.want {
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
			args: args{name: "white"},
			want: &sheets.Color{
				Alpha: 1,
				Blue:  1,
				Red:   1,
				Green: 1},
		},
		{
			name: "Test color green",
			args: args{name: "green"},
			want: &sheets.Color{
				Alpha: 1,
				Blue:  0,
				Red:   0.5,
				Green: 1,
			},
		},
		{
			name: "Test color black",
			args: args{name: "black"},
			want: &sheets.Color{
				Alpha: 1,
				Blue:  0,
				Red:   0,
				Green: 0,
			},
		},
		{
			name: "Test color yellow",
			args: args{name: "yellow"},
			want: &sheets.Color{
				Alpha: 1,
				Blue:  0.6,
				Red:   1,
				Green: 1,
			},
		},
		{
			name: "Test color blue",
			args: args{name: "blue"},
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
					Color: mkColor("black"),
					Style: "SOLID",
				},
				Right: &sheets.Border{
					Color: mkColor("black"),
					Style: "SOLID",
				},
				Bottom: &sheets.Border{
					Color: mkColor("black"),
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
			args: args{align: "left", colorName: "blue", bordersOn: true},
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
						Color: mkColor("black"),
						Style: "SOLID",
					},
					Right: &sheets.Border{
						Color: mkColor("black"),
						Style: "SOLID",
					},
					Bottom: &sheets.Border{
						Color: mkColor("black"),
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
			args: args{value: v, align: "left", color: "green", bordersOn: false},
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
			args: args{value: v, align: "left", color: "green", bordersOn: false},
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
			args: args{value: 100.1234, align: "left", color: "green", bordersOn: false},
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
			args: args{value: v, align: "left", colorName: "white", bordersOn: false},
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
			args: args{value: v, align: "right", colorName: "green", bordersOn: false},
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
			args: args{value: f, align: "center", colorName: "green", bordersOn: false},
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
			args: args{value: v, align: "right", colorName: "green", bordersOn: true},
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
						Blue:  0,
						Red:   0.5,
						Green: 1,
					},
					Borders: &sheets.Borders{
						Left: &sheets.Border{
							Color: mkColor("black"),
							Style: "SOLID",
						},
						Right: &sheets.Border{
							Color: mkColor("black"),
							Style: "SOLID",
						},
						Bottom: &sheets.Border{
							Color: mkColor("black"),
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
			args: args{dateString: "01/02/2020", align: "center", colorName: "green", bordersOn: false},
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
						Blue:  0,
						Red:   0.5,
						Green: 1,
					},
					Borders: &sheets.Borders{},
				},
			},
		},
		{
			name:  "Test return time.Parse error",
			args:  args{dateString: "01/02/2", align: "center", colorName: "green", bordersOn: false},
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
		text interface{}
	}

	var v interface{}
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
		value interface{}
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
			if got := readDollarsValue(tt.args.value); got != tt.want {
				t.Errorf("readDollarsValue() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func Test_readDateValue(t *testing.T) {
	type args struct {
		dateStr interface{}
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Test 2 digit year",
			args: args{dateStr: "01/02/03"},
			want: "01/02/03",
		},
		{
			name: "Test 2 digit month and day",
			args: args{dateStr: "1/2/03"},
			want: "01/02/03",
		},
		{
			name: "Test 4 digit year",
			args: args{dateStr: "1/2/2003"},
			want: "01/02/03",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := readDateValue(tt.args.dateStr); got != tt.want {
				t.Errorf("readDateValue() = %+v, want %+v", got, tt.want)
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
			args: args{amount: v, bgColor: "white", cells: []*sheets.CellData{}},
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
			args: args{amount: -v, bgColor: "white", cells: []*sheets.CellData{}},
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
			name: "Test add a deposit checking row entry",
			args: args{amount: v, bgColor: "white", cells: []*sheets.CellData{}},
			want: wantDeposit,
		},
		{
			name: "Test add a withdrawal checking row entry",
			args: args{amount: -v, bgColor: "white", cells: []*sheets.CellData{}},
			want: wantWithdrawal,
		},
	}
	tt := tests[0]
	t.Run(tt.name, func(t *testing.T) {
		if got := addCheckingTransaction(tt.args.amount, tt.args.bgColor, tt.args.cells); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("addCheckingTransaction() = %v, want %v",
				got[1].UserEnteredValue.NumberValue, tt.want[1].UserEnteredValue.NumberValue)
		}
	})
	tt = tests[1]
	t.Run(tt.name, func(t *testing.T) {
		if got := addCheckingTransaction(tt.args.amount, tt.args.bgColor, tt.args.cells); !reflect.DeepEqual(got, tt.want) {
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
			name: "Test add a credit card row entry",
			args: args{amount: -v, bgColor: "white", cells: []*sheets.CellData{}},
			want: want,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := addCCTransaction(tt.args.amount, tt.args.bgColor, tt.args.cells); !reflect.DeepEqual(got, tt.want) {
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
	var v = make([]interface{}, 2)
	v[0] = "string"
	type args struct {
		values []interface{}
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

func Test_intInSlice(t *testing.T) {
	type args struct {
		a    int
		list []int
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Test in slice",
			args: args{a: 1, list: []int{2, 3, 4}},
			want: false,
		},
		{
			name: "Test not in slice",
			args: args{a: 3, list: []int{2, 3, 4}},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := intInSlice(tt.args.a, tt.args.list); got != tt.want {
				t.Errorf("intInSlice() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func Test_sortAggregateMapKeys(t *testing.T) {
	type args struct {
		aggMap *map[string]map[string]float64
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
		want *[]string
	}{
		{
			name: "Test key sort",
			args: args{aggMap: &in},
			want: &res,
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
	values := make([]interface{}, len(strings))
	for i, s := range strings {
		values[i] = s
	}

	type args struct {
		values []interface{}
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Test get source field",
			args: args{values: values},
			want: "Fidelity",
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

func Test_getDateField(t *testing.T) {
	strings := []string{"X", "Fidelity", "01/02/2023", "Rent", "123.45", "", ""}
	values := make([]interface{}, len(strings))
	for i, s := range strings {
		values[i] = s
	}

	type args struct {
		values []interface{}
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Test get date field",
			args: args{values: values},
			want: "01/02/23",
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

func Test_getAmountString(t *testing.T) {
	strings := []string{"X", "Fidelity", "01/02/2023", "Rent", "123.45", "", ""}
	values := make([]interface{}, len(strings))
	for i, v := range strings {
		values[i] = v
	}

	type args struct {
		values []interface{}
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Test get amount field",
			args: args{values: values},
			want: "-123.45",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getAmountString(tt.args.values); got != tt.want {
				t.Errorf("getAmountString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getTransactionKey1(t *testing.T) {
	strings := []string{"X", "Fidelity", "01/02/2023", "Rent", "123.45", "", ""}
	values := make([]interface{}, len(strings))
	for i, v := range strings {
		values[i] = v
	}

	type args struct {
		values []interface{}
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Test get transaction key",
			args: args{values: values},
			want: "Fidelity:01/02/23:-123.45",
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

func Test_addSourceDateNameCells(t *testing.T) {
	reconcile := "X"
	source := "Fidelity"
	date := float64(44928)
	name := "Amazon"

	want := []*sheets.CellData{
		{
			UserEnteredValue: &sheets.ExtendedValue{
				StringValue: &reconcile,
			},
			UserEnteredFormat: &sheets.CellFormat{
				HorizontalAlignment: "CENTER",
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
				Borders: &sheets.Borders{},
			},
		},
		{
			UserEnteredValue: &sheets.ExtendedValue{
				StringValue: &source,
			},
			UserEnteredFormat: &sheets.CellFormat{
				HorizontalAlignment: "CENTER",
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
				NumberValue: &date,
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
		{
			UserEnteredValue: &sheets.ExtendedValue{
				StringValue: &name,
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
		cells   []*sheets.CellData
		trans   *models.Transaction
		bgColor string
	}
	tests := []struct {
		name    string
		args    args
		want    []*sheets.CellData
		wantErr bool
	}{
		{
			name: "Test add reconcile, source, date, & name cells",
			args: args{cells: []*sheets.CellData{}, bgColor: "white", trans: &models.Transaction{
				Source: "Fidelity",
				Date:   "01/02/2023",
				Name:   "Amazon",
			}},
			want: want,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, _ := addSourceDateNameCells(tt.args.cells, tt.args.trans, tt.args.bgColor); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("addSourceDateNameCells(): reconcile got: %v, want %v; source got: %v, want %v; date got: %v, want %v; name got: %v, want %v",
					*got[0].UserEnteredValue.StringValue, *tt.want[0].UserEnteredValue.StringValue,
					*got[1].UserEnteredValue.StringValue, *tt.want[1].UserEnteredValue.StringValue,
					*got[2].UserEnteredValue.NumberValue, *tt.want[2].UserEnteredValue.NumberValue,
					*got[3].UserEnteredValue.StringValue, *tt.want[3].UserEnteredValue.StringValue,
				)
			}
		})
	}
}

func Test_addAmountCell(t *testing.T) {
	s := ""
	v := 101.56

	transWithdrawal := models.Transaction{
		Source: CheckingAccountSourceName,
		Amount: -v,
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
		Source: CheckingAccountSourceName,
		Amount: v,
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
		Source:     "Fidelity",
		Amount:     v,
		CreditCard: -v,
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
			args: args{cells: []*sheets.CellData{}, trans: &transWithdrawal, bgColor: "white"},
			want: wantWithdrawal,
		},
		{
			name: "Test add a checking deposit column amount",
			args: args{cells: []*sheets.CellData{}, trans: &transDeposit, bgColor: "white"},
			want: wantDeposit,
		},
		{
			name: "Test add a credit card column amount",
			args: args{cells: []*sheets.CellData{}, trans: &transCC, bgColor: "white"},
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
				got[0].UserEnteredValue.NumberValue, tt.want[3].UserEnteredValue.NumberValue)
		}
	})
}

func Test_addCategoryCells(t *testing.T) {
	j, err := os.ReadFile(sheetsServiceJSONDir + "transactions.json")
	checkTestingError(t, err)
	var trans []models.Transaction
	err = json.Unmarshal(j, &trans)
	checkTestingError(t, err)

	j, err = os.ReadFile(sheetsServiceJSONDir + "columns.json")
	checkTestingError(t, err)
	var columns []models.Column
	err = json.Unmarshal(j, &columns)
	checkTestingError(t, err)

	j, err = os.ReadFile(sheetsServiceJSONDir + "transNameToColName.json")
	checkTestingError(t, err)
	var transNameToColName map[string]string
	err = json.Unmarshal(j, &transNameToColName)
	checkTestingError(t, err)

	j, err = os.ReadFile(sheetsServiceJSONDir + "want_addCategoryCells.json")
	checkTestingError(t, err)
	var want []*sheets.CellData
	err = json.Unmarshal(j, &want)
	checkTestingError(t, err)

	type args struct {
		cells              []*sheets.CellData
		trans              *models.Transaction
		columns            []models.Column
		transNameToColName map[string]string
		totalsFormulas     []string
	}
	tests := []struct {
		name string
		args args
		want []*sheets.CellData
	}{
		{
			name: "Test adding in a category column",
			args: args{
				cells:              []*sheets.CellData{},
				columns:            columns,
				trans:              &trans[0],
				transNameToColName: transNameToColName,
				totalsFormulas:     []string{"=A1*2", "=A2*5", "=A2*5"},
			},
			want: want,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := addCategoryCells(tt.args.cells, tt.args.trans, tt.args.columns, tt.args.transNameToColName, tt.args.totalsFormulas); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("addCategoryCells() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getDollarsCellByIndex(t *testing.T) {
	strings := []string{"X", "Fidelity", "01/02/2023", "Rent", "123.45", "", ""}
	values := make([]interface{}, len(strings))
	for i, v := range strings {
		values[i] = v
	}

	type args struct {
		values []interface{}
		i      int
	}
	tests := []struct {
		name string
		args args
		want float64
	}{
		{
			name: "Test get dollar amount by values index",
			args: args{values: values, i: 4},
			want: 123.45,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getDollarsCellByIndex(tt.args.values, tt.args.i); got != tt.want {
				t.Errorf("getDollarsCellByIndex() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getNameField(t *testing.T) {
	strings := []string{"X", "Fidelity", "01/02/2023", "Rent", "123.45", "", ""}
	values := make([]interface{}, len(strings))
	for i, v := range strings {
		values[i] = v
	}

	type args struct {
		values []interface{}
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Test get name field from values",
			args: args{values: values},
			want: "Rent",
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
