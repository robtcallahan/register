/*
Copyright © 2020 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cmd

import (
	"fmt"
	"strings"
	"time"

	"register/api/providers/sheets_provider"
	"register/api/services/sheets_service"
	"register/pkg/repository"
	"register/pkg/repository/mysql"

	"github.com/spf13/cobra"
)

// Flags for the affordability command. The trailing window and the floor
// are Rob's calls (a parameter, and $200/mo); the tab defaults to the
// House Budget tab; balances and price are only needed for the
// down-payment picture.
var (
	affordMonths    int
	affordFloor     float64
	affordTab       string
	affordIRA       float64
	affordStocks    float64
	affordHomePrice float64
)

// affordabilityCmd represents the affordability command
var affordabilityCmd = &cobra.Command{
	Use:   "affordability",
	Short: "Show whether the house on the House Budget tab fits on current income",
	Long: `Affordability reads the House Budget tab and the Register tab and answers
one question from real numbers: can I afford this house, and what does it
leave me? Read-only: it writes nothing to the sheet or the database.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return affordability()
	},
}

func init() {
	rootCmd.AddCommand(affordabilityCmd)
	affordabilityCmd.Flags().IntVar(&affordMonths, "months", 6, "trailing complete months to average")
	affordabilityCmd.Flags().Float64Var(&affordFloor, "floor", 200, "minimum monthly surplus to keep after the move")
	affordabilityCmd.Flags().StringVar(&affordTab, "tab", "House Budget", "housing budget tab to read")
	affordabilityCmd.Flags().Float64Var(&affordIRA, "ira", 0, "IRA balance for the down-payment picture")
	affordabilityCmd.Flags().Float64Var(&affordStocks, "stocks", 0, "stocks balance for the down-payment picture")
	affordabilityCmd.Flags().Float64Var(&affordHomePrice, "price", 0, "home price for the down-payment picture (0 skips it)")
}

func affordability() error {
	opts := sheets_service.AffordabilityOptions{
		Months:    affordMonths,
		Floor:     affordFloor,
		IRA:       affordIRA,
		Stocks:    affordStocks,
		HomePrice: affordHomePrice,
	}
	if err := validateAffordabilityOptions(opts); err != nil {
		return err
	}

	conn, err := repository.Connect(&config.DB)
	if err != nil {
		return err
	}
	qHandler := mysql.NewMySQLQueryRepo(conn)
	cols, err := qHandler.GetColumns()
	if err != nil {
		return err
	}

	sheetsProvider, err := sheets_provider.New(options.SpreadsheetID, config)
	if err != nil {
		return err
	}
	registerService := sheets_service.NewRegisterService(sheetsProvider)
	registerSheet, err := registerService.NewRegisterSheet(config, cols)
	if err != nil {
		return err
	}

	fmt.Println("Reading Register...")
	_, err = registerService.ReadRegisterSheet(registerSheet)
	if err != nil {
		return err
	}

	monthlyService := sheets_service.NewMonthlyService(sheetsProvider, config.IncomeSources)
	catAgg, _, err := monthlyService.Aggregate(*registerSheet, cols)
	if err != nil {
		return err
	}

	budgetService := sheets_service.NewBudgetService(sheetsProvider)
	fmt.Printf("Reading %s...\n", affordTab)
	housing, err := budgetService.ReadHousingSheet(affordTab, *config)
	if err != nil {
		return err
	}

	actuals := sheets_service.TrailingActuals(catAgg, registerSheet.Entries, config.IncomeSources, opts.Months, time.Now())
	result := sheets_service.ComputeAffordability(housing, actuals, opts)
	printAffordability(affordTab, result, opts)
	return nil
}

// validateAffordabilityOptions rejects flag values the computation can't
// use. The tab name is free-form: a missing tab fails later, at the read,
// with the provider's own error.
func validateAffordabilityOptions(opts sheets_service.AffordabilityOptions) error {
	if opts.Months < 1 {
		return fmt.Errorf("--months must be at least 1, got %d", opts.Months)
	}
	if opts.Floor < 0 {
		return fmt.Errorf("--floor must not be negative, got %v", opts.Floor)
	}
	if opts.IRA < 0 {
		return fmt.Errorf("--ira must not be negative, got %v", opts.IRA)
	}
	if opts.Stocks < 0 {
		return fmt.Errorf("--stocks must not be negative, got %v", opts.Stocks)
	}
	if opts.HomePrice < 0 {
		return fmt.Errorf("--price must not be negative, got %v", opts.HomePrice)
	}
	return nil
}

func printAffordability(tab string, result sheets_service.AffordabilityResult, opts sheets_service.AffordabilityOptions) {
	fmt.Println("")
	fmt.Printf("%-44s $%s/mo\n", fmt.Sprintf("Housing (from the %s tab)", tab), formatAccounting(result.HousingTotal))
	if len(result.Housing) > 0 {
		parts := make([]string, 0, len(result.Housing))
		for _, h := range result.Housing {
			parts = append(parts, fmt.Sprintf("%s $%s", h.Name, formatAccounting(h.Monthly)))
		}
		fmt.Printf("  %s\n", strings.Join(parts, " · "))
	}
	fmt.Printf("%-44s $%s/mo\n", "Current rent", formatAccounting(result.Rent))
	switch {
	case result.HousingDelta > 0:
		fmt.Printf("Housing costs $%s/mo more than rent\n", formatAccounting(result.HousingDelta))
	case result.HousingDelta < 0:
		fmt.Printf("Housing costs $%s/mo less than rent\n", formatAccounting(-result.HousingDelta))
	default:
		fmt.Printf("Housing costs the same as rent\n")
	}
	fmt.Println("")
	fmt.Printf("Trailing %d complete months: %s\n", len(result.WindowMonths), formatWindowLabel(result.WindowMonths))
	fmt.Printf("%-44s $%s/mo\n", fmt.Sprintf("Trailing %d-mo income", len(result.WindowMonths)), formatAccounting(result.AvgIncome))
	fmt.Printf("%-44s $%s/mo\n", fmt.Sprintf("Trailing %d-mo spending (incl. rent)", len(result.WindowMonths)), formatAccounting(result.AvgSpending))
	fmt.Printf("%-44s $%s/mo\n", "Surplus today", formatAccounting(result.Surplus))
	fits := "  <- fits"
	if result.SurplusAfterMove < 0 {
		fits = "  <- doesn't fit as priced"
	}
	fmt.Printf("%-44s $%s/mo%s\n", "Surplus after the move", formatAccounting(result.SurplusAfterMove), fits)
	fmt.Println("")
	fmt.Printf("%-44s $%s/mo\n", "Max housing cost at zero surplus", formatAccounting(result.MaxHousing))
	fmt.Printf("%-44s $%s/mo\n", fmt.Sprintf("Max housing cost keeping a $%s/mo floor", formatAccounting(opts.Floor)), formatAccounting(result.MaxHousingAtFloor))

	if len(result.DownPayments) > 0 {
		fmt.Println("")
		fmt.Printf("Down payment on a $%s home:\n", formatAccounting(opts.HomePrice))
		for _, dp := range result.DownPayments {
			pct := fmt.Sprintf("%.0f%%", dp.Percent)
			fmt.Printf("  %s ($%s) from IRA first — IRA $%s left, stocks $%s left%s\n",
				pct, formatAccounting(dp.Amount), formatAccounting(dp.IRAFirstIRA), formatAccounting(dp.IRAFirstStocks), downPaymentShortfall(dp.Amount, opts))
			fmt.Printf("  %s ($%s) from stocks first — IRA $%s left, stocks $%s left%s\n",
				pct, formatAccounting(dp.Amount), formatAccounting(dp.StocksFirstIRA), formatAccounting(dp.StocksFirstStocks), downPaymentShortfall(dp.Amount, opts))
		}
		fmt.Println("Note: IRA withdrawals are taxable income. These are the numbers, not tax advice.")
	}
	fmt.Println("")
}

// downPaymentShortfall names the gap when the down payment is bigger than
// the IRA and stocks put together; empty when it fits.
func downPaymentShortfall(amount float64, opts sheets_service.AffordabilityOptions) string {
	if shortfall := amount - opts.IRA - opts.Stocks; shortfall > 0 {
		return fmt.Sprintf(" (shortfall $%s — not enough in IRA + stocks)", formatAccounting(shortfall))
	}
	return ""
}

// formatWindowLabel renders the trailing window's month keys ("MM/YY") as
// a readable range, e.g. "May 2026 – Oct 2026".
func formatWindowLabel(months []string) string {
	parse := func(key string) string {
		t, err := time.Parse("01/06", key)
		if err != nil {
			return key
		}
		return t.Format("Jan 2006")
	}
	switch len(months) {
	case 0:
		return "no months"
	case 1:
		return parse(months[0])
	default:
		return parse(months[0]) + " – " + parse(months[len(months)-1])
	}
}
