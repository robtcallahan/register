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
	"math"
	"os"
	"regexp"
	"strconv"
	"strings"

	"register/pkg/banking"
	cfg "register/pkg/config"
	repo "register/pkg/repository"
	"register/pkg/sheets"

	"github.com/spf13/cobra"
)

// updateCmd represents the get command
var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Reads bank transactions and updates the financial register spreadsheet",
	Long: `Register reads bank and credit card transactions from Wells Fargo, Fidlity, Chase,
and Citi, both the Register and Budget tabs from your Google Sheets financial spreadsheet,
removes duplicates and updates the Register tab with new transactions subtracting those
amounts from the appropriate budget category columns.`,
	Run: func(cmd *cobra.Command, args []string) {
		update(cmd, args)
	},
}

func init() {
	config = cfg.ReadConfig()
	rootCmd.AddCommand(updateCmd)
}

func update(cmd *cobra.Command, args []string) {
	var err error

	bankClient := banking.New(&banking.ClientOptions{
		StartDate:     config.StartDate,
		EndDate:       config.EndDate,
		BankInfo:      config.BankInfo,
		Debug:         Debug,
		PlaidClientID: config.PlaidClientID,
		PlaidSecret:   config.PlaidSecret,
	})

	db := repo.NewRepository(repo.NewRepositoryParams{
		Debug:      Debug,
		DBName:     config.DBName,
		DBUsername: config.DBUsername,
		DBPassword: config.DBPassword,
	})

	sheetService := &sheets.SheetService{
		Service:       sheets.NewService(),
		SpreadsheetID: SpreadsheetID,
	}

	fmt.Printf("Reading Register...\n")
	regSrv := sheets.NewRegisterSheet(sheetService, *config, StartRow, EndRow, Debug)
	regSrv.ID, err = sheetService.GetSheetID(config.TabNames["register"])
	checkError(err)
	regSrv.Register, regSrv.KeysMap, _ = regSrv.Read()

	fmt.Println("Getting transactions...")
	transactions := bankClient.GetTransactions()

	fmt.Printf("Sorting...\n")
	transactions = bankClient.SortTransactions(transactions)

	fmt.Printf("Filtering rows...\n")
	transactions = bankClient.FilterRows(transactions, regSrv.KeysMap)

	fmt.Println("Updating merchants...")
	lookupData := db.GetLookupData()
	transactions = bankClient.FormatMerchants(transactions, lookupData)

	if len(transactions) > 0 {
		fmt.Printf("Transaction updates...\n")
		if Debug {
			printTransactions(bankClient, transactions)
		}
	}

	if needInfo := needInfo(transactions); needInfo {
		fmt.Println("Info needed...")
		transactions = getBankNameToName(db, transactions)
	}

	fmt.Printf("Reading Budget...\n")
	budget := sheets.NewBudgetSheet(sheetService, config.TabNames["budget"], config.BudgetStartRow, config.BudgetEndRow)
	budget.ID, err = sheetService.GetSheetID(config.TabNames["budget"])
	checkError(err)
	budget.Read()
	regSrv.CategoriesMap = budget.CategoriesMap

	if len(transactions) > 0 {
		fmt.Printf("Transaction updates...\n")
		for i, r := range transactions {
			fmt.Printf("    (%2d) %-5s %-10s %8.2f %s\n", i+1, r.Source, r.Date, r.Amount, r.Name)
		}

		if !Test {
			fmt.Printf("Updating spreadsheet...\n")
			cols := db.GetColumns()
			nameToCol := db.GetNameMapToColumn()
			regSrv.UpdateRows(cols, nameToCol, transactions)
		}
	} else {
		fmt.Println("No updates needed")
	}
}

func needInfo(trans []*banking.Transaction) bool {
	for _, t := range trans {
		if t.Name == "" {
			return true
		}
	}
	return false
}

func getBankNameToName(db *mysql.db, trans []*banking.Transaction) []*banking.Transaction {
	cols := db.GetColumns()
	filter := []repo.Column{}
	re := regexp.MustCompile(`old-\d+`)
	for _, c := range cols {
		chk := re.Match([]byte(c.Name))
		if !c.IsCategory || chk {
			continue
		}
		filter = append(filter, c)
	}
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

	reader := bufio.NewReader(os.Stdin)
	for i, t := range trans {
		if t.Name == "" {
			fmt.Printf("    Bank Name: %s\n", t.BankName)

			fmt.Printf("    Name: ")
			name, _ := reader.ReadString('\n')
			trans[i].Name = strings.Replace(name, "\n", "", -1)

			fmt.Printf("    Column Index: ")
			s, _ := reader.ReadString('\n')
			s = strings.Replace(s, "\n", "", -1)
			colInx, _ := strconv.Atoi(s)
			trans[i].ColumnIndex = colInx

			db.CreateMerchant(&repo.Merchant{
				Name:     name,
				BankName: t.BankName,
				ColumnID: colInx,
			})
		}
	}
	return trans
}

func printTransactions(client *banking.Client, trans []*banking.Transaction) {
	fmt.Println("")
	client.PrintTransactionHead()
	for i, t := range trans {
		t.PrintTransaction(i)
	}
}

func checkError(err error) {
	if err != nil {
		panic(err)
	}
}
