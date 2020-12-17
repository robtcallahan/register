package sheets_service

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"reflect"
	"testing"

	"google.golang.org/api/sheets/v4"
	"register/pkg/models"
)

const dir = "/Users/rob/ws/go/src/register/api/services/sheets_service/json/"

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

	j, err := ioutil.ReadFile(dir + "spreadsheet.json")
	if err != nil {
		return nil, errors.New(fmt.Sprintf("could not read json file: %s, %s", dir+"spreadsheet.json", err.Error()))
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
	w, err := ioutil.ReadFile(dir + "WriteCell.json")
	checkError(err)
	var resp *sheets.UpdateValuesResponse
	err = json.Unmarshal(w, &resp)
	checkError(err)
	return resp, nil
}

var ss *sheetsService

func init() {
	var err error

	c, err := ioutil.ReadFile("json/categoriesMap.json")
	checkError(err)
	var entries map[string]*BudgetEntry
	err = json.Unmarshal(c, &entries)
	checkError(err)

	provider := providerMock{
		service:       &sheets.Service{},
		spreadsheetID: "ssID",
	}
	ss, err = New(&provider)
	if err != nil {
		fmt.Errorf("%s", err.Error())
	}
	ss.RegisterSheet = &RegisterSheet{
		TabName:       "Register",
		CategoriesMap: entries,
	}
	ss.BudgetSheet = &BudgetSheet{
		TabName:        "Budget",
		StartRow:       3,
		EndRow:         63,
		EndColumnName:  "J",
		EndColumnIndex: 9,
	}
}

func Test_sheetsService_copyRowsBatchUpdateRequest(t *testing.T) {
	w, err := ioutil.ReadFile(dir + "copyRowsBatchUpdateRequest.json")
	checkTestingError(t, err)
	var want sheets.BatchUpdateSpreadsheetRequest
	err = json.Unmarshal(w, &want)
	checkTestingError(t, err)

	type args struct {
		numCopies int
	}
	tests := []struct {
		name   string
		args   args
		want   sheets.BatchUpdateSpreadsheetRequest
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
	w, err := ioutil.ReadFile(dir + "ReadBudgetSheet.json")
	checkTestingError(t, err)
	var want BudgetSheet
	err = json.Unmarshal(w, &want)
	checkTestingError(t, err)

	getValuesProviderFunc = func(range_ string) (*sheets.ValueRange, error) {
		r, err := ioutil.ReadFile(dir + "BudgetValues.json")
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
	w, err := ioutil.ReadFile(dir + "ReadRegisterSheet.json")
	checkTestingError(t, err)
	var want RegisterSheet
	err = json.Unmarshal(w, &want)
	checkTestingError(t, err)

	getValuesProviderFunc = func(range_ string) (*sheets.ValueRange, error) {
		r, err := ioutil.ReadFile(dir + "RegisterValues.json")
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
	w, err := ioutil.ReadFile(dir + "WriteCell.json")
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
			if got := ss.WriteCell(tt.args.cell, tt.args.value); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("WriteCell() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_sheetsService_addAmountCells(t *testing.T) {
	a, err := ioutil.ReadFile(dir + "addAmountCells.json")
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
			if got := ss.addAmountCells(tt.args.cells, tt.args.trans, tt.args.bgColor); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("addAmountCells() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_sheetsService_addCategoryCells(t *testing.T) {
	c, err := ioutil.ReadFile(dir + "columns.json")
	checkTestingError(t, err)
	var cols []models.Column
	err = json.Unmarshal(c, &cols)
	checkTestingError(t, err)

	n, err := ioutil.ReadFile(dir + "nameToCol.json")
	checkTestingError(t, err)
	var name2Col map[string]string
	err = json.Unmarshal(n, &name2Col)
	checkTestingError(t, err)

	a, err := ioutil.ReadFile(dir + "addCategoryCells.json")
	checkTestingError(t, err)
	var want []*sheets.CellData
	err = json.Unmarshal(a, &want)
	checkTestingError(t, err)

	type args struct {
		cells          []*sheets.CellData
		trans          *models.Transaction
		columns        []models.Column
		nameToCol      map[string]string
		totalsFormulas []string
	}
	tests := []struct {
		name string
		args args
		want []*sheets.CellData
	}{
		{
			name: "Test add category cells",
			args: args{
				cells:     []*sheets.CellData{},
				trans:     &models.Transaction{},
				columns:   cols,
				nameToCol: name2Col,
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
			if got := ss.addCategoryCells(tt.args.cells, tt.args.trans, tt.args.columns, tt.args.nameToCol, tt.args.totalsFormulas); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("addCategoryCells() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_sheetsService_addSalaryCells(t *testing.T) {
	c, err := ioutil.ReadFile("json/columns.json")
	checkTestingError(t, err)
	var cols []models.Column
	err = json.Unmarshal(c, &cols)
	checkTestingError(t, err)

	w, err := ioutil.ReadFile("json/addSalaryCells.json")
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
	j, err := ioutil.ReadFile(dir + "addSourceDataNameCells.json")
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
			if got := ss.addSourceDateNameCells(tt.args.cells, tt.args.trans, tt.args.bgColor); !reflect.DeepEqual(got, tt.want) {
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

func Test_sheetsService_getSourceField(t *testing.T) {
	type fields struct {
		service       *sheets.Service
		SpreadsheetID string
		BudgetSheet   *BudgetSheet
		RegisterSheet *RegisterSheet
		Debug         bool
		Verbose       bool
	}
	type args struct {
		values []interface{}
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ss := &sheetsService{
				service:       tt.fields.service,
				SpreadsheetID: tt.fields.SpreadsheetID,
				BudgetSheet:   tt.fields.BudgetSheet,
				RegisterSheet: tt.fields.RegisterSheet,
				Debug:         tt.fields.Debug,
				Verbose:       tt.fields.Verbose,
			}
			if got := ss.getSourceField(tt.args.values); got != tt.want {
				t.Errorf("getSourceField() = %v, want %v", got, tt.want)
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
		columns      []models.Column
		nameToCol    map[string]string
		transactions []*models.Transaction
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
			ss := &sheetsService{
				service:       tt.fields.service,
				SpreadsheetID: tt.fields.SpreadsheetID,
				BudgetSheet:   tt.fields.BudgetSheet,
				RegisterSheet: tt.fields.RegisterSheet,
				Debug:         tt.fields.Debug,
				Verbose:       tt.fields.Verbose,
			}
			if got := ss.populateCells(tt.args.columns, tt.args.nameToCol, tt.args.transactions); !reflect.DeepEqual(got, tt.want) {
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
			//	Verbose:       tt.fields.Verbose,
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

func Test_sheetsService_getAmount(t *testing.T) {
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
			if got := ss.getAmount(tt.args.values); got != tt.want {
				t.Errorf("getAmount() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_sheetsService_GetRegisterField(t *testing.T) {
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
			args: args{values: []interface{}{ "$ 10.20 " }, i: 0 },
			want: 10.20,
		},
		{
			name: "Test negative dollars",
			args: args{values: []interface{}{ "$ (10.20) " }, i: 0 },
			want: -10.20,
		},
		{
			name: "Test for dash",
			args: args{values: []interface{}{ " - " }, i: 0 },
			want: 0,
		},
		{
			name: "Test empty cell",
			args: args{values: []interface{}{ "" }, i: 0 },
			want: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ss.GetRegisterField(tt.args.values, tt.args.i); got != tt.want {
				t.Errorf("GetRegisterField() = %v, want %v", got, tt.want)
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
			if got := ss.getDateField(tt.args.values); got != tt.want {
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
			if got := ss.getNameField(tt.args.values); got != tt.want {
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
			if got := ss.ReadStringCell(tt.args.cell); got != tt.want {
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
			if got := ss.ReadDateCell(tt.args.cell); got != tt.want {
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
			if got := ss.ReadDollarsCell(tt.args.cell); got != tt.want {
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
			if got := ss.ReadFormulaCell(tt.args.cell); got != tt.want {
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
		dType string
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
				dType: "formula",
			},
			want: "=SUM(A1:A2)",
		},
		{
			name: "Test reading a string cell",
			args: args{
				cell:  "A1",
				dType: "string",
			},
			want: "a string",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ss.ReadCell(tt.args.cell, tt.args.dType); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ReadCell() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_addSummaryRows(t *testing.T) {
	b, err := ioutil.ReadFile("json/cat_agg.json")
	checkTestingError(t, err)

	var catAgg map[string]map[string]float64
	err = json.Unmarshal(b, &catAgg)
	checkTestingError(t, err)

	c, err := ioutil.ReadFile("json/column_names.json")
	checkTestingError(t, err)

	var cNames []string
	err = json.Unmarshal(c, &cNames)
	checkTestingError(t, err)

	w, err := ioutil.ReadFile("json/addSummaryRows.json")
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
	b, err := ioutil.ReadFile("json/cat_agg.json")
	checkTestingError(t, err)

	var catAgg map[string]map[string]float64
	err = json.Unmarshal(b, &catAgg)
	checkTestingError(t, err)

	c, err := ioutil.ReadFile("json/columns.json")
	checkTestingError(t, err)

	var cols []models.Column
	err = json.Unmarshal(c, &cols)
	checkTestingError(t, err)

	w, err := ioutil.ReadFile("json/populateMonthlyCategories.json")
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
	b, err := ioutil.ReadFile("json/payee_agg.json")
	if err != nil {
		t.Fatalf("could not open json test input file: %s\n", err.Error())
	}
	var payeeAgg map[string]map[string]float64
	err = json.Unmarshal(b, &payeeAgg)
	if err != nil {
		t.Fatalf("could not unmarshal json test data: %s\n", err.Error())
	}

	w, err := ioutil.ReadFile("json/populateMonthlyPayees.json")
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

func Test_addSalaryRow(t *testing.T) {
	months := []string{"Jan", "Feb", "Mar"}
	b, err := ioutil.ReadFile("json/addSalaryRow.json")
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
			if got := addSalaryRow(tt.args.rNum, tt.args.months, tt.args.aggData); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("addSalaryRow() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_addSummaryTopRow(t *testing.T) {
	months := []string{"Jan", "Feb", "Mar"}
	b, err := ioutil.ReadFile("json/addSummaryTopRow.json")
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

func Test_rowColor(t *testing.T) {
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
			if got := rowColor(tt.args.i, tt.args.even, tt.args.odd); got != tt.want {
				t.Errorf("rowColor() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_sortKeys(t *testing.T) {
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
			if got := sortKeys(tt.args.aggMap); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("sortKeys() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_mkStringCell(t *testing.T) {
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
			if got := mkStringCell(tt.args.value, tt.args.align, tt.args.color, tt.args.bordersOn); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("mkStringCell() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_mkBoldStringCell(t *testing.T) {
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
			if got := mkBoldStringCell(tt.args.value, tt.args.align, tt.args.color, tt.args.bordersOn); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("mkBoldStringCell() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_mkNumberCell(t *testing.T) {
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
			if got := mkNumberCell(tt.args.value, tt.args.align, tt.args.color, tt.args.bordersOn); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("mkNumberCell() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_mkDollarsCell(t *testing.T) {
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
			name: "Test return complete CellData from dollars string",
			args: args{value: 100.1234, align: "right", colorName: "green", bordersOn: false},
			want: &sheets.CellData{
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
			if got := mkDollarsCell(tt.args.value, tt.args.align, tt.args.colorName, tt.args.bordersOn); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("mkDollarsCell() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_mkDollarsCellFromFormulaString(t *testing.T) {
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
			if got := mkDollarsCellFromFormulaString(tt.args.value, tt.args.align, tt.args.colorName, tt.args.bordersOn); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("mkDollarsCellFromFormulaString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_mkDateCell(t *testing.T) {
	f := 43832.0
	type args struct {
		dateString string
		align      string
		colorName  string
		bordersOn  bool
	}
	tests := []struct {
		name string
		args args
		want *sheets.CellData
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mkDateCell(tt.args.dateString, tt.args.align, tt.args.colorName, tt.args.bordersOn); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("mkDateCell() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_mkOpaqueCell(t *testing.T) {
	type args struct {
		colorName string
		bordersOn bool
	}
	tests := []struct {
		name string
		args args
		want *sheets.CellData
	}{
		{
			name: "Test return empty cell with color and border specified",
			args: args{colorName: "white", bordersOn: false},
			want: &sheets.CellData{
				UserEnteredFormat: &sheets.CellFormat{
					TextFormat: font(),
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
			if got := mkOpaqueCell(tt.args.colorName, tt.args.bordersOn); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("mkOpaqueCell() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_formatCell(t *testing.T) {
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
			name: "Test complete Cell Format",
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
						Color: color("black"),
						Style: "SOLID",
					},
					Right: &sheets.Border{
						Color: color("black"),
						Style: "SOLID",
					},
					Bottom: &sheets.Border{
						Color: color("black"),
						Style: "SOLID",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatCell(tt.args.align, tt.args.colorName, tt.args.bordersOn); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("formatCell() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_dollarFormat(t *testing.T) {
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
			if got := dollarFormat(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("dollarFormat() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_dateFormat(t *testing.T) {
	tests := []struct {
		name string
		want *sheets.NumberFormat
	}{
		{
			name: "Test NumberFormat as DATE",
			want: &sheets.NumberFormat{
				Pattern: "mm/dd/yy",
				Type:    "DATE",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := dateFormat(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("dateFormat() = %v, want %v", got, tt.want)
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
				t.Errorf("getStringField() = %v, want %v", got, tt.want)
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
				t.Errorf("readStringValue() = %v, want %v", got, tt.want)
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
				t.Errorf("readDollarsValue() = %v, want %v", got, tt.want)
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
				t.Errorf("readDateValue() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_color(t *testing.T) {
	type args struct {
		name string
	}
	tests := []struct {
		name string
		args args
		want *sheets.Color
	}{
		{
			name: "Test White",
			args: args{name: "white"},
			want: &sheets.Color{
				Alpha: 1,
				Blue:  1,
				Red:   1,
				Green: 1},
		},
		{
			name: "Test Green",
			args: args{name: "green"},
			want: &sheets.Color{
				Alpha: 1,
				Blue:  0,
				Red:   0.5,
				Green: 1,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := color(tt.args.name); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("color() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_borders(t *testing.T) {
	type args struct {
		on bool
	}
	tests := []struct {
		name string
		args args
		want *sheets.Borders
	}{
		{
			name: "Test Borders On",
			args: args{on: false},
			want: &sheets.Borders{},
		},
		{
			name: "Test Borders Off",
			args: args{on: true},
			want: &sheets.Borders{
				Left: &sheets.Border{
					Color: color("black"),
					Style: "SOLID",
				},
				Right: &sheets.Border{
					Color: color("black"),
					Style: "SOLID",
				},
				Bottom: &sheets.Border{
					Color: color("black"),
					Style: "SOLID",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := borders(tt.args.on); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("borders() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_font(t *testing.T) {
	tests := []struct {
		name string
		want *sheets.TextFormat
	}{
		{
			name: "Test Standard Font",
			want: &sheets.TextFormat{
				FontFamily: "Arial",
				FontSize:   10,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := font(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("font() = %v, want %v", got, tt.want)
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
				t.Errorf("intInSlice() = %v, want %v", got, tt.want)
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
				t.Errorf("formatYear() = %v, want %v", got, tt.want)
			}
		})
	}
}

func checkTestingError(t *testing.T, err error) {
	if err != nil {
		t.Fatalf("error: %s\n", err.Error())
	}
}
