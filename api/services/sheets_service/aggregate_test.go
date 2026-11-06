package sheets_service

import (
	"errors"
	"reflect"
	"strconv"
	"testing"

	"register/pkg/config"
	"register/pkg/models"
)

func Test_monthlyService_Aggregate(t *testing.T) {
	// Category columns for the aggregate: Grocery and Rent accumulate,
	// the grey payoff column and the fixed Source column are skipped.
	cols := []models.Column{
		{Name: "Source", ColumnIndex: 2},
		{Name: "Grocery", ColumnIndex: 11, Color: models.ColorBlue},
		{Name: "Rent", ColumnIndex: 12, Color: models.ColorBlue},
		{Name: "Credit Cards", ColumnIndex: 13, Color: models.ColorGrey},
	}

	// row builds one sheet row with category dollars at the columns'
	// 0-based positions (Grocery 10, Rent 11, Credit Cards 12).
	row := func(grocery, rent, card string) []any {
		values := make([]any, 13)
		values[10] = grocery
		values[11] = rent
		values[12] = card
		return values
	}

	// rangeValues places each entry's row at the even index Aggregate
	// reads (entry i reads RangeValues[i*2]); odd slots are the blank
	// second row of each entry and are never read.
	rangeValues := func(rows ...[]any) [][]any {
		out := make([][]any, 2*len(rows))
		for i, r := range rows {
			out[i*2] = r
		}
		return out
	}

	ms := NewMonthlyService(&providerMock{}, []config.IncomeSource{
		{Name: "Taylor House Paycheck", Frequency: "biweekly"},
	})

	tests := []struct {
		name      string
		entries   []*RegisterEntry
		rangeVals [][]any
		wantCat   map[string]map[string]float64
		wantPayee map[string]map[string]float64
		wantErr   error
	}{
		{
			name: "spending row accumulates payee and categories",
			entries: []*RegisterEntry{
				{Name: "TARGET", Date: "12/07/20", Withdrawal: 46.80},
			},
			rangeVals: rangeValues(row("46.80", "20.00", "99.00")),
			wantCat: map[string]map[string]float64{
				"12/20": {"Grocery": 46.80, "Rent": 20},
			},
			wantPayee: map[string]map[string]float64{
				"12/20": {"TARGET": -46.80},
			},
		},
		{
			name: "same payee accumulates within a month, months stay separate",
			entries: []*RegisterEntry{
				{Name: "TARGET", Date: "12/07/20", Withdrawal: 46.80},
				{Name: "TARGET", Date: "12/09/20", Withdrawal: 10},
				{Name: "AMAZON", Date: "01/03/21", CreditPurchase: 30},
			},
			rangeVals: rangeValues(
				row("46.80", "20.00", ""),
				row("10.00", "", ""),
				row("", "5.00", ""),
			),
			wantCat: map[string]map[string]float64{
				"12/20": {"Grocery": 56.80, "Rent": 20},
				"01/21": {"Grocery": 0, "Rent": 5},
			},
			wantPayee: map[string]map[string]float64{
				"12/20": {"TARGET": -56.80},
				"01/21": {"AMAZON": -30},
			},
		},
		{
			name: "reallocation and deposit rows are skipped",
			entries: []*RegisterEntry{
				{Name: "Reallocation", Date: "12/07/20", Withdrawal: 100},
				{Name: "Taylor House Paycheck", Date: "12/08/20", Deposit: 1000},
				{Name: "TARGET", Date: "12/07/20", Withdrawal: 10},
			},
			rangeVals: rangeValues(
				row("999.00", "", ""),
				row("888.00", "", ""),
				row("10.00", "", ""),
			),
			wantCat: map[string]map[string]float64{
				"12/20": {"Grocery": 10, "Rent": 0},
			},
			wantPayee: map[string]map[string]float64{
				"12/20": {"TARGET": -10},
			},
		},
		{
			name: "income row with zero deposit adds zero and skips the column loop",
			entries: []*RegisterEntry{
				{Name: "Taylor House Paycheck", Date: "12/08/20", Withdrawal: 5},
			},
			rangeVals: rangeValues(row("99.00", "88.00", "77.00")),
			wantCat: map[string]map[string]float64{
				"12/20": {"Taylor House Paycheck": 0},
			},
			wantPayee: map[string]map[string]float64{
				"12/20": {"Taylor House Paycheck": -5},
			},
		},
		{
			name: "date without a match is skipped",
			entries: []*RegisterEntry{
				{Name: "TARGET", Date: "not-a-date", Withdrawal: 10},
			},
			rangeVals: rangeValues(row("10.00", "", "")),
			wantCat:   map[string]map[string]float64{},
			wantPayee: map[string]map[string]float64{},
		},
		{
			name: "missing range row yields zero categories",
			entries: []*RegisterEntry{
				{Name: "TARGET", Date: "12/07/20", Withdrawal: 10},
			},
			rangeVals: nil,
			wantCat: map[string]map[string]float64{
				"12/20": {"Grocery": 0, "Rent": 0},
			},
			wantPayee: map[string]map[string]float64{
				"12/20": {"TARGET": -10},
			},
		},
		{
			name: "bad category cell returns error and no maps",
			entries: []*RegisterEntry{
				{Name: "TARGET", Date: "12/07/20", Withdrawal: 10},
			},
			rangeVals: rangeValues(row("xx", "", "")),
			wantErr:   strconv.ErrSyntax,
		},
		{
			name:      "empty sheet aggregates to empty maps",
			wantCat:   map[string]map[string]float64{},
			wantPayee: map[string]map[string]float64{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sheet := RegisterSheet{Entries: tt.entries, RangeValues: tt.rangeVals}
			gotCat, gotPayee, err := ms.Aggregate(sheet, cols)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Aggregate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr != nil {
				return
			}
			if !reflect.DeepEqual(gotCat, tt.wantCat) {
				t.Errorf("Aggregate() catAgg = %v, want %v", gotCat, tt.wantCat)
			}
			if !reflect.DeepEqual(gotPayee, tt.wantPayee) {
				t.Errorf("Aggregate() payeeAgg = %v, want %v", gotPayee, tt.wantPayee)
			}
		})
	}
}
