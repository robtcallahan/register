package cmd

import (
	"fmt"

	"register/api/providers/sheets_provider"
	"register/api/services/sheets_service"

	"github.com/spf13/cobra"
)

var budgetCmd = &cobra.Command{
	Use:   "budget",
	Short: "Budget sheet commands",
}

var budgetCheckCmd = &cobra.Command{
	Use:   "check",
	Short: "Validate the Budget sheet layout (read-only)",
	Long: `Checks that the Budget tab has the expected headers.
Run this after editing the Budget sheet to catch layout
problems before a real update.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return checkBudget()
	},
}

func init() {
	rootCmd.AddCommand(budgetCmd)
	budgetCmd.AddCommand(budgetCheckCmd)
}

func checkBudget() error {
	provider, err := sheets_provider.New(options.SpreadsheetID, config)
	if err != nil {
		return err
	}
	bs := sheets_service.NewBudgetService(provider)

	budgetSheet, err := bs.NewBudgetSheet(*config)
	if err != nil {
		return fmt.Errorf("could not initialize budget sheet: %w", err)
	}

	results, err := bs.CheckBudgetLayout(*budgetSheet)
	if err != nil {
		return err
	}

	allOk := true
	for _, r := range results {
		mark := "✓"
		if !r.Ok {
			mark = "✗"
			allOk = false
		}
		fmt.Printf("%s %s\n", mark, r.Message)
	}

	if !allOk {
		return fmt.Errorf("budget sheet has layout problems (see above)")
	}
	fmt.Println("Budget sheet OK")
	return nil
}
