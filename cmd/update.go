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
	Run: func(cmd *cobra.Command, args []string) {
		update(cmd, args)
	},
}

/**/
func init() {
	config, _ = cfg.ReadConfig(ConfigFile)
	rootCmd.AddCommand(updateCmd)

	// TODO: Fix default values

	//flag.BoolVarP(&options.Update, "no-updates", "u", false, "If set, no spreadsheet updates performed")
	//flag.BoolVarP(&options.UseCSVFiles, "csv", "c", false, "Read CSV files; default=false")
	//flag.Parse()

	updateCmd.Flags().BoolVarP(&options.Update, "no-updates", "u", false, "If set, no spreadsheet updates performed")
	updateCmd.Flags().BoolVarP(&options.UseCSVFiles, "csv", "c", false, "Read CSV files; default=false")
}

func update(cmd *cobra.Command, args []string) {
	var (
		client       *Client
		transactions []*models.Transaction
		err          error
	)

	conn, err := driver.ConnectSQL(&driver.ConnectParams{
		DBType: driver.DBType(config.DBType),
		Host:   config.DBHost,
		Port:   config.DBPort,
		DBName: config.DBName,
		User:   config.DBUsername,
		Pass:   config.DBPassword,
	})
	checkError(err)
	qHandler := handler.NewQueryHandler(conn)

	sheetsProvider, err := sheets_provider.New(options.SpreadsheetID, config)
	checkError(err)
	sheetsService := sheets_service.New(sheetsProvider)
	checkError(err)
	err = sheetsService.NewRegisterSheet(config)
	checkError(err)

	fmt.Println("Reading Register...")
	_, err = sheetsService.ReadRegisterSheet()
	checkError(err)

	client = getBankingClient()
	//if options.UseCSVFiles {
	//	fmt.Println("Getting transactions (CSV)...")
	//	transactions, err = getCSVTransactions()
	//	checkError(err)
	//} else {
	//	fmt.Println("Getting transactions (Plaid)...")
	//	transactions, err = getTransactions(client, options.BankIDs)
	//	checkError(err)
	//}

	fmt.Println("Getting Fidelity transactions (CSV)...")
	// TODO: limited csv:GetTransactions() to Fidelitty
	transactions, err = getCSVTransactions()
	checkError(err)

	fmt.Println("Getting Wells Fargo & Chase transactions (Plaid)...")
	options.BankIDs = []string{"wellsfargo", "chase"}
	tmp, err := getTransactions(client, options.BankIDs)
	checkError(err)

	transactions = append(transactions, tmp...)

	if len(transactions) < 1 {
		fmt.Println("No transactions")
		return
	}

	fmt.Println("Updating merchants...")
	lookupData := qHandler.GetLookupData()

	transactions = client.BankClient.FormatMerchantNames(transactions, lookupData)
	if options.Debug {
		printTransactions(transactions)
	}

	fmt.Printf("Filtering transactions...\n")
	transactions = client.BankClient.FilterRecordedTransactions(transactions, sheetsService.RegisterSheet.KeysMap)

	fmt.Printf("Sorting...\n")
	transactions = client.BankClient.SortTransactions(transactions)

	printTransactions(transactions)

	if !options.Update {
		fmt.Println("Updating transactions table...")
		qHandler.UpdateTransactionTables(transactions)
	}

	if needTransactionName(transactions) {
		fmt.Println("Info needed...")
		printColumns(qHandler)
		transactions, err = getBankNameToName(client.BankClient, qHandler, transactions)
		checkError(err)
	}
	transactions = getNotes(transactions)

	fmt.Printf("Reading Budget...\n")
	err = sheetsService.NewBudgetSheet(config)
	checkError(err)
	_, err = sheetsService.ReadBudgetSheet()
	checkError(err)

	// add the needed number of rows for transactions
	fmt.Println("Adding rows...")
	_, _, err = shellout("register copy -n " + strconv.Itoa(len(transactions)))
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	if len(transactions) > 0 {
		fmt.Printf("Transaction updates...\n")
		fmt.Printf("    (%3s) %-12s %-10s %8s %-30s %s\n", "Num", "Source", "Date", "Amount", "Name", "Note")
		fmt.Printf("    (%3s) %-12s %-10s %8s %-30s %s\n", dashes(3), dashes(12), dashes(18), dashes(8), dashes(30), dashes(15))
		for i, r := range transactions {
			fmt.Printf("    (%3d) %-12s %-10s %8.2f %-30s %s\n", i+1, r.Source, r.Date, -1*r.Amount, r.Name, r.Note)
		}
		if options.Update {
			return
		}

		fmt.Printf("Updating spreadsheet...\n")
		columns := qHandler.GetColumns()
		transNameToColName := qHandler.GetNameMapToColumn()

		err = sheetsService.UpdateRows(columns, transNameToColName, transactions)
		checkError(err)

		lastRowUpdated := sheetsService.RegisterSheet.SheetCoords.FirstRowToUpdate + int64(len(transactions)*2) + 1
		_, err = sheetsService.WriteCell("F1", time.Now().Format("01/02/2006"))
		checkError(err)
		_, err = sheetsService.WriteCell("G2", fmt.Sprintf("=SUM(G1-I%d)", lastRowUpdated))
		checkError(err)

		if !options.UseCSVFiles {
			fmt.Println("Getting accounts balances...")
			balances := client.BankClient.GetBalances(options.BankIDs)
			printBalances(balances)
			fmt.Println("Updating balances...")
			updateBalances(sheetsService, balances)
		}
	} else {
		fmt.Println("No updates needed")
	}
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

func updateBalances(sheetsService *sheets_service.SheetsService, balances map[string]banking.Balance) {
	if balances[banking.WellsFargoID].Error == nil {
		_, err := sheetsService.WriteCell("G1", balances[banking.WellsFargoID].Amount)
		checkError(err)
	}
	if balances[banking.FidelityID].Error == nil {
		_, err := sheetsService.WriteCell("AA2", balances[banking.FidelityID].Amount)
		checkError(err)
	}
	if balances[banking.ChaseID].Error == nil {
		_, err := sheetsService.WriteCell("AB2", balances[banking.ChaseID].Amount)
		checkError(err)
	}
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
	// start from 2 weeks ago
	startDate := weeksAgo(2)
	endDate := today()

	transactions, err := client.BankClient.GetTransactions(bankIDs, startDate, endDate)
	if err != nil {
		return nil, err
	}
	return transactions, nil
}

func getCSVTransactions() ([]*models.Transaction, error) {
	client := csv.New(csv.ConfigOptions{
		FinanceDir: config.FinanceDir,
		Banks:      config.Banks,
	})

	transactions, err := client.GetTransactions()
	if err != nil {
		return nil, err
	}
	return transactions, nil
}

func printTransactions(trans []*models.Transaction) {
	fmt.Printf("    (%3s) [%-28s] %-12s %-10s %-8s %-30s %s\n", "Num", "Key", "Source", "Date", "Amount", "Name", "Bank Name")
	for i, t := range trans {
		fmt.Printf("    (%3d) [%-28s] %-12s %-10s %8.2f %-30s %s\n", i+1, t.Key, t.Source, t.Date, -1*t.Amount, t.Name, t.BankName)
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

func printColumns(db *handler.Query) {
	columns := db.GetColumns()
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
		lookupData := db.GetLookupData()
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

			db.CreateMerchant(&models.Merchant{
				Name:     trans[i].Name,
				BankName: t.BankName,
				ColumnID: trans[i].ColumnIndex,
			})
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
