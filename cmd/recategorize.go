package cmd

import (
	"fmt"

	"register/api/providers/sheets_provider"
	"register/api/services/sheets_service"
	"register/pkg/models"
	"register/pkg/repository"
	"register/pkg/repository/mysql"

	"github.com/spf13/cobra"
)

// recategorizeCmd moves transactions between category columns, in the sheet
// and (best-effort) in the DB. The sheet leads: rows are found by matching
// the Description column, and each row's category amount is relocated.
var recategorizeCmd = &cobra.Command{
	Use:   "recategorize",
	Short: "Move transactions to a different category column",
	Long: `Move transactions to a different category column, in the register sheet
and in the transactions table. Rows are found by case-insensitive substring
match on the Description column (--match may be repeated). Honors --dry-run.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return recategorize()
	},
}

var recategorizeTo string
var recategorizeMatches []string

func init() {
	recategorizeCmd.Flags().StringVar(&recategorizeTo, "to", "", "target category column name (required)")
	recategorizeCmd.Flags().StringSliceVar(&recategorizeMatches, "match", nil, "case-insensitive substring to match on Description (repeatable, required)")
	recategorizeCmd.Flags().BoolVar(&options.DryRun, "dry-run", false, "Print-only run: no DB or spreadsheet updates performed")
	_ = recategorizeCmd.MarkFlagRequired("to")
	_ = recategorizeCmd.MarkFlagRequired("match")
	rootCmd.AddCommand(recategorizeCmd)
}

func recategorize() error {
	conn, err := repository.Connect(&config.DB)
	if err != nil {
		return err
	}
	qHandler := mysql.NewMySQLQueryRepo(conn)

	cols, err := qHandler.GetColumns()
	if err != nil {
		return err
	}
	var target *models.Column
	for i := range cols {
		if cols[i].Name == recategorizeTo {
			target = &cols[i]
			break
		}
	}
	if target == nil {
		return fmt.Errorf("no column named %q", recategorizeTo)
	}
	if !target.IsCategory() {
		return fmt.Errorf("column %q is not a category", recategorizeTo)
	}

	sheetsProvider, err := sheets_provider.New(options.SpreadsheetID, config)
	if err != nil {
		return err
	}
	sheetsService := sheets_service.NewRegisterService(sheetsProvider)
	registerSheet, err := sheetsService.NewRegisterSheet(config, cols)
	if err != nil {
		return err
	}
	if _, err := sheetsService.ReadRegisterSheet(registerSheet); err != nil {
		return err
	}

	moves, notes := sheetsService.PlanCategoryMoves(*registerSheet, recategorizeMatches, *target, cols)
	for _, n := range notes {
		fmt.Println("note:", n)
	}
	if len(moves) == 0 {
		fmt.Println("nothing to move")
		return nil
	}
	for _, m := range moves {
		fmt.Printf("move row %d: %.2f from column %d to %s\n", m.RowID, m.Value, m.FromIndex, target.Name)
	}
	if options.DryRun {
		fmt.Println("dry run: no changes made")
		return nil
	}

	if err := sheetsService.MoveCategoryCells(*registerSheet, moves); err != nil {
		return fmt.Errorf("sheet move failed: %w", err)
	}

	// best-effort DB update so the transactions table agrees with the sheet
	all, err := qHandler.GetTransactions()
	if err != nil {
		return fmt.Errorf("sheet updated, but could not read transactions for the DB update: %w", err)
	}
	byKey := make(map[string]*models.Transaction)
	for i := range all {
		byKey[all[i].Key] = &all[i]
	}
	var dbUpdates []*models.Transaction
	for _, m := range moves {
		t, ok := byKey[m.Key]
		if !ok {
			fmt.Printf("note: row %d has no matching DB transaction, DB left as-is\n", m.RowID)
			continue
		}
		t.ColumnIndex = target.ColumnIndex
		t.Color = target.Color
		dbUpdates = append(dbUpdates, t)
	}
	if len(dbUpdates) > 0 {
		if err := qHandler.UpdateTransactionTables(dbUpdates); err != nil {
			return fmt.Errorf("sheet updated, but DB update failed: %w", err)
		}
	}
	fmt.Printf("moved %d transaction(s) to %s\n", len(moves), target.Name)
	return nil
}
