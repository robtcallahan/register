package cmd

import (
	"fmt"
	"register/api/providers/sheets_provider"
	"register/api/services/sheets_service"
	"register/pkg/models"
	"register/pkg/repository"
	"register/pkg/repository/mysql"
	"strings"

	"github.com/spf13/cobra"
	"google.golang.org/api/sheets/v4"
)

var popTransIDCmd = &cobra.Command{
	Use:   "populate-transaction-ids",
	Short: "",
	RunE: func(cmd *cobra.Command, args []string) error {
		return popTransID()
	},
}
var dryRun bool

func init() {
	rootCmd.AddCommand(popTransIDCmd)
	popTransIDCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print-only run: no DB or spreadsheet updates performed")
}

func popTransID() error {
	conn, err := repository.Connect(&config.DB)
	if err != nil {
		return err
	}
	queryRepo := mysql.NewMySQLQueryRepo(conn)

	sheetsProvider, err := sheets_provider.New(options.SpreadsheetID, config)
	if err != nil {
		return err
	}
	registerService := sheets_service.NewRegisterService(sheetsProvider)
	registerSheet, err := registerService.NewRegisterSheet(config, nil)
	if err != nil {
		return err
	}
	registerSheet, err = registerService.ReadRegisterSheet(registerSheet)

	fmt.Printf("Num register entries: %d\n", len(registerSheet.Entries))

	dbTrans, err := queryRepo.GetTransactions()
	if err != nil {
		return err
	}
	fmt.Printf("Num database transactions: %d\n", len(dbTrans))

	dbByKey := make(map[string][]*models.Transaction)
	for i := range dbTrans {
		dbByKey[dbTrans[i].Key] = append(dbByKey[dbTrans[i].Key], &dbTrans[i])
	}
	fmt.Printf("Num dbByKey: %d\n", len(dbByKey))

	first := registerSheet.Entries[0].RowID
	last := registerSheet.Entries[len(registerSheet.Entries)-1].RowID
	rows := make([][]any, last-first+1)
	for i := range rows {
		rows[i] = []any{""}
	}

	var rowsUpdated = 0
	var noMatchIdx = 0
	var noMatches = make(map[int]*sheets_service.RegisterEntry)

	for id, regRow := range registerSheet.Entries {
		matches := dbByKey[regRow.Key]
		switch {
		case len(matches) == 1 && matches[0].TransactionID != "":
			// unique match with an ID — write matches[0].TransactionID
			rows[regRow.RowID-first] = []any{id}
			rowsUpdated = rowsUpdated + 1
		case len(matches) > 1:
			// twin — leave AR empty for manual resolution
		default:
			// no match, or the one match has an empty ID — leave empty
			if strings.HasPrefix(regRow.Name, "VOID") ||
				strings.HasPrefix(regRow.Name, "Reallocation") {
				continue
			}
			noMatches[noMatchIdx] = regRow
			noMatchIdx = noMatchIdx + 1
		}
	}
	writeRange := fmt.Sprintf("%s!AR%d:AR%d", registerSheet.TabName, first, last)
	fmt.Printf("Will write %d Transaction IDs to %s\n", rowsUpdated, writeRange)

	if dryRun {
		fmt.Println("Dry-run mode: no spreadsheet updates performed")
		fmt.Printf("Unmatched Register entries (%d):\n", len(noMatches))
		for i := range noMatches {
			fmt.Printf("%-30s %s\n", noMatches[i].Key, noMatches[i].Name)
		}
		return nil
	}

	fmt.Printf("Writing %d transactions to %s\n", len(rows), writeRange)
	sheetsProvider.Update(writeRange, &sheets.ValueRange{Values: rows})

	return nil
}
