package sheets_service

import (
	"encoding/json"
	"os"
	"reflect"
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
	b := readTestFixture(t, "json/addSummarySalaryRow.json")
	var want *sheets.RowData
	err := json.Unmarshal(b, &want)
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
		months  []string
		aggData map[string]map[string]float64
	}
	tests := []struct {
		name string
		args args
		want *sheets.RowData
	}{
		{
			name: "Test add salary row",
			args: args{rNum: 10, months: months, aggData: aggData},
			want: want,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := addSummarySalaryRow("", tt.args.rNum, tt.args.months, tt.args.aggData); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("addSummarySalaryRow() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_monthlyService_addSummaryTopRow(t *testing.T) {
	months := []string{"Jan", "Feb", "Mar"}
	b := readTestFixture(t, "json/addSummaryTopRow.json")
	var want *sheets.RowData
	err := json.Unmarshal(b, &want)
	if err != nil {
		t.Fatalf("could not unmarshal json test data: %s\n", err.Error())
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
			if got := addSummaryTopRow(tt.args.months); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("addSummaryTopRow() = %v, want %v", got, tt.want)
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

// readTestFixture reads a test fixture file, skipping the test when the
// fixture is absent. Fixtures live on Rob's Mac, not in the repo.
func readTestFixture(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("fixture not in repo: %s", path)
	}
	return b
}
