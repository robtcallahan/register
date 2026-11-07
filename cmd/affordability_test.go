package cmd

import (
	"strings"
	"testing"

	"register/api/services/sheets_service"
)

func TestPrintAffordability(t *testing.T) {
	result := sheets_service.AffordabilityResult{
		Housing: []sheets_service.HousingCategory{
			{Name: "mortgage", Monthly: 1450},
			{Name: "mortgage insurance", Monthly: 95},
			{Name: "home insurance", Monthly: 180},
			{Name: "property tax", Monthly: 325},
			{Name: "home-expenses savings", Monthly: 50},
		},
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
		DownPayments: []sheets_service.DownPayment{
			{Percent: 20, Amount: 50000, IRAFirstIRA: 80000, IRAFirstStocks: 50000, StocksFirstIRA: 130000, StocksFirstStocks: 0},
			{Percent: 10, Amount: 25000, IRAFirstIRA: 105000, IRAFirstStocks: 50000, StocksFirstIRA: 130000, StocksFirstStocks: 25000},
		},
	}
	opts := sheets_service.AffordabilityOptions{Months: 6, Floor: 200, IRA: 130000, Stocks: 50000, HomePrice: 250000}

	out := captureStdout(t, func() { printAffordability("House Budget", result, opts) })

	for _, want := range []string{
		"Housing (from the House Budget tab)",
		"$2,100.00/mo",
		"mortgage $1,450.00",
		"Housing costs $700.00/mo more than rent",
		"Trailing 6 complete months: May 2026 – Oct 2026",
		"Surplus after the move",
		"(200.00)",
		"doesn't fit as priced",
		"Max housing cost at zero surplus",
		"$1,900.00/mo",
		"Max housing cost keeping a $200.00/mo floor",
		"$1,700.00/mo",
		"Down payment on a $250,000.00 home:",
		"20% ($50,000.00) from IRA first — IRA $80,000.00 left, stocks $50,000.00 left",
		"20% ($50,000.00) from stocks first — IRA $130,000.00 left, stocks $0.00 left",
		"10% ($25,000.00) from IRA first — IRA $105,000.00 left, stocks $50,000.00 left",
		"Note: IRA withdrawals are taxable income.",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("printAffordability() output missing %q\noutput:\n%s", want, out)
		}
	}
}

func TestPrintAffordabilityFits(t *testing.T) {
	// A cheaper house and no home price: the fits marker, and no
	// down-payment section at all.
	result := sheets_service.AffordabilityResult{
		Housing: []sheets_service.HousingCategory{
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
	}
	opts := sheets_service.AffordabilityOptions{Months: 6, Floor: 200}

	out := captureStdout(t, func() { printAffordability("House Budget", result, opts) })

	for _, want := range []string{
		"Housing costs $200.00/mo less than rent",
		"<- fits",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("printAffordability() output missing %q\noutput:\n%s", want, out)
		}
	}
	if strings.Contains(out, "Down payment") {
		t.Errorf("printAffordability() printed a down-payment section with no home price\noutput:\n%s", out)
	}
	if strings.Contains(out, "doesn't fit") {
		t.Errorf("printAffordability() printed the doesn't-fit marker for a surplus that fits\noutput:\n%s", out)
	}
}

func TestPrintAffordabilitySameAsRent(t *testing.T) {
	// Housing exactly equal to rent: the delta line says so.
	result := sheets_service.AffordabilityResult{
		Housing: []sheets_service.HousingCategory{
			{Name: "mortgage", Monthly: 1400},
		},
		HousingTotal:      1400,
		Rent:              1400,
		HousingDelta:      0,
		AvgIncome:         5200,
		AvgSpending:       4700,
		Surplus:           500,
		SurplusAfterMove:  500,
		MaxHousing:        1900,
		MaxHousingAtFloor: 1700,
		WindowMonths:      []string{"10/26"},
	}
	opts := sheets_service.AffordabilityOptions{Months: 1, Floor: 200}

	out := captureStdout(t, func() { printAffordability("House Budget", result, opts) })

	if !strings.Contains(out, "Housing costs the same as rent") {
		t.Errorf("printAffordability() output missing the same-as-rent line\noutput:\n%s", out)
	}
}

func TestPrintAffordabilityShortfall(t *testing.T) {
	// A $100,000 home against $1,000 IRA + $500 stocks: the 20% down
	// payment ($20,000) is bigger than both accounts put together.
	result := sheets_service.AffordabilityResult{
		HousingTotal:      2100,
		Rent:              1400,
		HousingDelta:      700,
		AvgIncome:         5200,
		AvgSpending:       4700,
		Surplus:           500,
		SurplusAfterMove:  -200,
		MaxHousing:        1900,
		MaxHousingAtFloor: 1700,
		WindowMonths:      []string{"10/26"},
		DownPayments: []sheets_service.DownPayment{
			{Percent: 20, Amount: 20000, IRAFirstIRA: 0, IRAFirstStocks: -18500, StocksFirstIRA: -18500, StocksFirstStocks: 0},
		},
	}
	opts := sheets_service.AffordabilityOptions{Months: 1, Floor: 200, IRA: 1000, Stocks: 500, HomePrice: 100000}

	out := captureStdout(t, func() { printAffordability("House Budget", result, opts) })

	if !strings.Contains(out, "shortfall $18,500.00") {
		t.Errorf("printAffordability() output missing the shortfall\noutput:\n%s", out)
	}
}

func TestValidateAffordabilityOptions(t *testing.T) {
	valid := sheets_service.AffordabilityOptions{Months: 6, Floor: 200, IRA: 130000, Stocks: 50000, HomePrice: 250000}
	tests := []struct {
		name string
		opts sheets_service.AffordabilityOptions
		want string
	}{
		{name: "valid", opts: valid},
		{name: "zero months", opts: func() sheets_service.AffordabilityOptions { o := valid; o.Months = 0; return o }(), want: "--months"},
		{name: "negative floor", opts: func() sheets_service.AffordabilityOptions { o := valid; o.Floor = -1; return o }(), want: "--floor"},
		{name: "negative IRA", opts: func() sheets_service.AffordabilityOptions { o := valid; o.IRA = -1; return o }(), want: "--ira"},
		{name: "negative stocks", opts: func() sheets_service.AffordabilityOptions { o := valid; o.Stocks = -1; return o }(), want: "--stocks"},
		{name: "negative price", opts: func() sheets_service.AffordabilityOptions { o := valid; o.HomePrice = -1; return o }(), want: "--price"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateAffordabilityOptions(tt.opts)
			if tt.want == "" {
				if err != nil {
					t.Errorf("validateAffordabilityOptions() error = %v, want nil", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("validateAffordabilityOptions() error = %v, want it to mention %q", err, tt.want)
			}
		})
	}
}

func TestFormatWindowLabel(t *testing.T) {
	tests := []struct {
		name   string
		months []string
		want   string
	}{
		{name: "no months", months: nil, want: "no months"},
		{name: "one month", months: []string{"10/26"}, want: "Oct 2026"},
		{name: "six months", months: []string{"05/26", "06/26", "07/26", "08/26", "09/26", "10/26"}, want: "May 2026 – Oct 2026"},
		{name: "across a year boundary", months: []string{"11/25", "12/25", "01/26", "02/26", "03/26", "04/26"}, want: "Nov 2025 – Apr 2026"},
		{name: "unreadable key passes through", months: []string{"junk"}, want: "junk"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatWindowLabel(tt.months); got != tt.want {
				t.Errorf("formatWindowLabel() = %q, want %q", got, tt.want)
			}
		})
	}
}
