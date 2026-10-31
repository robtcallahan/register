package cmd

import (
	"fmt"
	"register/api/providers/sheets_provider"
	"register/api/services/sheets_service"
	"register/api/services/update_service"
	cfg "register/pkg/config"
	"register/pkg/csv"
	"register/pkg/models"
	"register/pkg/repository"
	"register/pkg/repository/mysql"
	"time"

	"github.com/spf13/cobra"
)

var backfillCmd = &cobra.Command{
	Use:   "backfill-transaction-ids",
	Short: "",
	RunE: func(cmd *cobra.Command, args []string) error {
		return backfill()
	},
}
var dryRun bool

func init() {
	rootCmd.AddCommand(backfillCmd)
	backfillCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print-only run: no DB or spreadsheet updates performed")
}

func backfill() error {
	conn, err := repository.Connect(&config.DB)
	if err != nil {
		return err
	}
	queryRepo := mysql.NewMySQLQueryRepo(conn)

	sheetsProvider, err := sheets_provider.New(options.SpreadsheetID, config)
	if err != nil {
		return err
	}

	bankClient := GetBankingClient()
	registerService := sheets_service.NewRegisterService(sheetsProvider)
	registerService.IncomeSources = config.IncomeSources
	budgetService := sheets_service.NewBudgetService(sheetsProvider)
	updateService := update_service.NewUpdateService(registerService, budgetService, bankClient, queryRepo, config, options)

	// define the listener for printing progress strings
	updateService.Progress = func(msg string) { fmt.Println(msg) }
	bankClient.Progress = func(msg string) { fmt.Println(msg) }

	fmt.Println("Getting Bank Transactions...")
	bankTrans, err := bankClient.GetTransactions(
		[]string{models.WellsFargoID, models.ChaseID},
		"2026-01-01",
		time.Now().Format("2006-01-02"),
	)

	fmt.Println("Getting CSV transactions...")
	const ConfigFile = "config/config.json"
	var config = &cfg.Config{}
	config, err = cfg.ReadConfig(ConfigFile)
	if err != nil {
		fmt.Println(err)
		return err
	}
	csvClient := csv.New(config.FinanceDir, config.Banks, []string{models.FidelityID}, func(_ string) {})
	csvClient.Progress = func(msg string) { fmt.Println(msg) }
	csvTrans, err := csvClient.GetTransactions()
	if err != nil {
		return err
	}

	allTrans := append(bankTrans, csvTrans...)
	fmt.Printf("bankTrans: %d, csvTrans: %d, total: %d\n", len(bankTrans), len(csvTrans), len(allTrans))

	if !dryRun {
		if err := queryRepo.UpdateTransactionTables(allTrans); err != nil {
			return fmt.Errorf("unable to insert transactions: %w", err)
		}
	}

	//fmt.Println("Getting Database transactions...")
	//dbTrans, err := queryRepo.GetTransactions()
	//if err != nil {
	//	return err
	//}
	//
	//dbByKey := make(map[string]bool, len(dbTrans))
	//for _, dbTran := range dbTrans {
	//	dbByKey[dbTran.Key] = true
	//}

	return nil
}
