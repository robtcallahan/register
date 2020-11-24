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
	// "os"
	// "strings"

	"register/pkg/banking"
	cfg "register/pkg/config"
	"register/pkg/sheets"

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
	updateCmd.Flags().BoolP("test", "t", false, "Test mode; no updates performed")
}

func update(cmd *cobra.Command, args []string) {
	var err error
	ssID, _ := cmd.Flags().GetString("id")
	startRow, _ := cmd.Flags().GetInt64("start")
	endRow, _ := cmd.Flags().GetInt64("end")
	debug, _ := cmd.Flags().GetBool("debug")
	test, _ := cmd.Flags().GetBool("test")

	client := banking.New(&banking.ClientOptions{
		StartDate:     config.StartDate,
		EndDate:       config.EndDate,
		BankInfo:      config.BankInfo,
		Debug:         debug,
		PlaidClientID: config.PlaidClientID,
		PlaidSecret:   config.PlaidSecret,
		Merchants:     config.Merchants,
	})

	srv := &sheets.SheetService{
		Service:       sheets.NewService(),
		SpreadsheetID: ssID,
	}

	reg := sheets.NewRegisterSheet(srv, *config, startRow, endRow)
	fmt.Printf("Reading Register...\n")
	id, err := srv.GetSheetID(config.TabNames["register"])
	reg.ID = id
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
			fmt.Printf("    (%2d) %-5s %-10s %8.2f %s\n", i+1, r.Source, r.Date, r.Amount, r.Name)
		}

		if !test {
			fmt.Printf("Updating spreadsheet...\n")
			reg.UpdateRows()
		}
	} else {
		fmt.Println("No updates needed")
	}
}

func checkError(err error) {
	if err != nil {
		panic(err)
	}
}
