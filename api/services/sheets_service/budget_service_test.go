package sheets_service

import (
	"reflect"
	"testing"
)

func TestBudgetService_BudgetRows(t *testing.T) {
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
