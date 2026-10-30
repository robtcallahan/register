package sheets_service

import (
	"reflect"
	"register/pkg/config"
	"register/pkg/models"
	"testing"

	"google.golang.org/api/sheets/v4"
)

func Test_monthlyService_updateMonthly(t *testing.T) {
	type fields struct {
		service       *sheets.Service
		SpreadsheetID string
		BudgetSheet   *BudgetSheet
		RegisterSheet *RegisterSheet
	}
	type args struct {
		sheetID int64
		rows    []*sheets.RowData
	}
	var tests []struct {
		name   string
		fields fields
		args   args
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			//ss := &sheetsService{
			//	service:       tt.fields.service,
			//	BudgetSheet:   tt.fields.BudgetSheet,
			//	RegisterSheet: tt.fields.RegisterSheet,
			//}
		})
	}
}

func Test_monthlyService_addSummarySalaryRow(t *testing.T) {
	months := []string{"Jan", "Feb", "Mar"}

	// the full want is name cell, three month cells, this SUM cell, and the AVERAGE cell.
	nameCell := mkCellDataString("CrowdStrike Salary", "right", models.ColorGrey, false)
	janCell := mkCellDataDollars(10.0, "right", models.ColorGrey, false)
	febCell := mkCellDataDollars(20.0, "right", models.ColorGrey, false)
	marCell := mkCellDataDollars(30.0, "right", models.ColorGrey, false)
	sumCell := mkCellDataFormula("=SUM(B10:D10) * -1", "right", models.ColorGrey, false)
	avgCell := mkCellDataFormula("=AVERAGE(B10:D10) * -1", "right", models.ColorGrey, false)
	want := &sheets.RowData{
		Values: []*sheets.CellData{nameCell, janCell, febCell, marCell, sumCell, avgCell},
	}

	aggData := make(map[string]map[string]float64)
	aggData["Jan"] = make(map[string]float64)
	aggData["Jan"]["CrowdStrike Salary"] = 10.00
	aggData["Feb"] = make(map[string]float64)
	aggData["Feb"]["CrowdStrike Salary"] = 20.00
	aggData["Mar"] = make(map[string]float64)
	aggData["Mar"]["CrowdStrike Salary"] = 30.00

	type args struct {
		rNum     int
		cellName string
		months   []string
		aggData  map[string]map[string]float64
	}
	tests := []struct {
		name string
		args args
		want *sheets.RowData
	}{
		{
			name: "Test add salary row",
			args: args{cellName: "CrowdStrike Salary", rNum: 10, months: months, aggData: aggData},
			want: want,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := addSummarySalaryRow(tt.args.cellName, tt.args.rNum, tt.args.months, tt.args.aggData)
			if len(got.Values) != 6 {
				t.Errorf("addSummarySalaryRow() len(got) = %v, want %v", len(got.Values), 6)
			}
			name := "CrowdStrike Salary"
			w := &sheets.ExtendedValue{StringValue: &name}
			if *got.Values[0].UserEnteredValue.StringValue != *w.StringValue {
				t.Errorf("addSummarySalaryRow() *got.Values[0].UserEnteredValue.StringValue = %v, want %v", *got.Values[0].UserEnteredValue.StringValue, name)
			}
			for idx := 1; idx <= 3; idx++ {
				if *got.Values[idx].UserEnteredValue.NumberValue != *want.Values[idx].UserEnteredValue.NumberValue {
					t.Errorf("addSummarySalaryRow() *got.Values[%d].UserEnteredValue.NumberValue = %v, want %v",
						idx, *got.Values[idx].UserEnteredValue.NumberValue, *want.Values[idx].UserEnteredValue.NumberValue)
				}
			}
			for idx := 4; idx <= 5; idx++ {
				if *got.Values[idx].UserEnteredValue.FormulaValue != *want.Values[idx].UserEnteredValue.FormulaValue {
					t.Errorf("addSummarySalaryRow() *got.Values[%d].UserEnteredValue.FormulaValue = %v, want %v",
						idx, *got.Values[idx].UserEnteredValue.FormulaValue, *want.Values[idx].UserEnteredValue.FormulaValue)
				}
			}
		})
	}
}

func Test_monthlyService_addSummaryTopRow(t *testing.T) {
	months := []string{"Jan", "Feb", "Mar"}
	catCell := mkCellDataString("Category", "right", models.ColorGrey, false)
	janCell := mkCellDataString("Jan", "right", models.ColorGrey, false)
	febCell := mkCellDataString("Feb", "right", models.ColorGrey, false)
	marCell := mkCellDataString("Mar", "right", models.ColorGrey, false)
	want := &sheets.RowData{
		Values: []*sheets.CellData{catCell, janCell, febCell, marCell},
	}

	type args struct {
		months []string
	}
	tests := []struct {
		name string
		args args
		want *sheets.RowData
	}{
		{
			name: "Test add summary row",
			args: args{months: months},
			want: want,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := addSummaryTopRow(tt.args.months)
			for idx := range 4 {
				if *got.Values[idx].UserEnteredValue.StringValue != *want.Values[idx].UserEnteredValue.StringValue {
					t.Errorf("addSummaryTopRow() *got.Values[%d].UserEnteredValue.StringValue = %v, want %v",
						idx, *got.Values[idx].UserEnteredValue.StringValue, *want.Values[idx].UserEnteredValue.StringValue)
				}
			}

		})
	}
}

func Test_monthlyService_readRangeFormulas(t *testing.T) {
	getFormulaProviderFunc = func(range_ string) ([][]any, error) {
		values := make([][]any, 2)
		values[0] = make([]any, 2)
		values[0][0] = "=A1*2"
		values[0][1] = "=A2*5"
		return values, nil
	}

	want := []string{"=A1*2", "=A2*5"}

	type args struct {
		readRange string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "Test reading a string cell",
			args: args{
				readRange: "A1:A2",
			},
			want: want,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, _ := ss.readRangeFormulas(tt.args.readRange); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("readRangeFormulas() = %v, want %v", got, tt.want)
			}
		})
	}

	getFormulaProviderFunc = func(range_ string) ([][]any, error) {
		values := make([][]any, 10)
		values[0] = make([]any, 10)
		values[0][0] = "=SUM(A1:A2)"
		return values, nil
	}
}

func Test_monthlyService_UpdateMonthlyCategories(t *testing.T) {
	ms := NewMonthlyService(&providerMock{}, nil)
	catAgg := map[string]map[string]float64{
		"01/26": {"Grocery": 50},
	}
	columns := []models.Column{
		{Name: "Check", ColumnIndex: 2},
		{Name: "Grocery", ColumnIndex: 11, Color: models.ColorBlue},
		{Name: "Rent", ColumnIndex: 12, Color: models.ColorBlue},
	}
	if err := ms.UpdateMonthlyCategories("Budget", catAgg, columns); err != nil {
		t.Errorf("UpdateMonthlyCategories() error = %v, want nil", err)
	}
	if err := ms.UpdateMonthlyCategories("Nope", catAgg, columns); err == nil {
		t.Errorf("UpdateMonthlyCategories() error = nil, want error for unknown tab")
	}
}

func Test_monthlyService_UpdateMonthlyPayees(t *testing.T) {
	ms := NewMonthlyService(&providerMock{}, []config.IncomeSource{
		{Name: "CrowdStrike Salary", Frequency: "monthly"},
	})
	payeeAgg := map[string]map[string]float64{
		"01/26": {"TARGET": 20, "CrowdStrike Salary": 100},
	}
	if err := ms.UpdateMonthlyPayees("Budget", payeeAgg); err != nil {
		t.Errorf("UpdateMonthlyPayees() error = %v, want nil", err)
	}
	if err := ms.UpdateMonthlyPayees("Nope", payeeAgg); err == nil {
		t.Errorf("UpdateMonthlyPayees() error = nil, want error for unknown tab")
	}
}

func Test_monthlyService_updateMonthlyExecutes(t *testing.T) {
	ms := NewMonthlyService(&providerMock{}, nil)
	rows := []*sheets.RowData{
		{Values: []*sheets.CellData{mkCellDataString("x", "left", "", false)}},
	}
	if err := ms.updateMonthly("Budget", 987654321, rows); err != nil {
		t.Errorf("updateMonthly() error = %v, want nil", err)
	}
	if err := ms.updateMonthly("Budget", 987654321, nil); err != nil {
		t.Errorf("updateMonthly(nil rows) error = %v, want nil", err)
	}
}

func Test_populateMonthlyCategories(t *testing.T) {
	catAgg := map[string]map[string]float64{
		"02/26": {"Rent": 30},
		"01/26": {"Grocery": 10, "Rent": 20},
	}
	columns := []models.Column{
		{Name: "Check", ColumnIndex: 2},
		{Name: "Grocery", ColumnIndex: 11, Color: models.ColorBlue},
		{Name: "Rent", ColumnIndex: 12, Color: models.ColorBlue},
	}
	rows := populateMonthlyCategories(catAgg, columns)
	if len(rows) != 3 {
		t.Fatalf("len(rows) = %d, want 3 (top + 2 categories)", len(rows))
	}
	top := rows[0]
	if len(top.Values) != 5 {
		t.Fatalf("top row len = %d, want 5", len(top.Values))
	}
	wantTop := []string{"Category", "01/26", "02/26", "Yearly Totals", "Monthly Average"}
	for i, want := range wantTop {
		if i >= len(top.Values) {
			break
		}
		if got := *top.Values[i].UserEnteredValue.StringValue; got != want {
			t.Errorf("top cell[%d] = %q, want %q", i, got, want)
		}
	}
	// Category rows follow column order, not alphabetical order.
	grocery := rows[1]
	if got := *grocery.Values[0].UserEnteredValue.StringValue; got != "Grocery" {
		t.Errorf("row[1] name = %q, want Grocery", got)
	}
	if got := *grocery.Values[1].UserEnteredValue.NumberValue; got != 10 {
		t.Errorf("Grocery 01/26 = %v, want 10", got)
	}
	if got := *grocery.Values[2].UserEnteredValue.NumberValue; got != 0 {
		t.Errorf("Grocery 02/26 = %v, want 0", got)
	}
	if got := *grocery.Values[3].UserEnteredValue.FormulaValue; got != "=SUM(B2:C2) * -1" {
		t.Errorf("Grocery totals formula = %q, want =SUM(B2:C2) * -1", got)
	}
	if got := *grocery.Values[4].UserEnteredValue.FormulaValue; got != "=AVERAGE(B2:C2) * -1" {
		t.Errorf("Grocery average formula = %q, want =AVERAGE(B2:C2) * -1", got)
	}
	rent := rows[2]
	if got := *rent.Values[1].UserEnteredValue.NumberValue; got != 20 {
		t.Errorf("Rent 01/26 = %v, want 20", got)
	}
	if got := *rent.Values[2].UserEnteredValue.NumberValue; got != 30 {
		t.Errorf("Rent 02/26 = %v, want 30", got)
	}
}

func Test_populateMonthlyPayees(t *testing.T) {
	payeeAgg := map[string]map[string]float64{
		"01/26": {"TARGET": 5, "AMAZON": 7},
		"02/26": {"TARGET": 3},
	}
	rows := populateMonthlyPayees(payeeAgg, []string{"CrowdStrike Salary"})
	if len(rows) != 3 {
		t.Fatalf("len(rows) = %d, want 3 (top + 2 payees)", len(rows))
	}
	// Payees are sorted across all months, and the absent income source
	// adds no salary row.
	if got := *rows[1].Values[0].UserEnteredValue.StringValue; got != "AMAZON" {
		t.Errorf("row[1] name = %q, want AMAZON", got)
	}
	if got := *rows[1].Values[1].UserEnteredValue.NumberValue; got != 7 {
		t.Errorf("AMAZON 01/26 = %v, want 7", got)
	}
	if got := *rows[1].Values[2].UserEnteredValue.NumberValue; got != 0 {
		t.Errorf("AMAZON 02/26 = %v, want 0", got)
	}
	if got := *rows[2].Values[0].UserEnteredValue.StringValue; got != "TARGET" {
		t.Errorf("row[2] name = %q, want TARGET", got)
	}

	withIncome := map[string]map[string]float64{
		"01/26": {"TARGET": 5, "CrowdStrike Salary": 100},
	}
	rows = populateMonthlyPayees(withIncome, []string{"CrowdStrike Salary"})
	// The income name appears twice by design: once as a payee row (it is
	// a payee in the aggregate) and once as the salary row appended for
	// incomeNames.
	if len(rows) != 4 {
		t.Fatalf("len(rows) with income = %d, want 4 (top + 2 payee + salary)", len(rows))
	}
	if got := *rows[1].Values[0].UserEnteredValue.StringValue; got != "CrowdStrike Salary" {
		t.Errorf("payee row name = %q, want CrowdStrike Salary", got)
	}
	if got := *rows[2].Values[0].UserEnteredValue.StringValue; got != "TARGET" {
		t.Errorf("payee row name = %q, want TARGET", got)
	}
	if got := *rows[3].Values[0].UserEnteredValue.StringValue; got != "CrowdStrike Salary" {
		t.Errorf("salary row name = %q, want CrowdStrike Salary", got)
	}
	if got := *rows[3].Values[1].UserEnteredValue.NumberValue; got != 100 {
		t.Errorf("salary 01/26 = %v, want 100", got)
	}
}

func Test_addSummaryRows(t *testing.T) {
	aggData := map[string]map[string]float64{
		"01/26": {"Grocery": 42, "Pay": 100},
	}
	rows := addSummaryRows(nil, aggData, []string{"01/26"}, []string{"Grocery"}, nil)
	if len(rows) != 1 {
		t.Fatalf("len(rows) = %d, want 1", len(rows))
	}
	row := rows[0]
	if len(row.Values) != 4 {
		t.Fatalf("len(row cells) = %d, want 4 (name + month + totals + average)", len(row.Values))
	}
	if got := *row.Values[0].UserEnteredValue.StringValue; got != "Grocery" {
		t.Errorf("name cell = %q, want Grocery", got)
	}
	if got := *row.Values[1].UserEnteredValue.NumberValue; got != 42 {
		t.Errorf("month cell = %v, want 42", got)
	}
	if got := *row.Values[2].UserEnteredValue.FormulaValue; got != "=SUM(B2:B2) * -1" {
		t.Errorf("totals formula = %q, want =SUM(B2:B2) * -1", got)
	}
	if got := *row.Values[3].UserEnteredValue.FormulaValue; got != "=AVERAGE(B2:B2) * -1" {
		t.Errorf("average formula = %q, want =AVERAGE(B2:B2) * -1", got)
	}

	withIncome := addSummaryRows(nil, aggData, []string{"01/26"}, []string{"Grocery"}, []string{"Pay"})
	if len(withIncome) != 2 {
		t.Fatalf("len(rows) with income = %d, want 2", len(withIncome))
	}
	if got := *withIncome[1].Values[0].UserEnteredValue.StringValue; got != "Pay" {
		t.Errorf("salary row name = %q, want Pay", got)
	}

	absentIncome := addSummaryRows(nil, aggData, []string{"01/26"}, []string{"Grocery"}, []string{"Missing"})
	if len(absentIncome) != 1 {
		t.Errorf("len(rows) with absent income = %d, want 1", len(absentIncome))
	}
}

func Test_aggDataHasName(t *testing.T) {
	tests := []struct {
		name    string
		aggData map[string]map[string]float64
		want    bool
	}{
		{"present in one month", map[string]map[string]float64{"01/26": {"Grocery": 1}}, true},
		{"present in a later month", map[string]map[string]float64{"01/26": {}, "02/26": {"Grocery": 1}}, true},
		{"absent", map[string]map[string]float64{"01/26": {"Rent": 1}}, false},
		{"empty map", map[string]map[string]float64{}, false},
		{"nil map", nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := aggDataHasName(tt.aggData, "Grocery"); got != tt.want {
				t.Errorf("aggDataHasName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_monthlyService_UpdateYearlySummary(t *testing.T) {
	ms := NewMonthlyService(&providerMock{}, nil)
	columns := []models.Column{
		{Name: "Grocery", ColumnIndex: 11, Color: models.ColorBlue},
		{Name: "Rent", ColumnIndex: 12, Color: models.ColorBlue},
	}
	catAgg := map[string]map[string]float64{
		"01/26": {"Grocery": 10},
	}
	if err := ms.UpdateYearlySummary("Budget", catAgg, columns); err != nil {
		t.Errorf("UpdateYearlySummary() error = %v, want nil", err)
	}
	if err := ms.UpdateYearlySummary("Nope", catAgg, columns); err == nil {
		t.Errorf("UpdateYearlySummary() error = nil, want error for unknown tab")
	}
}

func Test_monthlyService_ensureColumnCount(t *testing.T) {
	ms := NewMonthlyService(&providerMock{}, nil)
	if err := ms.ensureColumnCount(1, 10, 4); err != nil {
		t.Errorf("ensureColumnCount(10, 4) error = %v, want nil", err)
	}
	if err := ms.ensureColumnCount(1, 3, 10); err != nil {
		t.Errorf("ensureColumnCount(3, 10) error = %v, want nil", err)
	}
}

func Test_populateYearlySummary(t *testing.T) {
	catAgg := map[string]map[string]float64{
		"01/26": {"Grocery": 10, "Rent": 20},
		"02/26": {"Grocery": 5},
	}
	columns := []models.Column{
		{Name: "Grocery", ColumnIndex: 11, Color: models.ColorBlue},
		{Name: "Rent", ColumnIndex: 12, Color: models.ColorBlue},
	}
	rows := populateYearlySummary(catAgg, columns, []string{"Pay"})
	if len(rows) != 4 {
		t.Fatalf("len(rows) = %d, want 4 (top + 2 months + totals)", len(rows))
	}
	top := rows[0]
	wantTop := []string{"Month", "Grocery", "Rent", "Pay", "Total"}
	if len(top.Values) != len(wantTop) {
		t.Fatalf("top row len = %d, want %d", len(top.Values), len(wantTop))
	}
	for i, want := range wantTop {
		if got := *top.Values[i].UserEnteredValue.StringValue; got != want {
			t.Errorf("top cell[%d] = %q, want %q", i, got, want)
		}
	}
	jan := rows[1]
	if got := *jan.Values[0].UserEnteredValue.StringValue; got != "01/26" {
		t.Errorf("month row name = %q, want 01/26", got)
	}
	if got := *jan.Values[1].UserEnteredValue.NumberValue; got != 10 {
		t.Errorf("01/26 Grocery = %v, want 10", got)
	}
	if got := *jan.Values[2].UserEnteredValue.NumberValue; got != 20 {
		t.Errorf("01/26 Rent = %v, want 20", got)
	}
	if got := *jan.Values[3].UserEnteredValue.NumberValue; got != 0 {
		t.Errorf("01/26 Pay = %v, want 0", got)
	}
	if got := *jan.Values[4].UserEnteredValue.FormulaValue; got != "=SUM(B2:D2)" {
		t.Errorf("01/26 total formula = %q, want =SUM(B2:D2)", got)
	}
	totals := rows[3]
	if got := *totals.Values[0].UserEnteredValue.StringValue; got != "Total" {
		t.Errorf("totals row name = %q, want Total", got)
	}
	if got := *totals.Values[1].UserEnteredValue.FormulaValue; got != "=SUM(B2:B3)" {
		t.Errorf("totals Grocery formula = %q, want =SUM(B2:B3)", got)
	}
	if got := *totals.Values[4].UserEnteredValue.FormulaValue; got != "=SUM(B4:D4)" {
		t.Errorf("totals row formula = %q, want =SUM(B4:D4)", got)
	}
}

func Test_addYearlyTopRow(t *testing.T) {
	got := addYearlyTopRow([]string{"Grocery", "Rent"})
	want := []string{"Month", "Grocery", "Rent", "Total"}
	if len(got.Values) != len(want) {
		t.Fatalf("len(cells) = %d, want %d", len(got.Values), len(want))
	}
	for i, w := range want {
		if got := *got.Values[i].UserEnteredValue.StringValue; got != w {
			t.Errorf("cell[%d] = %q, want %q", i, got, w)
		}
	}
	empty := addYearlyTopRow(nil)
	if len(empty.Values) != 2 {
		t.Errorf("len(cells) with no headers = %d, want 2 (Month + Total)", len(empty.Values))
	}
}
