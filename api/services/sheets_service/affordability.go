package sheets_service

import (
	"math"
	"regexp"
	"time"

	"register/pkg/config"
)

// House affordability (register affordability): one read-only view of
// whether the house on the House Budget tab fits on Rob's current income.
// The tab is read generically — every row on it is a housing category — so
// a category added later (HOA fees, say) shows up with no code change.

// HousingCategory is one row of the House Budget tab: the category name
// and its monthly amount (the Monthly column, whether raw or derived from
// another cadence).
type HousingCategory struct {
	Name    string
	Monthly float64
}

// AffordabilityOptions are the command's flags.
type AffordabilityOptions struct {
	Months    int     // trailing window, in complete months before this one
	Floor     float64 // minimum monthly surplus to keep after the move
	IRA       float64 // IRA balance, for the down-payment picture
	Stocks    float64 // stocks balance, for the down-payment picture
	HomePrice float64 // 0 = skip the down-payment picture
}

// DownPayment is one down-payment size (10% or 20% of the home price)
// shown both ways: spend the IRA first, or spend the stocks first.
// Each side reports what would be left in each account afterwards; a
// negative remainder is the shortfall — the price is bigger than both
// accounts put together.
type DownPayment struct {
	Percent           float64
	Amount            float64
	IRAFirstIRA       float64
	IRAFirstStocks    float64
	StocksFirstIRA    float64
	StocksFirstStocks float64
}

// AffordabilityResult is everything the command prints.
type AffordabilityResult struct {
	Housing           []HousingCategory
	HousingTotal      float64
	Rent              float64 // average monthly rent over the window
	HousingDelta      float64 // HousingTotal - Rent
	AvgIncome         float64
	AvgSpending       float64
	Surplus           float64 // AvgIncome - AvgSpending
	SurplusAfterMove  float64 // Surplus - HousingDelta
	MaxHousing        float64 // Surplus + Rent: housing total that leaves zero surplus
	MaxHousingAtFloor float64 // Surplus + Rent - Floor: housing total that keeps the floor
	WindowMonths      []string
	DownPayments      []DownPayment
}

// MonthlyActuals is one month of real numbers from the Register tab.
type MonthlyActuals struct {
	Month    string // "MM/YY", the Aggregate key
	Income   float64
	Spending float64
	Rent     float64
}

// RentCategoryName is the register column that holds rent. Rent comes from
// the aggregate's category columns, not from entry names.
const RentCategoryName = "Rent"

// registerDateMonth matches a register date (MM/DD/YY) and yields the
// Aggregate month key (MM/YY). Same shape as Aggregate's own regex.
var registerDateMonth = regexp.MustCompile(`(\d\d)/\d\d/(\d\d)`)

func monthKeyFromRegisterDate(date string) (string, bool) {
	m := registerDateMonth.FindStringSubmatch(date)
	if len(m) == 0 {
		return "", false
	}
	return m[1] + "/" + m[2], true
}

// trailingMonthKeys returns the keys of the `months` complete months
// before the month `now` falls in, oldest first. The current month is
// partial, so it never counts: on 2026-11-06 with months=6 the window is
// May..Oct 2026.
func trailingMonthKeys(months int, now time.Time) []string {
	if months <= 0 {
		return nil
	}
	keys := make([]string, 0, months)
	for i := months; i >= 1; i-- {
		m := now.AddDate(0, -i, 0)
		keys = append(keys, m.Format("01/06"))
	}
	return keys
}

// TrailingActuals builds one MonthlyActuals per month in the trailing
// window, from the register aggregate and the raw entries. Income comes
// from the entries themselves — Aggregate skips deposit rows, so income
// never lands in the aggregate — while spending and rent come from the
// aggregate's category columns. Months with no transactions still appear,
// as zeros: the average divides by the window, not by the months that
// happen to have data.
func TrailingActuals(catAgg map[string]map[string]float64, entries []*RegisterEntry, incomeSources []config.IncomeSource, months int, now time.Time) []MonthlyActuals {
	keys := trailingMonthKeys(months, now)
	inWindow := make(map[string]bool, len(keys))
	for _, k := range keys {
		inWindow[k] = true
	}

	// Pre-sized so the pointers below never move: appends can't
	// reallocate a slice built with capacity len(keys).
	actuals := make([]MonthlyActuals, 0, len(keys))
	byMonth := make(map[string]*MonthlyActuals, len(keys))
	for _, k := range keys {
		actuals = append(actuals, MonthlyActuals{Month: k})
		byMonth[k] = &actuals[len(actuals)-1]
	}

	// Spending and rent, from the aggregate's category columns. Income
	// names only reach the aggregate in the zero-deposit edge case, but
	// exclude them anyway so a stray income entry can't inflate spending.
	for _, a := range byMonth {
		for name, amount := range catAgg[a.Month] {
			if isIncomeName(name, incomeSources) {
				continue
			}
			a.Spending += amount
			if name == RentCategoryName {
				a.Rent += amount
			}
		}
	}

	// Income, from the entries: deposits whose display name is a
	// configured income source. Anything else deposited (a refund, a
	// transfer) is not income.
	for _, e := range entries {
		if e == nil || e.Deposit == 0 || !isIncomeName(e.Name, incomeSources) {
			continue
		}
		key, ok := monthKeyFromRegisterDate(e.Date)
		if !ok || !inWindow[key] {
			continue
		}
		byMonth[key].Income += e.Deposit
	}

	return actuals
}

// ComputeAffordability turns housing categories and trailing actuals into
// the numbers the command prints. It is pure: inputs in, numbers out.
func ComputeAffordability(housing []HousingCategory, actuals []MonthlyActuals, opts AffordabilityOptions) AffordabilityResult {
	var result AffordabilityResult
	result.Housing = housing

	for _, h := range housing {
		result.HousingTotal += h.Monthly
	}

	var totalIncome, totalSpending, totalRent float64
	for _, a := range actuals {
		result.WindowMonths = append(result.WindowMonths, a.Month)
		totalIncome += a.Income
		totalSpending += a.Spending
		totalRent += a.Rent
	}
	n := float64(len(actuals))
	if n > 0 {
		result.AvgIncome = totalIncome / n
		result.AvgSpending = totalSpending / n
		result.Rent = totalRent / n
	}
	result.Surplus = result.AvgIncome - result.AvgSpending
	result.HousingDelta = result.HousingTotal - result.Rent
	result.SurplusAfterMove = result.Surplus - result.HousingDelta
	result.MaxHousing = result.Surplus + result.Rent
	result.MaxHousingAtFloor = result.Surplus + result.Rent - opts.Floor

	if opts.HomePrice > 0 {
		// 20% first, then 10%: the bigger commitment first.
		for _, pct := range []float64{20, 10} {
			amount := opts.HomePrice * pct / 100
			iraFirstIRA, iraFirstStocks := drawDown(amount, opts.IRA, opts.Stocks)
			stocksFirstStocks, stocksFirstIRA := drawDown(amount, opts.Stocks, opts.IRA)
			result.DownPayments = append(result.DownPayments, DownPayment{
				Percent:           pct,
				Amount:            amount,
				IRAFirstIRA:       iraFirstIRA,
				IRAFirstStocks:    iraFirstStocks,
				StocksFirstIRA:    stocksFirstIRA,
				StocksFirstStocks: stocksFirstStocks,
			})
		}
	}
	return result
}

// drawDown spends amount from first, then second, and returns what is
// left in each. A negative remainder means amount was bigger than both
// put together.
func drawDown(amount, first, second float64) (firstLeft, secondLeft float64) {
	fromFirst := math.Min(amount, first)
	return first - fromFirst, second - (amount - fromFirst)
}
