package sheets_service

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"reflect"
	"testing"

	"register/pkg/models"

	"google.golang.org/api/sheets/v4"
)

const sheetsServiceJSONDir = "/Users/rob/ws/go/src/register/api/services/sheets_service/json/"

var months = []string{"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}

type providerMock struct {
	service       *sheets.Service
	spreadsheetID string
}

var getValuesProviderFunc = func(range_ string) (*sheets.ValueRange, error) {
	vr := &sheets.ValueRange{Values: [][]interface{}{}}
	vr.Values = make([][]interface{}, 10)
	vr.Values[0] = make([]interface{}, 10)
	vr.Values[0][0] = "a string"
	return vr, nil
}
var getFormulaProviderFunc = func(range_ string) (*sheets.ValueRange, error) {
	vr := &sheets.ValueRange{Values: [][]interface{}{}}
	vr.Values = make([][]interface{}, 10)
	vr.Values[0] = make([]interface{}, 10)
	vr.Values[0][0] = "=SUM(A1:A2)"
	return vr, nil
}

func (p *providerMock) GetValues(range_ string) (*sheets.ValueRange, error) {
	return getValuesProviderFunc(range_)
}

func (p *providerMock) GetFormula(range_ string) (*sheets.ValueRange, error) {
	return getFormulaProviderFunc(range_)
}

func (p *providerMock) GetSpreadsheet() (*sheets.Spreadsheet, error) {
	var s sheets.Spreadsheet

	j, err := os.ReadFile(sheetsServiceJSONDir + "spreadsheet.json")
	if err != nil {
		return nil, errors.New(fmt.Sprintf("could not read json file: %s, %s", sheetsServiceJSONDir+"spreadsheet.json", err.Error()))
	}
	err = json.Unmarshal(j, &s)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("could not unmarshal JSON: %s", err.Error()))
	}
	return &s, nil
}

func (p *providerMock) BatchUpdate(updateReq *sheets.BatchUpdateSpreadsheetRequest) (*sheets.BatchUpdateSpreadsheetResponse, error) {
	return &sheets.BatchUpdateSpreadsheetResponse{}, nil
}

func (p *providerMock) Update(writeRange string, vRange *sheets.ValueRange) (*sheets.UpdateValuesResponse, error) {
	w, err := os.ReadFile(sheetsServiceJSONDir + "WriteCell.json")
	if err != nil {
		log.Fatalf("error: %s\n", err.Error())
	}
	var resp *sheets.UpdateValuesResponse
	err = json.Unmarshal(w, &resp)
	if err != nil {
		log.Fatalf("error: %s\n", err.Error())
	}
	return resp, nil
}

var ss *SheetsService

func init() {
	var err error

	c, err := os.ReadFile("json/categoriesMap.json")
	if err != nil {
		log.Fatalf("error: %s\n", err.Error())
	}
	var entries map[string]*BudgetEntry
	err = json.Unmarshal(c, &entries)
	if err != nil {
		log.Fatalf("error: %s\n", err.Error())
	}

	provider := providerMock{
		service:       &sheets.Service{},
		spreadsheetID: "ssID",
	}
	ss = New(&provider)
	if err != nil {
		fmt.Printf("%s", err.Error())
		return
	}
	ss.RegisterSheet = &RegisterSheet{
		TabName: "Register",
	}
	ss.BudgetSheet = &BudgetSheet{
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

func Test_sheetsService_copyRowsBatchUpdateRequest(t *testing.T) {
	w, err := os.ReadFile(sheetsServiceJSONDir + "copyRowsBatchUpdateRequest.json")
	checkTestingError(t, err)
	var want sheets.BatchUpdateSpreadsheetRequest
	err = json.Unmarshal(w, &want)
	checkTestingError(t, err)

	type args struct {
		numCopies int
	}
	tests := []struct {
		name string
		args args
		want sheets.BatchUpdateSpreadsheetRequest
	}{
		{
			name: "Test copy rows batch update request",
			args: args{numCopies: 5},
			want: want,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ss.copyRowsBatchUpdateRequest(tt.args.numCopies); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("copyRowsBatchUpdateRequest() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_sheetsService_ReadBudgetSheet(t *testing.T) {
	w, err := os.ReadFile(sheetsServiceJSONDir + "ReadBudgetSheet.json")
	checkTestingError(t, err)
	var want BudgetSheet
	err = json.Unmarshal(w, &want)
	checkTestingError(t, err)

	getValuesProviderFunc = func(range_ string) (*sheets.ValueRange, error) {
		r, err := os.ReadFile(sheetsServiceJSONDir + "BudgetValues.json")
		checkTestingError(t, err)
		var valueRange sheets.ValueRange
		err = json.Unmarshal(r, &valueRange)
		checkTestingError(t, err)
		return &valueRange, nil
	}

	tests := []struct {
		name    string
		want    *BudgetSheet
		wantErr bool
	}{
		{
			name:    "Test read budget sheet",
			want:    &want,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, err := ss.ReadBudgetSheet(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ReadBudgetSheet() = %v, want %v", got, tt.want)
				t.Errorf("ReadBudgetSheet() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_sheetsService_ReadRegisterSheet(t *testing.T) {
	w, err := os.ReadFile(sheetsServiceJSONDir + "ReadRegisterSheet.json")
	checkTestingError(t, err)
	var want RegisterSheet
	err = json.Unmarshal(w, &want)
	checkTestingError(t, err)

	getValuesProviderFunc = func(range_ string) (*sheets.ValueRange, error) {
		r, err := os.ReadFile(sheetsServiceJSONDir + "RegisterValues.json")
		checkTestingError(t, err)
		var valueRange sheets.ValueRange
		err = json.Unmarshal(r, &valueRange)
		checkTestingError(t, err)
		return &valueRange, nil
	}

	tests := []struct {
		name    string
		want    *RegisterSheet
		wantErr bool
	}{
		{
			name:    "Test read register sheet",
			want:    &want,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, err := ss.ReadRegisterSheet(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ReadRegisterSheet() = %v, want %v", got, tt.want)
				t.Errorf("ReadRegisterSheet() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_sheetsService_WriteCell(t *testing.T) {
	w, err := os.ReadFile(sheetsServiceJSONDir + "WriteCell.json")
	checkTestingError(t, err)
	var want *sheets.UpdateValuesResponse
	err = json.Unmarshal(w, &want)
	checkTestingError(t, err)

	type args struct {
		cell  string
		value interface{}
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
			if got, _ := ss.WriteCell(tt.args.cell, tt.args.value); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("WriteCell() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_sheetsService_addAmountCell(t *testing.T) {
	a, err := os.ReadFile(sheetsServiceJSONDir + "addAmountCell.json")
	checkTestingError(t, err)
	var want []*sheets.CellData
	err = json.Unmarshal(a, &want)
	checkTestingError(t, err)

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
			name: "Test add amount cells",
			args: args{
				cells:   []*sheets.CellData{},
				trans:   &models.Transaction{},
				bgColor: "white",
			},
			want: want,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := addAmountCell(tt.args.cells, tt.args.trans, tt.args.bgColor); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("addAmountCell() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_sheetsService_addCategoryCells(t *testing.T) {
	c, err := os.ReadFile(sheetsServiceJSONDir + "columns.json")
	checkTestingError(t, err)
	var cols []models.Column
	err = json.Unmarshal(c, &cols)
	checkTestingError(t, err)

	n, err := os.ReadFile(sheetsServiceJSONDir + "transNameToColName.json")
	checkTestingError(t, err)
	var name2Col map[string]string
	err = json.Unmarshal(n, &name2Col)
	checkTestingError(t, err)

	a, err := os.ReadFile(sheetsServiceJSONDir + "addCategoryCells.json")
	checkTestingError(t, err)
	var want []*sheets.CellData
	err = json.Unmarshal(a, &want)
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
			name: "Test add category cells",
			args: args{
				cells:              []*sheets.CellData{},
				trans:              &models.Transaction{},
				columns:            cols,
				transNameToColName: name2Col,
				totalsFormulas: []string{
					"=A1+B1",
					"=C1+D1",
					"=E1+F1",
				},
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

func Test_sheetsService_addSalaryCells(t *testing.T) {
	c, err := os.ReadFile("json/columns.json")
	checkTestingError(t, err)
	var cols []models.Column
	err = json.Unmarshal(c, &cols)
	checkTestingError(t, err)

	w, err := os.ReadFile("json/addSalaryCells.json")
	checkTestingError(t, err)
	var want []*sheets.CellData
	err = json.Unmarshal(w, &want)
	checkTestingError(t, err)

	type args struct {
		cells          []*sheets.CellData
		columns        []models.Column
		totalsFormulas []string
	}
	tests := []struct {
		name string
		args args
		want []*sheets.CellData
	}{
		{
			name: "Test add salary cells",
			args: args{
				cells:   []*sheets.CellData{},
				columns: cols,
				totalsFormulas: []string{
					"=A1+B1",
					"=C1+D1",
					"=E1+F1",
				},
			},
			want: want,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ss.addSalaryCells(tt.args.cells, tt.args.columns, tt.args.totalsFormulas); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("addSalaryCells() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_sheetsService_addSourceDateNameCells(t *testing.T) {
	j, err := os.ReadFile(sheetsServiceJSONDir + "addSourceDataNameCells.json")
	if err != nil {
		t.Errorf("could not read JSON file: %s\n", err.Error())
	}
	var want []*sheets.CellData
	err = json.Unmarshal(j, &want)
	if err != nil {
		t.Errorf("could not unmarshal JSON: %s\n", err.Error())
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
			name: "Test add source data name cells",
			args: args{
				cells: []*sheets.CellData{},
				trans: &models.Transaction{
					Source: "Fidelity",
					Date:   "01/02/03",
					Name:   "Amazon",
				},
				bgColor: "white",
			},
			want: want,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, _ := addSourceDateNameCells(tt.args.cells, tt.args.trans, tt.args.bgColor); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("addSourceDateNameCells() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_sheetsService_getSheetID(t *testing.T) {
	type args struct {
		tabName string
	}
	tests := []struct {
		name    string
		args    args
		want    int64
		wantErr bool
	}{
		{
			name:    "Test get spreadsheet ID",
			args:    args{tabName: "Register"},
			want:    617336355,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ss.getSheetID(tt.args.tabName)
			if (err != nil) != tt.wantErr {
				t.Errorf("getSheetID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("getSheetID() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_sheetsService_populateCells(t *testing.T) {
	type fields struct {
		service       *sheets.Service
		SpreadsheetID string
		BudgetSheet   *BudgetSheet
		RegisterSheet *RegisterSheet
		Debug         bool
		Verbose       bool
	}
	type args struct {
		columns            []models.Column
		transNameToColName map[string]string
		transactions       []*models.Transaction
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   []*sheets.RowData
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ss := &SheetsService{
				service:       tt.fields.service,
				SpreadsheetID: tt.fields.SpreadsheetID,
				BudgetSheet:   tt.fields.BudgetSheet,
				RegisterSheet: tt.fields.RegisterSheet,
				Debug:         tt.fields.Debug,
				Verbose:       tt.fields.Debug,
			}
			if got, _ := ss.populateCells(tt.args.columns, tt.args.transNameToColName, tt.args.transactions); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("populateCells() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_sheetsService_updateMonthly(t *testing.T) {
	type fields struct {
		service       *sheets.Service
		SpreadsheetID string
		BudgetSheet   *BudgetSheet
		RegisterSheet *RegisterSheet
		Debug         bool
		Verbose       bool
	}
	type args struct {
		sheetID int64
		rows    []*sheets.RowData
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			//ss := &sheetsService{
			//	service:       tt.fields.service,
			//	SpreadsheetID: tt.fields.SpreadsheetID,
			//	BudgetSheet:   tt.fields.BudgetSheet,
			//	RegisterSheet: tt.fields.RegisterSheet,
			//	Debug:         tt.fields.Debug,
			//	Verbose:       tt.fields.Debug,
			//}
		})
	}
}

func Test_sheetsService_readRangeFormulas(t *testing.T) {
	getFormulaProviderFunc = func(range_ string) (*sheets.ValueRange, error) {
		vr := &sheets.ValueRange{Values: [][]interface{}{}}
		vr.Values = make([][]interface{}, 2)
		vr.Values[0] = make([]interface{}, 2)
		vr.Values[0][0] = "=A1*2"
		vr.Values[0][1] = "=A2*5"
		return vr, nil
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
			if got := ss.readRangeFormulas(tt.args.readRange); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("readRangeFormulas() = %v, want %v", got, tt.want)
			}
		})
	}

	getFormulaProviderFunc = func(range_ string) (*sheets.ValueRange, error) {
		vr := &sheets.ValueRange{Values: [][]interface{}{}}
		vr.Values = make([][]interface{}, 10)
		vr.Values[0] = make([]interface{}, 10)
		vr.Values[0][0] = "=SUM(A1:A2)"
		return vr, nil
	}
}

func Test_sheetsService_getAmountString(t *testing.T) {
	var (
		values1 []interface{}
		values2 []interface{}
		values3 []interface{}
	)
	values1 = make([]interface{}, CreditCards+1)
	values2 = make([]interface{}, CreditCards+1)
	values3 = make([]interface{}, CreditCards+1)
	for i := 0; i <= CreditCards; i++ {
		values1[i] = ""
		values2[i] = ""
		values3[i] = ""
	}

	type args struct {
		values []interface{}
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
		want: "-10.00",
	})
	values2[Deposits] = "$ 20.00 "
	tests = append(tests, ts{
		name: "Test getting the total Amount - Deposit",
		args: args{values: values2},
		want: "20.00",
	})
	values3[CreditCards] = "$ 30.00 "
	tests = append(tests, ts{
		name: "Test getting the total Amount - Credit Card",
		args: args{values: values3},
		want: "-30.00",
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getAmountString(tt.args.values); got != tt.want {
				t.Errorf("getAmountString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_sheetsService_GetDollarsCellByIndex(t *testing.T) {
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
			name: "Test positive dollars",
			args: args{values: []interface{}{"$ 10.20 "}, i: 0},
			want: 10.20,
		},
		{
			name: "Test negative dollars",
			args: args{values: []interface{}{"$ (10.20) "}, i: 0},
			want: -10.20,
		},
		{
			name: "Test for dash",
			args: args{values: []interface{}{" - "}, i: 0},
			want: 0,
		},
		{
			name: "Test empty cell",
			args: args{values: []interface{}{""}, i: 0},
			want: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getDollarsCellByIndex(tt.args.values, tt.args.i); got != tt.want {
				t.Errorf("GetDollarsCellByIndex() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_sheetsService_getDateField(t *testing.T) {
	var values []interface{}
	var values2 []interface{}
	values = make([]interface{}, Date+1)
	values2 = make([]interface{}, Date+1)
	for i := 0; i <= Date; i++ {
		values[i] = "x"
		values2[i] = "x"
	}
	values[Date] = "01/02/2020"
	values2[Date] = ""

	type args struct {
		values []interface{}
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

func Test_sheetsService_getNameField(t *testing.T) {
	var values []interface{}
	values = make([]interface{}, Description+1)
	for i := 0; i <= Description; i++ {
		values[i] = "x"
	}
	values[Description] = "merchant"

	type args struct {
		values []interface{}
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

func Test_sheetsService_ReadStringCell(t *testing.T) {
	getValuesProviderFunc = func(range_ string) (*sheets.ValueRange, error) {
		vr := &sheets.ValueRange{Values: [][]interface{}{}}
		vr.Values = make([][]interface{}, 10)
		vr.Values[0] = make([]interface{}, 10)
		vr.Values[0][0] = "a string"
		return vr, nil
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
			if got, _ := ss.ReadStringCell(tt.args.cell); got != tt.want {
				t.Errorf("ReadStringCell() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_sheetsService_ReadDateCell(t *testing.T) {
	getValuesProviderFunc = func(range_ string) (*sheets.ValueRange, error) {
		vr := &sheets.ValueRange{Values: [][]interface{}{}}
		vr.Values = make([][]interface{}, 10)
		vr.Values[0] = make([]interface{}, 10)
		vr.Values[0][0] = "01/02/20"
		return vr, nil
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
			if got, _ := ss.ReadDateCell(tt.args.cell); got != tt.want {
				t.Errorf("ReadDateCell() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_sheetsService_ReadDollarsCell(t *testing.T) {
	getValuesProviderFunc = func(range_ string) (*sheets.ValueRange, error) {
		vr := &sheets.ValueRange{Values: [][]interface{}{}}
		vr.Values = make([][]interface{}, 10)
		vr.Values[0] = make([]interface{}, 10)
		vr.Values[0][0] = " $  20.20 "
		return vr, nil
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
			if got, _ := ss.ReadDollarsCell(tt.args.cell); got != tt.want {
				t.Errorf("ReadDollarsCell() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_sheetsService_ReadFormulaCell(t *testing.T) {
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
			if got, _ := ss.ReadFormulaCell(tt.args.cell); got != tt.want {
				t.Errorf("ReadFormulaCell() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_sheetsService_readCell(t *testing.T) {
	getValuesProviderFunc = func(range_ string) (*sheets.ValueRange, error) {
		vr := &sheets.ValueRange{Values: [][]interface{}{}}
		vr.Values = make([][]interface{}, 10)
		vr.Values[0] = make([]interface{}, 10)
		vr.Values[0][0] = "a string"
		return vr, nil
	}
	getFormulaProviderFunc = func(range_ string) (*sheets.ValueRange, error) {
		vr := &sheets.ValueRange{Values: [][]interface{}{}}
		vr.Values = make([][]interface{}, 10)
		vr.Values[0] = make([]interface{}, 10)
		vr.Values[0][0] = "=SUM(A1:A2)"
		return vr, nil
	}

	type args struct {
		cell  string
		dType CellDataType
	}
	tests := []struct {
		name string
		args args
		want interface{}
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
			if got, _ := ss.ReadCell(tt.args.cell, tt.args.dType); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ReadCell() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_addSummaryRows(t *testing.T) {
	b, err := os.ReadFile("json/cat_agg.json")
	checkTestingError(t, err)

	var catAgg map[string]map[string]float64
	err = json.Unmarshal(b, &catAgg)
	checkTestingError(t, err)

	c, err := os.ReadFile("json/column_names.json")
	checkTestingError(t, err)

	var cNames []string
	err = json.Unmarshal(c, &cNames)
	checkTestingError(t, err)

	w, err := os.ReadFile("json/addSummaryRows.json")
	checkTestingError(t, err)

	var want []*sheets.RowData
	err = json.Unmarshal(w, &want)
	checkTestingError(t, err)

	var rows []*sheets.RowData

	type args struct {
		rows    []*sheets.RowData
		aggData map[string]map[string]float64
		months  *[]string
		cats    *[]string
	}
	tests := []struct {
		name string
		args args
		want []*sheets.RowData
	}{
		{
			name: "Test populate monthly categories",
			args: args{rows: rows, aggData: catAgg, months: &months, cats: &cNames},
			want: want,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := addSummaryRows(tt.args.rows, tt.args.aggData, tt.args.months, tt.args.cats); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("addSummaryRows() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_populateMonthlyCategories(t *testing.T) {
	b, err := os.ReadFile("json/cat_agg.json")
	checkTestingError(t, err)

	var catAgg map[string]map[string]float64
	err = json.Unmarshal(b, &catAgg)
	checkTestingError(t, err)

	c, err := os.ReadFile("json/columns.json")
	checkTestingError(t, err)

	var cols []models.Column
	err = json.Unmarshal(c, &cols)
	checkTestingError(t, err)

	w, err := os.ReadFile("json/populateMonthlyCategories.json")
	checkTestingError(t, err)

	var want []*sheets.RowData
	err = json.Unmarshal(w, &want)
	checkTestingError(t, err)

	type args struct {
		catAgg map[string]map[string]float64
		cats   []models.Column
	}
	tests := []struct {
		name string
		args args
		want []*sheets.RowData
	}{
		{
			name: "Test populate monthly categories",
			args: args{catAgg: catAgg, cats: cols},
			want: want,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := populateMonthlyCategories(tt.args.catAgg, tt.args.cats); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("populateMonthlyCategories() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_populateMonthlyPayees(t *testing.T) {
	b, err := os.ReadFile("json/payee_agg.json")
	if err != nil {
		t.Fatalf("could not open json test input file: %s\n", err.Error())
	}
	var payeeAgg map[string]map[string]float64
	err = json.Unmarshal(b, &payeeAgg)
	if err != nil {
		t.Fatalf("could not unmarshal json test data: %s\n", err.Error())
	}

	w, err := os.ReadFile("json/populateMonthlyPayees.json")
	if err != nil {
		t.Fatalf("could not open json test input file: %s\n", err.Error())
	}
	var want []*sheets.RowData
	err = json.Unmarshal(w, &want)
	if err != nil {
		t.Fatalf("could not unmarshal json test data: %s\n", err.Error())
	}

	type args struct {
		payeeAgg map[string]map[string]float64
	}
	tests := []struct {
		name string
		args args
		want []*sheets.RowData
	}{
		{
			name: "Test populate monthly payees",
			args: args{payeeAgg: payeeAgg},
			want: want,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := populateMonthlyPayees(tt.args.payeeAgg); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("populateMonthlyPayees() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_addSummarySalaryRow(t *testing.T) {
	months := []string{"Jan", "Feb", "Mar"}
	b, err := os.ReadFile("json/addSummarySalaryRow.json")
	if err != nil {
		t.Fatalf("could not open json test input file: %s\n", err.Error())
	}
	var want *sheets.RowData
	err = json.Unmarshal(b, &want)
	if err != nil {
		t.Fatalf("could not unmarshal json test data: %s\n", err.Error())
	}

	aggData := make(map[string]map[string]float64)
	aggData["Jan"] = make(map[string]float64)
	aggData["Jan"]["CrowdStrike Salary"] = 10.00
	aggData["Feb"] = make(map[string]float64)
	aggData["Feb"]["CrowdStrike Salary"] = 10.00
	aggData["Mar"] = make(map[string]float64)
	aggData["Mar"]["CrowdStrike Salary"] = 10.00

	type args struct {
		rNum    int
		months  *[]string
		aggData map[string]map[string]float64
	}
	tests := []struct {
		name string
		args args
		want *sheets.RowData
	}{
		{
			name: "Test add salary row",
			args: args{rNum: 10, months: &months, aggData: aggData},
			want: want,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := addSummarySalaryRow(tt.args.rNum, tt.args.months, tt.args.aggData); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("addSummarySalaryRow() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_addSummaryTopRow(t *testing.T) {
	months := []string{"Jan", "Feb", "Mar"}
	b, err := os.ReadFile("json/addSummaryTopRow.json")
	if err != nil {
		t.Fatalf("could not open json test input file: %s\n", err.Error())
	}
	var want *sheets.RowData
	err = json.Unmarshal(b, &want)
	if err != nil {
		t.Fatalf("could not unmarshal json test data: %s\n", err.Error())
	}

	type args struct {
		months *[]string
	}
	tests := []struct {
		name string
		args args
		want *sheets.RowData
	}{
		{
			name: "Test add summary row",
			args: args{months: &months},
			want: want,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := addSummaryTopRow(tt.args.months); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("addSummaryTopRow() = %v, want %v", got, tt.want)
			}
		})
	}
}

func checkTestingError(t *testing.T, err error) {
	if err != nil {
		t.Fatalf("error: %s\n", err.Error())
	}
}
