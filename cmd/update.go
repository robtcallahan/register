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
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	// "os"
	// "strings"

	"register/pkg/banking"
	cfg "register/pkg/config"
	"register/pkg/sheets"

	"github.com/plaid/plaid-go/plaid"
	"github.com/spf13/cobra"
)

// updateCmd represents the get command
var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Get all back and CC transactions and update the Google Sheets register spreadsheet",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. `,
	Run: func(cmd *cobra.Command, args []string) {
		update(cmd, args)
	},
}

var config *cfg.Config

func init() {
	rootCmd.AddCommand(updateCmd)
	config = cfg.ReadConfig()

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// updateCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// updateCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	updateCmd.Flags().StringP("banks", "b", "wellsfargo,fidelity,costcocitivisa,chasevisa", "The desired bank CSV files to read")
	updateCmd.Flags().Int64P("start", "s", config.RegisterStartRow, "The last used row in the spreadsheet")
	updateCmd.Flags().Int64P("end", "e", config.RegisterEndRow, "The last used row in the spreadsheet")
	updateCmd.Flags().StringP("id", "i", config.SpreadsheetID, "The Google spreadsheet id")
	updateCmd.Flags().BoolP("debug", "d", false, "Debug mode")
}

func update(cmd *cobra.Command, args []string) {
	var err error

	// banksOpt, _ := cmd.Flags().GetString("banks")
	// banks := strings.Split(banksOpt, ",")

	ssID, _ := cmd.Flags().GetString("id")
	startRow, _ := cmd.Flags().GetInt64("start")
	endRow, _ := cmd.Flags().GetInt64("end")
	debug, _ := cmd.Flags().GetBool("debug")

	client := &banking.Client{
		Keys: &banking.Keys{
			Products:     "transactions",
			CountryCodes: "US",
		},
		StartDate: config.StartDate,
		EndDate:   config.EndDate,
		BankInfo:  config.BankInfo,
		Debug:     debug,
	}
	client.PlaidClient = func() *plaid.Client {
		client, err := plaid.NewClient(plaid.ClientOptions{
			ClientID:    config.PlaidClientID,
			Secret:      config.PlaidSecret,
			Environment: plaid.Development,
			HTTPClient:  &http.Client{},
		})
		checkError(err)
		return client
	}()

	srv := &sheets.SheetService{
		Service:       sheets.NewService(),
		SpreadsheetID: ssID,
	}

	reg := sheets.NewRegisterSheet(srv, *config, startRow, endRow)
	fmt.Printf("Reading Register...\n")
	reg.ID, err = srv.GetSheetID(config.TabNames["register"])
	checkError(err)
	reg.Read(debug)

	fmt.Println("Getting transactions...")
	reg.Transactions = client.GetTransactions()

	fmt.Printf("Sorting...\n")
	reg.SortByCSVDate()

	fmt.Printf("Reading Budget...\n")
	bud := sheets.NewBudgetSheet(srv, config.TabNames["budget"], config.BudgetStartRow, config.BudgetEndRow)
	bud.ID, err = srv.GetSheetID(config.TabNames["budget"])
	checkError(err)
	bud.Read()
	reg.CategoriesMap = bud.CategoriesMap

	fmt.Printf("Filtering rows...\n")
	reg.Transactions = client.FilterRows(reg.ValuesMap, reg.Transactions)

	if len(reg.Transactions) > 0 {
		fmt.Printf("Transaction updates...\n")
		for i, r := range reg.Transactions {
			reg.Transactions[i].Name = formatMerchants(r.Name)
			fmt.Printf("    (%2d) %-5s %-10s %8.2f %s\n", i, r.Source, r.Date, r.Amount, r.Name)
		}

		fmt.Printf("Updating spreadsheet...\n")
		reg.UpdateRows()
	} else {
		fmt.Println("No updates needed")
	}
}

func formatMerchants(merch string) string {
	for substr, replace := range config.Merchants {
		if strings.Contains(merch, substr) {
			return replace
		}
	}
	return merch
}

func formatDate(date string) string {
	re := regexp.MustCompile(`(20)?(\d\d)-(\d\d)-(\d\d)`)
	m := re.FindAllStringSubmatch(date, -1)
	yy, _ := strconv.Atoi(m[0][2])
	mm, _ := strconv.Atoi(m[0][3])
	dd, _ := strconv.Atoi(m[0][4])
	d := fmt.Sprintf("%02d/%02d/%02d", mm, dd, yy)
	return d
}

func checkError(err error) {
	if err != nil {
		panic(err)
	}
}
