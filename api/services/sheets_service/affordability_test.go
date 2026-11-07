package sheets_service

import (
	"reflect"
	"testing"
	"time"

	"register/pkg/config"
)

func Test_ComputeAffordability(t *testing.T) {
	// Six trailing months at the brief's example numbers: $5,200 income,
	// $4,700 spending (rent $1,400 inside it) every month.
	actuals := make([]MonthlyActuals, 6)
	months := []string{"05/26", "06/26", "07/26", "08/26", "09/26", "10/26"}
	for i := range actuals {
		actuals[i] = MonthlyActuals{Month: months[i], Income: 5200, Spending: 4700, Rent: 1400}
	}

	briefHousing := []HousingCategory{
		{Name: "mortgage", Monthly: 1450},
		{Name: "mortgage insurance", Monthly: 95},
		{Name: "home insurance", Monthly: 180},
		{Name: "property tax", Monthly: 325},
		{Name: "home-expenses savings", Monthly: 50},
	} // total $2,100

	tests := []struct {
		name    string
		housing []HousingCategory
		actuals []MonthlyActuals
		opts    AffordabilityOptions
		want    AffordabilityResult
	}{
		{
			name:    "house costs more than rent and doesn't fit",
			housing: briefHousing,
			actuals: actuals,
			opts:    AffordabilityOptions{Months: 6, Floor: 200},
			want: AffordabilityResult{
				Housing:           briefHousing,
				HousingTotal:      2100,
				Rent:              1400,
				HousingDelta:      700,
				AvgIncome:         5200,
				AvgSpending:       4700,
				Surplus:           500,
				SurplusAfterMove:  -200,
				MaxHousing:        1900,
				MaxHousingAtFloor: 1700,
				WindowMonths:      []string{"05/26", "06/26", "07/26", "08/26", "09/26", "10/26"},
			},
		},
		{
			name: "house cheaper than rent fits with room to spare",
			housing: []HousingCategory{
				{Name: "mortgage", Monthly: 900},
				{Name: "home insurance", Monthly: 150},
				{Name: "property tax", Monthly: 150},
			},
			actuals: actuals,
			opts:    AffordabilityOptions{Months: 6, Floor: 200},
			want: AffordabilityResult{
				Housing: []HousingCategory{
					{Name: "mortgage", Monthly: 900},
					{Name: "home insurance", Monthly: 150},
					{Name: "property tax", Monthly: 150},
				},
				HousingTotal:      1200,
				Rent:              1400,
				HousingDelta:      -200,
				AvgIncome:         5200,
				AvgSpending:       4700,
				Surplus:           500,
				SurplusAfterMove:  700,
				MaxHousing:        1900,
				MaxHousingAtFloor: 1700,
				WindowMonths:      []string{"05/26", "06/26", "07/26", "08/26", "09/26", "10/26"},
			},
		},
		{
			name:    "a category added later is summed with no code change",
			housing: append(append([]HousingCategory{}, briefHousing...), HousingCategory{Name: "HOA dues", Monthly: 75}),
			actuals: actuals,
			opts:    AffordabilityOptions{Months: 6, Floor: 200},
			want: AffordabilityResult{
				Housing:           append(append([]HousingCategory{}, briefHousing...), HousingCategory{Name: "HOA dues", Monthly: 75}),
				HousingTotal:      2175,
				Rent:              1400,
				HousingDelta:      775,
				AvgIncome:         5200,
				AvgSpending:       4700,
				Surplus:           500,
				SurplusAfterMove:  -275,
				MaxHousing:        1900,
				MaxHousingAtFloor: 1700,
				WindowMonths:      []string{"05/26", "06/26", "07/26", "08/26", "09/26", "10/26"},
			},
		},
		{
			name:    "no home price skips the down-payment picture",
			housing: briefHousing,
			actuals: actuals,
			opts:    AffordabilityOptions{Months: 6, Floor: 200, IRA: 130000, Stocks: 50000, HomePrice: 0},
			want: AffordabilityResult{
				Housing:           briefHousing,
				HousingTotal:      2100,
				Rent:              1400,
				HousingDelta:      700,
				AvgIncome:         5200,
				AvgSpending:       4700,
				Surplus:           500,
				SurplusAfterMove:  -200,
				MaxHousing:        1900,
				MaxHousingAtFloor: 1700,
				WindowMonths:      []string{"05/26", "06/26", "07/26", "08/26", "09/26", "10/26"},
			},
		},
		{
			name:    "down payments show both waterfalls",
			housing: briefHousing,
			actuals: actuals,
			opts:    AffordabilityOptions{Months: 6, Floor: 200, IRA: 130000, Stocks: 50000, HomePrice: 250000},
			want: AffordabilityResult{
				Housing:           briefHousing,
				HousingTotal:      2100,
				Rent:              1400,
				HousingDelta:      700,
				AvgIncome:         5200,
				AvgSpending:       4700,
				Surplus:           500,
				SurplusAfterMove:  -200,
				MaxHousing:        1900,
				MaxHousingAtFloor: 1700,
				WindowMonths:      []string{"05/26", "06/26", "07/26", "08/26", "09/26", "10/26"},
				DownPayments: []DownPayment{
					{Percent: 20, Amount: 50000, IRAFirstIRA: 80000, IRAFirstStocks: 50000, StocksFirstIRA: 130000, StocksFirstStocks: 0},
					{Percent: 10, Amount: 25000, IRAFirstIRA: 105000, IRAFirstStocks: 50000, StocksFirstIRA: 130000, StocksFirstStocks: 25000},
				},
			},
		},
		{
			name:    "down payment bigger than both accounts leaves a negative remainder",
			housing: briefHousing,
			actuals: actuals,
			opts:    AffordabilityOptions{Months: 6, Floor: 200, IRA: 130000, Stocks: 50000, HomePrice: 1000000},
			want: AffordabilityResult{
				Housing:           briefHousing,
				HousingTotal:      2100,
				Rent:              1400,
				HousingDelta:      700,
				AvgIncome:         5200,
				AvgSpending:       4700,
				Surplus:           500,
				SurplusAfterMove:  -200,
				MaxHousing:        1900,
				MaxHousingAtFloor: 1700,
				WindowMonths:      []string{"05/26", "06/26", "07/26", "08/26", "09/26", "10/26"},
				DownPayments: []DownPayment{
					{Percent: 20, Amount: 200000, IRAFirstIRA: 0, IRAFirstStocks: -20000, StocksFirstIRA: -20000, StocksFirstStocks: 0},
					{Percent: 10, Amount: 100000, IRAFirstIRA: 30000, IRAFirstStocks: 50000, StocksFirstIRA: 80000, StocksFirstStocks: 0},
				},
			},
		},
		{
			name:    "empty inputs don't divide by zero",
			housing: nil,
			actuals: nil,
			opts:    AffordabilityOptions{Months: 6, Floor: 200},
			want: AffordabilityResult{
				HousingTotal:      0,
				Rent:              0,
				HousingDelta:      0,
				AvgIncome:         0,
				AvgSpending:       0,
				Surplus:           0,
				SurplusAfterMove:  0,
				MaxHousing:        0,
				MaxHousingAtFloor: -200,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ComputeAffordability(tt.housing, tt.actuals, tt.opts)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ComputeAffordability() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func Test_TrailingActuals(t *testing.T) {
	incomeSources := []config.IncomeSource{
		{Name: "Taylor House Paycheck"},
		{Name: "Social Security"},
	}
	// "now" is 2026-11-06, so the 3-month window is Aug..Oct 2026 and
	// the 6-month window adds empty May..Jul.
	now := time.Date(2026, time.November, 6, 12, 0, 0, 0, time.UTC)

	catAgg := map[string]map[string]float64{
		"08/26": {"Grocery": 400, "Rent": 1400},
		"09/26": {"Grocery": 500, "Rent": 1400},
		"10/26": {"Grocery": 450, "Rent": 1400, "Taylor House Paycheck": 0},
	}
	entries := []*RegisterEntry{
		{Name: "Taylor House Paycheck", Date: "08/07/26", Deposit: 1100},
		{Name: "Taylor House Paycheck", Date: "08/21/26", Deposit: 1100},
		{Name: "Social Security", Date: "08/27/26", Deposit: 2981},
		{Name: "Taylor House Paycheck", Date: "09/04/26", Deposit: 1100},
		{Name: "Social Security", Date: "09/28/26", Deposit: 2981},
		{Name: "Taylor House Paycheck", Date: "10/02/26", Deposit: 1100},
		{Name: "Taylor House Paycheck", Date: "10/16/26", Deposit: 1100},
		{Name: "Social Security", Date: "10/27/26", Deposit: 2981},
		// Outside the window: the current (partial) month, and April.
		{Name: "Taylor House Paycheck", Date: "11/03/26", Deposit: 1100},
		{Name: "Taylor House Paycheck", Date: "04/30/26", Deposit: 1100},
		// A deposit that isn't income (a refund): not income, and
		// Aggregate skips deposit rows, so not spending either.
		{Name: "TARGET", Date: "08/07/26", Deposit: 50},
		// An income deposit with an unreadable date: skipped.
		{Name: "Social Security", Date: "garbage", Deposit: 2981},
	}

	tests := []struct {
		name   string
		months int
		want   []MonthlyActuals
	}{
		{
			name:   "three months of income, spending and rent",
			months: 3,
			want: []MonthlyActuals{
				{Month: "08/26", Income: 5181, Spending: 1800, Rent: 1400},
				{Month: "09/26", Income: 4081, Spending: 1900, Rent: 1400},
				{Month: "10/26", Income: 5181, Spending: 1850, Rent: 1400},
			},
		},
		{
			name:   "months with no transactions still appear, as zeros",
			months: 6,
			want: []MonthlyActuals{
				{Month: "05/26"},
				{Month: "06/26"},
				{Month: "07/26"},
				{Month: "08/26", Income: 5181, Spending: 1800, Rent: 1400},
				{Month: "09/26", Income: 4081, Spending: 1900, Rent: 1400},
				{Month: "10/26", Income: 5181, Spending: 1850, Rent: 1400},
			},
		},
		{
			name:   "zero months is empty",
			months: 0,
			want:   []MonthlyActuals{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TrailingActuals(catAgg, entries, incomeSources, tt.months, now)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("TrailingActuals() = %+v, want %+v", got, tt.want)
			}
		})
	}
}
