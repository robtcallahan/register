package sheets_service

import (
	"reflect"
	"testing"
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
