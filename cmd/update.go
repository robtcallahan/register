/*
Copyright © 2020 Rob Callahan <robtcallahan@aol.com>

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
	"bufio"
	stdcsv "encoding/csv"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"register/api/providers/sheets_provider"
	"register/api/services/sheets_service"
	"register/pkg/banking"
	cfg "register/pkg/config"
	"register/pkg/csv"
	"register/pkg/driver"
	"register/pkg/handler"
	"register/pkg/models"

	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Reads bank transactions and updates the financial register spreadsheet",
	Long: `Register reads bank and credit card transactions from Wells Fargo, Fidelity, Chase,
and Citi, both the Register and Budget tabs from your Google Sheets financial spreadsheet,
removes duplicates and updates the Register tab with new transactions subtracting those
amounts from the appropriate budget category columns.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return update(cmd, args)
	},
}

/**/
func init() {
	config, _ = cfg.ReadConfig(ConfigFile)
	rootCmd.AddCommand(updateCmd)

	updateCmd.Flags().BoolVar(&options.DryRun, "dry-run", false, "Print-only run: no DB or spreadsheet updates performed")
	updateCmd.Flags().BoolVarP(&options.UseCSVFiles, "csv", "c", false, "Read CSV files; default=false")
}

func update(cmd *cobra.Command, args []string) error {
	var (
		client       *Client
		transactions []*models.Transaction
		err          error
	)

	conn, err := driver.ConnectSQL(&driver.ConnectParams{
		DBType: driver.DBType(config.DB.Type),
		Host:   config.DB.Host,
		Port:   config.DB.Port,
		DBName: config.DB.Name,
		User:   config.DB.Username,
		Pass:   config.DB.Password,
	})
	if err != nil {
		return err
	}
	qHandler := handler.NewQueryHandler(conn)

	sheetsProvider, err := sheets_provider.New(options.SpreadsheetID, config)
	if err != nil {
		return err
	}
	sheetsService := sheets_service.New(sheetsProvider)
	sheetsService.IncomeSources = config.IncomeSources
	if err != nil {
		return err
	}
	columns, err := qHandler.GetColumns()
	if err != nil {
		return err
	}
	err = sheetsService.NewRegisterSheet(config, columns)
	if err != nil {
		return err
	}

	fmt.Println("Reading Register...")
	_, err = sheetsService.ReadRegisterSheet()
	if err != nil {
		return err
	}

	client = getBankingClient()

	transactions, err = fetchTransactions(client)
	if err != nil {
		return err
	}

	if len(transactions) < 1 {
		fmt.Println("No transactions")
		return nil
	}

	if err := writeRawTransactionsCSV(transactions); err != nil {
		return err
	}

	transactions, err = normalizeMerchants(client, qHandler, transactions)
	if err != nil {
		return err
	}

	transactions = dedupeAgainstSheet(client, transactions, sheetsService.RegisterSheet.KeysMap)

	fmt.Println("Filtering out credit card payments...")
	transactions = client.BankClient.FilterCreditCardPayments(transactions)

	fmt.Println("Sorting...")
	transactions = client.BankClient.SortTransactions(transactions)

	if len(transactions) < 1 {
		fmt.Println("No transactions")
		return nil
	}

	printTransactions(transactions)

	transactions, err = persistTransactions(client, qHandler, transactions)
	if err != nil {
		return err
	}

	if err := updateSpreadsheet(client, sheetsService, qHandler, transactions); err != nil {
		return err
	}
	return nil
}

// fetchTransactions gathers transactions from every bank: Fidelity always
// comes from its CSV (Plaid no longer supports it), while Wells Fargo and
// Chase come from CSV or Plaid depending on the --csv flag.
func fetchTransactions(client *Client) ([]*models.Transaction, error) {
	fmt.Println("Getting Fidelity transactions (CSV)...")
	transactions, err := getCSVTransactions([]string{"fidelity"})
	if err != nil {
		return nil, err
	}

	options.BankIDs = []string{"wellsfargo", "chase"}
	var newTrans []*models.Transaction
	if options.UseCSVFiles {
		fmt.Println("Getting Wells Fargo & Chase transactions (CSV)...")
		newTrans, err = getCSVTransactions(options.BankIDs)
	} else {
		fmt.Println("Getting Wells Fargo & Chase transactions (Plaid)...")
		newTrans, err = getTransactions(client, options.BankIDs)
	}
	if err != nil {
		return nil, err
	}
	return append(transactions, newTrans...), nil
}

// writeRawTransactionsCSV dumps the fetched transactions to transactions.csv
// in the finance directory. It runs before merchant normalization and sheet
// dedupe, so the file holds every transaction pulled from the banks,
// including ones already recorded in the register. encoding/csv quotes the
// fields, so bank names containing commas can't shift the columns.
func writeRawTransactionsCSV(transactions []*models.Transaction) error {
	fmt.Println("Writing raw transactions CSV file...")
	file, err := os.Create(config.FinanceDir + "/transactions.csv")
	if err != nil {
		return fmt.Errorf("error creating file %s: %v", config.FinanceDir+"/transactions.csv", err)
	}
	defer file.Close()

	w := stdcsv.NewWriter(file)
	for _, t := range transactions {
		record := []string{
			t.Source, t.Date, t.BankName,
			strconv.FormatFloat(t.Amount, 'f', 2, 64),
			strconv.FormatFloat(t.Deposit, 'f', 2, 64),
			strconv.FormatFloat(t.Withdrawal, 'f', 2, 64),
			strconv.FormatFloat(t.CreditCard, 'f', 2, 64),
		}
		if err := w.Write(record); err != nil {
			return fmt.Errorf("error writing to file: %v", err)
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return fmt.Errorf("error writing to file: %v", err)
	}
	return nil
}

// normalizeMerchants assigns each transaction its display name, color, and
// budget column from the merchant lookup data.
func normalizeMerchants(client *Client, qHandler *handler.Query, transactions []*models.Transaction) ([]*models.Transaction, error) {
	fmt.Println("Updating merchants...")
	lookupData, err := qHandler.GetLookupData()
	if err != nil {
		return nil, err
	}

	matcher, err := banking.NewMerchantMatcher(lookupData)
	if err != nil {
		return nil, err
	}

	income, err := client.BankClient.IncomeMatcher()
	if err != nil {
		return nil, err
	}

	columns, err := qHandler.GetColumns()
	if err != nil {
		return nil, err
	}

	transactions = client.BankClient.FormatMerchantNames(transactions, matcher, income, columns)
	if options.Debug {
		printTransactions(transactions)
	}
	return transactions, nil
}

// dedupeAgainstSheet drops transactions already recorded in the register
// sheet. Matching tolerates a ±1-day date shift, because some sources (notably
// Amazon) report the same transaction on a different day from one pull to the
// next, which changes its dedupe key.
func dedupeAgainstSheet(client *Client, transactions []*models.Transaction, keysMap map[string]bool) []*models.Transaction {
	fmt.Println("Filtering out register transactions...")
	return client.BankClient.FilterRecordedTransactions(transactions, fuzzyKeySet(keysMap))
}

// fuzzyKeySet returns a copy of keys with each key's date also shifted by -1
// and +1 days. Keys that don't parse as source:date:amount are kept as-is.
func fuzzyKeySet(keys map[string]bool) map[string]bool {
	fuzzy := make(map[string]bool, len(keys)*3)
	for k := range keys {
		parts := strings.SplitN(k, ":", 3)
		if len(parts) != 3 {
			fuzzy[k] = true
			continue
		}
		d, err := time.Parse("01/02/06", parts[1])
		if err != nil {
			fuzzy[k] = true
			continue
		}
		amt, err := strconv.ParseFloat(parts[2], 64)
		if err != nil {
			fuzzy[k] = true
			continue
		}
		for _, shift := range []int{-1, 0, 1} {
			date := d.AddDate(0, 0, shift).Format("01/02/06")
			fuzzy[models.TransactionKey(parts[0], date, amt)] = true
		}
	}
	return fuzzy
}

// persistTransactions interactively fills in any missing names and notes,
// then saves the transactions to the DB. All questions come first, so the rest
// of the run is unattended. Skipped entirely on --dry-run runs.
func persistTransactions(client *Client, qHandler *handler.Query, transactions []*models.Transaction) ([]*models.Transaction, error) {
	if options.DryRun {
		return transactions, nil
	}

	if needTransactionName(transactions) {
		fmt.Println("Info needed...")
		if err := printColumns(qHandler); err != nil {
			return nil, err
		}
		var err error
		transactions, err = getBankNameToName(client.BankClient, qHandler, transactions)
		if err != nil {
			return nil, err
		}
	}
	transactions = getNotes(transactions)

	fmt.Println("Updating transactions table...")
	if err := qHandler.UpdateTransactionTables(transactions); err != nil {
		return nil, err
	}
	return transactions, nil
}

// updateSpreadsheet writes the transactions into the register sheet and
// refreshes the bank balances. Skipped when there is nothing to write or on
// --dry-run runs.
func updateSpreadsheet(client *Client, sheetsService *sheets_service.SheetsService, qHandler *handler.Query, transactions []*models.Transaction) error {
	if len(transactions) == 0 || options.DryRun {
		fmt.Println("No updates needed")
		return nil
	}

	fmt.Printf("Reading Budget...\n")
	if err := sheetsService.NewBudgetSheet(config); err != nil {
		return err
	}
	if _, err := sheetsService.ReadBudgetSheet(); err != nil {
		return err
	}

	fmt.Printf("    (%3s) %-12s %-10s %8s %-30s %s\n", "Num", "Source", "Date", "Amount", "Name", "Note")
	fmt.Printf("    (%3s) %-12s %-10s %8s %-30s %s\n", dashes(3), dashes(12), dashes(10), dashes(8), dashes(30), dashes(15))
	for i, r := range transactions {
		fmt.Printf("    (%3d) %-12s %-10s %8.2f %-30s %s\n", i+1, r.Source, r.Date, -1*r.Amount, r.Name, r.Note)
	}

	// add the needed number of rows for transactions
	fmt.Println("Adding rows...")
	if err := sheetsService.CopyRows(len(transactions)); err != nil {
		return err
	}

	fmt.Printf("Transaction updates...\n")

	fmt.Printf("Updating spreadsheet...\n")
	columns, err := qHandler.GetColumns()
	if err != nil {
		return err
	}
	transNameToColName, err := qHandler.GetNameMapToColumn()
	if err != nil {
		return err
	}

	if err := sheetsService.UpdateRows(columns, transNameToColName, transactions); err != nil {
		return err
	}

	lastRowUpdated := sheetsService.RegisterSheet.SheetCoords.FirstRowToUpdate + int64(len(transactions)*2) + 1
	if _, err := sheetsService.WriteCell(sheets_service.LastUpdateCell, time.Now().Format("01/02/2006")); err != nil {
		return err
	}
	if _, err := sheetsService.WriteCell(sheets_service.CheckingDeltaCell, fmt.Sprintf("=SUM(%s-I%d)", sheets_service.CheckingBalanceCell, lastRowUpdated)); err != nil {
		return err
	}

	if !options.UseCSVFiles {
		fmt.Println("Getting accounts balances...")
		balances := client.BankClient.GetBalances([]string{banking.WellsFargoID})
		printBalances(balances)
		fmt.Println("Updating balances...")
		if err := updateBalances(sheetsService, balances); err != nil {
			return err
		}
	}
	return nil
}

func updateBalances(sheetsService *sheets_service.SheetsService, balances map[string]banking.Balance) error {
	if balances[banking.WellsFargoID].Error == nil {
		if _, err := sheetsService.WriteCell(sheets_service.CheckingBalanceCell, balances[banking.WellsFargoID].Amount); err != nil {
			return err
		}
	}
	return nil
}

func dashes(count int) string {
	return strings.Repeat("-", count)
}

func printBalances(balances map[string]banking.Balance) {
	fmt.Printf("    Wells Fargo: $%8.2f\n", balances[banking.WellsFargoID].Amount)
}

func getTransactions(client *Client, bankIDs []string) ([]*models.Transaction, error) {
	//startDate := weeksAgo(16)
	//endDate := today()
	startDate := config.StartDate
	endDate := config.EndDate

	transactions, err := client.BankClient.GetTransactions(bankIDs, startDate, endDate)
	if err != nil {
		return nil, err
	}
	return transactions, nil
}

func getCSVTransactions(bankIDs []string) ([]*models.Transaction, error) {
	client := csv.New(csv.ConfigOptions{
		FinanceDir: config.FinanceDir,
		Banks:      config.Banks,
		BankIDs:    bankIDs,
	})

	transactions, err := client.GetTransactions()
	if err != nil {
		return nil, err
	}
	return transactions, nil
}

func printTransactions(trans []*models.Transaction) {
	fmt.Printf("    (%3s) [%-28s] %-12s %-10s %-8s %2s %-30s %-30s\n", "Num", "Key", "Source", "Date", "Amount", "CI", "Name", "Bank Name")
	for i, t := range trans {
		amt := 0.0
		if t.Source == "WellsFargo" {
			if t.Deposit != 0 {
				amt = t.Deposit
			} else {
				amt = t.Withdrawal
			}
		} else {
			amt = t.CreditPurchase
		}
		fmt.Printf("    (%3d) [%-28s] %-12s %-10s %8.2f %2d %-30s %-30s\n", i+1, t.Key, t.Source, t.Date, amt, t.ColumnIndex, t.Name, t.BankName)
	}
	fmt.Println("")
}

func printRegister(trans []*sheets_service.RegisterEntry) {
	for i, t := range trans {
		fmt.Printf("    (%2d) [%-28s] %-12s %-10s %8.2f %8.2f %8.2f %s\n", i+1, t.Key, t.Source, t.Date, t.Withdrawal, t.Deposit, t.CreditCard, t.Name)
	}
	fmt.Println("")
}

func needTransactionName(trans []*models.Transaction) bool {
	for _, t := range trans {
		if t.Name == "" {
			return true
		}
	}
	return false
}

func printColumns(db *handler.Query) error {
	columns, err := db.GetColumns()
	if err != nil {
		return err
	}
	filtered := filterNonCategoryColumns(columns)

	// this will allow us to print 3 columns on the screen
	numRows := int(math.Floor(float64(len(filtered)) / 3))
	remItems := len(filtered) % 3

	i := 0
	for r := 1; r <= numRows; r++ {
		fmt.Printf("%2d %-30s %2d %-30s %2d %-30s\n",
			filtered[i].ID, filtered[i].Name,
			filtered[i+1].ID, filtered[i+1].Name,
			filtered[i+2].ID, filtered[i+2].Name,
		)
		i += 3
	}
	for j := 0; j < remItems; i++ {
		fmt.Printf("%2d %-30s \n", filtered[i].ID, filtered[i].Name)
		j++
	}
	return nil
}

func filterNonCategoryColumns(columns []models.Column) []models.Column {
	var filtered []models.Column
	for _, col := range columns {
		if !col.IsCategory() {
			continue
		}
		filtered = append(filtered, col)
	}
	return filtered
}

func getNotes(trans []*models.Transaction) []*models.Transaction {
	for i, t := range trans {
		if t.Name == "CHECK" || t.Name == "Amazon" || t.Name == "Amazon Marketplace" {
			fmt.Printf("Source: %s, Name: %s, Date: %s, Amt: $%0.2f\n", t.Source, t.Name, t.Date, t.Amount)
			trans[i].Note = readString("    Note: ")
		}
	}
	return trans
}

func getBankNameToName(bankClient *banking.Client, db *handler.Query, trans []*models.Transaction) ([]*models.Transaction, error) {
	var err error
	updated := true

	for updated {
		updated, trans, err = readFromUser(db, trans)
		if err != nil {
			return nil, err
		}
		lookupData, err := db.GetLookupData()
		if err != nil {
			return nil, err
		}
		matcher, err := banking.NewMerchantMatcher(lookupData)
		if err != nil {
			return nil, err
		}
		income, err := bankClient.IncomeMatcher()
		if err != nil {
			return nil, err
		}
		columns, err := db.GetColumns()
		if err != nil {
			return nil, err
		}
		trans = bankClient.FormatMerchantNames(trans, matcher, income, columns)
	}
	return trans, nil
}

func readFromUser(db *handler.Query, trans []*models.Transaction) (bool, []*models.Transaction, error) {
	var err error

	for i, t := range trans {
		if t.Name == "" && !strings.Contains(t.BankName, "CHECK #") {
			fmt.Printf("Source: %s, Date: %s, Amt: $%0.2f\n", t.BankName, t.Date, t.Amount)

			trans[i].Name = readString("            Name: ")
			var columnID int
			for err = fmt.Errorf(""); err != nil; {
				columnID, err = readInt("       Column ID: ")
				if err != nil {
					fmt.Println(err.Error())
				}
			}
			trans[i].Note = readString("           Note: ")

			// the menu prints column IDs, but the transaction wants the
			// column's position — translate instead of sharing one number
			columns, err := db.GetColumns()
			if err != nil {
				return false, nil, err
			}
			if index, ok := models.NewColumnSet(columns).IndexForID(columnID); ok {
				trans[i].ColumnIndex = index
			}

			if err := db.CreateMerchant(&models.Merchant{
				Name:     trans[i].Name,
				BankName: t.BankName,
				ColumnID: columnID,
			}); err != nil {
				return false, nil, err
			}
			return true, trans, nil
		}
	}
	return false, trans, nil
}

func readString(prompt string) string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print(prompt)
	v, _ := reader.ReadString('\n')
	return strings.TrimSuffix(v, "\n")
}

func readInt(prompt string) (int, error) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print(prompt)
	v, _ := reader.ReadString('\n')
	v = strings.TrimSuffix(v, "\n")
	i, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("could not convert %s to an int: %s", v, err.Error())
	}
	return i, nil
}
