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
	"fmt"
	"github.com/plaid/plaid-go/plaid"
	"math"
	"os"
	"regexp"
	"register/api/providers/sheets_provider"
	"strconv"
	"strings"
	"time"

	"register/api/services/sheets_service"
	"register/pkg/banking"
	cfg "register/pkg/config"
	"register/pkg/driver"
	"register/pkg/handler"
	"register/pkg/models"

	"github.com/spf13/cobra"
)

// updateCmd represents the get command
var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Reads bank transactions and updates the financial register spreadsheet",
	Long: `Register reads bank and credit card transactions from Wells Fargo, Fidelity, Chase,
and Citi, both the Register and Budget tabs from your Google Sheets financial spreadsheet,
removes duplicates and updates the Register tab with new transactions subtracting those
amounts from the appropriate budget category columns.`,
	Run: func(cmd *cobra.Command, args []string) {
		update()
	},
}

/**/
func init() {
	config = cfg.ReadConfig()
	rootCmd.AddCommand(updateCmd)
}

func update() {
	bankClient := banking.New(&banking.ClientOptions{
		BankInfo:      config.BankInfo,
		Debug:         options.Debug,
		Verbose:       options.Verbose,
		PlaidClientID: config.PlaidClientID,
		PlaidSecret:   config.PlaidSecret,
	})

	conn, err := driver.ConnectSQL(&driver.ConnectParams{
		DBType: driver.DBType(config.DBType),
		Host:   config.DBHost,
		Port:   config.DBPort,
		DBName: config.DBName,
		User:   config.DBUsername,
		Pass:   config.DBPassword,
	})
	if err != nil {
		panic(err)
	}
	qHandler := handler.NewQueryHandler(conn)

	sheetsService, err := sheets_service.New(sheets_provider.New(options.SpreadsheetID))
	checkError(err)
	err = sheetsService.NewRegisterSheet(config)

	fmt.Printf("Reading Register...\n")
	_, err = sheetsService.ReadRegisterSheet()
	checkError(err)
	if options.Verbose {
		printRegister(sheetsService.RegisterSheet.Register)
	}

	fmt.Println("Getting transactions...")
	transactions := bankClient.GetTransactions(options.BankKeys, config.StartDate, config.EndDate)
	if options.Verbose {
		printTransactions(transactions)
	}

	fmt.Println("Getting account balances...")
	accountInfo := map[string]*plaid.Account{}
	for _, name := range []string{"fidelity", "citi", "chase"} {
		accountInfo[name] = bankClient.GetAccount(config.BankInfo[name], "credit")
	}
	accountInfo["wellsfargo"] = bankClient.GetAccount(config.BankInfo["wellsfargo"], "depository")

	fmt.Println("Updating merchants...")
	lookupData := qHandler.GetLookupData()
	transactions = bankClient.FormatMerchants(transactions, lookupData)
	if options.Verbose {
		printTransactions(transactions)
	}

	fmt.Printf("Filtering rows...\n")
	transactions = bankClient.FilterRows(transactions, sheetsService.RegisterSheet.KeysMap)

	fmt.Printf("Sorting...\n")
	transactions = bankClient.SortTransactions(transactions)

	fmt.Println("Updating transactions table...")
	qHandler.UpdateTransactionTables(transactions)

	if needInfo := needInfo(transactions); needInfo {
		fmt.Println("Info needed...")
		transactions = getBankNameToName(qHandler, transactions)
	}

	fmt.Printf("Reading Budget...\n")
	err = sheetsService.NewBudgetSheet(config)
	checkError(err)
	_, err = sheetsService.ReadBudgetSheet()
	checkError(err)

	if len(transactions) > 0 {
		fmt.Printf("Transaction updates...\n")
		for i, r := range transactions {
			fmt.Printf("    (%2d) %-12s %-10s %8.2f %s\n", i+1, r.Source, r.Date, r.Amount, r.Name)
		}
		if options.Test {
			return
		}

		fmt.Printf("Updating spreadsheet...\n")
		cols := qHandler.GetColumns()
		nameToCol := qHandler.GetNameMapToColumn()
		sheetsService.UpdateRows(cols, nameToCol, transactions)

		lastRowUpdated := sheetsService.RegisterSheet.FirstRowToUpdate + int64(len(transactions) * 2) + 1
		sheetsService.WriteCell("F1", time.Now().Format("01/02/2006"))
		sheetsService.WriteCell("G2", fmt.Sprintf("=SUM(G1-I%d)", lastRowUpdated))
		sheetsService.WriteCell("G1", accountInfo["wellsfargo"].Balances.Available)

		sheetsService.WriteCell("AB2", accountInfo["fidelity"].Balances.Current)
		sheetsService.WriteCell("AC2", accountInfo["citi"].Balances.Current)
		sheetsService.WriteCell("AD2", accountInfo["chase"].Balances.Current)
	} else {
		fmt.Println("No updates needed")
	}
}

func printTransactions(trans []*models.Transaction) {
	for i, t := range trans {
		fmt.Printf("    (%2d) [%-28s] %-12s %-10s %8.2f %-30s %s\n", i+1, t.Key, t.Source, t.Date, t.Amount, t.Name, t.BankName)
	}
	fmt.Println("")
}

func printRegister(trans []*sheets_service.RegisterEntry) {
	for i, t := range trans {
		fmt.Printf("    (%2d) [%-28s] %-12s %-10s %8.2f %8.2f %8.2f %s\n", i+1, t.Key, t.Source, t.Date, t.Withdrawal, t.Deposit, t.CreditCard, t.Name)
	}
	fmt.Println("")
}

func needInfo(trans []*models.Transaction) bool {
	for _, t := range trans {
		if t.Name == "" && !strings.Contains(t.BankName, "CHECK #") {
			return true
		}
	}
	return false
}

//goland:noinspection GoNilness
func getBankNameToName(db *handler.Query, trans []*models.Transaction) []*models.Transaction {
	cols := db.GetColumns()
	var filter []models.Column

	// first, filter out old categories or those where IsCategory is false
	re := regexp.MustCompile(`\(old\)`)
	for _, c := range cols {
		chk := re.Match([]byte(c.Name))
		if !c.IsCategory || chk {
			continue
		}
		filter = append(filter, c)
	}

	// this will allow us to print 3 columns on the screen
	numRows := int(math.Floor(float64(len(filter)) / 3))
	remItems := len(filter) % 3

	i := 0
	for r := 1; r <= numRows; r++ {
		fmt.Printf("%2d %-30s %2d %-30s %2d %-30s\n",
			filter[i].ID, filter[i].Name,
			filter[i+1].ID, filter[i+1].Name,
			filter[i+2].ID, filter[i+2].Name,
		)
		i += 3
	}
	for ; i < remItems; i++ {
		fmt.Printf("%2d %-30s \n", filter[i].ID, filter[i].Name)
	}


	// prompt the user and read desired merchant name and the category index
	reader := bufio.NewReader(os.Stdin)
	for i, t := range trans {
		if t.Name == "" && !strings.Contains(t.BankName, "CHECK #") {
			fmt.Printf("    Bank Name: %s\n", t.BankName)

			fmt.Printf("    Name: ")
			name, _ := reader.ReadString('\n')
			name = strings.TrimSuffix(name, "\n")
			trans[i].Name = name

			fmt.Printf("    Column Index: ")
			s, _ := reader.ReadString('\n')
			s = strings.TrimSuffix(s, "\n")
			colInx, _ := strconv.Atoi(s)
			trans[i].ColumnIndex = colInx

			db.CreateMerchant(&models.Merchant{
				Name:     name,
				BankName: t.BankName,
				ColumnID: colInx,
			})
		}
	}
	return trans
}

func checkError(err error) {
	if err != nil {
		panic(err)
	}
}
