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
	"bytes"
	"fmt"
	"math"
	"os"
	"os/exec"
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

	updateCmd.Flags().BoolVarP(&options.Update, "no-updates", "u", false, "If set, no spreadsheet updates performed")
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
	if err != nil {
		return err
	}
	err = sheetsService.NewRegisterSheet(config)
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

	fmt.Println("Writing CSV file...")
	file, err := os.Create(config.FinanceDir + "/transactions.csv")
	if err != nil {
		return fmt.Errorf("error creating file %s: %v", config.FinanceDir+"/transactions.csv", err)
	}
	defer file.Close() // Important: always close the file
	for _, t := range transactions {
		_, err = file.WriteString(fmt.Sprintf("%s,%s,%s,%0.2f,%0.2f,%0.2f,%0.2f\n",
			t.Source, t.Date, t.BankName, t.Amount, t.Deposit, t.Withdrawal, t.CreditCard))
		if err != nil {
			return fmt.Errorf("error writing to file: %v", err)
		}
	}

	transactions, err = normalizeMerchants(client, qHandler, transactions)
	if err != nil {
		return err
	}

	transactions = dedupeAgainstSheet(client, transactions, sheetsService.RegisterSheet.KeysMap)

	fmt.Println("Correcting transaction names that are non-generic...")
	transactions = client.BankClient.FormatUniqueTransactionNames(transactions)

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

	if len(transactions) > 0 && !options.Update {
		fmt.Printf("Reading Budget...\n")
		err = sheetsService.NewBudgetSheet(config)
		if err != nil {
			return err
		}
		_, err = sheetsService.ReadBudgetSheet()
		if err != nil {
			return err
		}

		fmt.Printf("    (%3s) %-12s %-10s %8s %-30s %s\n", "Num", "Source", "Date", "Amount", "Name", "Note")
		fmt.Printf("    (%3s) %-12s %-10s %8s %-30s %s\n", dashes(3), dashes(12), dashes(10), dashes(8), dashes(30), dashes(15))
		for i, r := range transactions {
			fmt.Printf("    (%3d) %-12s %-10s %8.2f %-30s %s\n", i+1, r.Source, r.Date, -1*r.Amount, r.Name, r.Note)
		}

		// add the needed number of rows for transactions
		fmt.Println("Adding rows...")
		out, errOut, err := shellout("./register copy -n " + strconv.Itoa(len(transactions)))
		if err != nil {
			return fmt.Errorf("error: %v\n--- stdout ---\n%s\n--- stderr ---\n%s", err, out, errOut)
		}

		fmt.Printf("Transaction updates...\n")
		if options.Update {
			return nil
		}

		fmt.Printf("Updating spreadsheet...\n")
		columns, err := qHandler.GetColumns()
		if err != nil {
			return err
		}
		transNameToColName, err := qHandler.GetNameMapToColumn()
		if err != nil {
			return err
		}

		err = sheetsService.UpdateRows(columns, transNameToColName, transactions)
		if err != nil {
			return err
		}

		lastRowUpdated := sheetsService.RegisterSheet.SheetCoords.FirstRowToUpdate + int64(len(transactions)*2) + 1
		_, err = sheetsService.WriteCell("F1", time.Now().Format("01/02/2006"))
		if err != nil {
			return err
		}
		_, err = sheetsService.WriteCell("G2", fmt.Sprintf("=SUM(G1-I%d)", lastRowUpdated))
		if err != nil {
			return err
		}

		if !options.UseCSVFiles {
			fmt.Println("Getting accounts balances...")
			balances := client.BankClient.GetBalances(options.BankIDs)
			printBalances(balances)
			fmt.Println("Updating balances...")
			if err := updateBalances(sheetsService, balances); err != nil {
				return err
			}
		}
	} else {
		fmt.Println("No updates needed")
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

// normalizeMerchants assigns each transaction its display name, color, and
// budget column from the merchant lookup data.
func normalizeMerchants(client *Client, qHandler *handler.Query, transactions []*models.Transaction) ([]*models.Transaction, error) {
	fmt.Println("Updating merchants...")
	lookupData, err := qHandler.GetLookupData()
	if err != nil {
		return nil, err
	}

	transactions = client.BankClient.FormatMerchantNames(transactions, lookupData)
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
		for _, shift := range []int{-1, 0, 1} {
			variant := fmt.Sprintf("%s:%s:%s", parts[0], d.AddDate(0, 0, shift).Format("01/02/06"), parts[2])
			fuzzy[variant] = true
		}
	}
	return fuzzy
}

// persistTransactions saves transactions to the DB and interactively fills
// in any missing names and notes. Skipped entirely on --no-updates runs.
func persistTransactions(client *Client, qHandler *handler.Query, transactions []*models.Transaction) ([]*models.Transaction, error) {
	if options.Update {
		return transactions, nil
	}

	fmt.Println("Updating transactions table...")
	if err := qHandler.UpdateTransactionTables(transactions); err != nil {
		return nil, err
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
	return transactions, nil
}

func shellout(command string) (string, string, error) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := exec.Command("bash", "-c", command)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

func updateBalances(sheetsService *sheets_service.SheetsService, balances map[string]banking.Balance) error {
	if balances[banking.WellsFargoID].Error == nil {
		if _, err := sheetsService.WriteCell("G1", balances[banking.WellsFargoID].Amount); err != nil {
			return err
		}
	}
	//if balances[banking.FidelityID].Error == nil {
	//	_, err := sheetsService.WriteCell("AA2", balances[banking.FidelityID].Amount)
	//	checkError(err)
	//}
	if balances[banking.ChaseID].Error == nil {
		if _, err := sheetsService.WriteCell("AB2", balances[banking.ChaseID].Amount); err != nil {
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
	fmt.Printf("    Fidelity:    $%8.2f\n", balances[banking.FidelityID].Amount)
	fmt.Printf("    Chase:       $%8.2f\n", balances[banking.ChaseID].Amount)
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
		if !col.IsCategory {
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
		trans = bankClient.FormatMerchantNames(trans, lookupData)
	}
	return trans, nil
}

func readFromUser(db *handler.Query, trans []*models.Transaction) (bool, []*models.Transaction, error) {
	var err error

	for i, t := range trans {
		if t.Name == "" && !strings.Contains(t.BankName, "CHECK #") {
			fmt.Printf("Source: %s, Date: %s, Amt: $%0.2f\n", t.BankName, t.Date, t.Amount)

			trans[i].Name = readString("            Name: ")
			for err = fmt.Errorf(""); err != nil; {
				trans[i].ColumnIndex, err = readInt("    Column Index: ")
				if err != nil {
					fmt.Println(err.Error())
				}
			}
			trans[i].Note = readString("           Note: ")

			if err := db.CreateMerchant(&models.Merchant{
				Name:     trans[i].Name,
				BankName: t.BankName,
				ColumnID: trans[i].ColumnIndex,
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
