/*
Copyright © 2020 NAME HERE <EMAIL ADDRESS>

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
	"os"
	"strings"

	"register/pkg/banking"
	cfg "register/pkg/config"
	"register/pkg/csv"
	"register/pkg/sheets"

	"github.com/plaid/plaid-go/plaid"
	"github.com/spf13/cobra"
)

// getCmd represents the get command
var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get all back and CC transactions and update the Google Sheets register spreadsheet",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. `,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Calling getTrans()...")
		getTrans(cmd, args)
	},
}

var config *cfg.Config

func init() {
	rootCmd.AddCommand(getCmd)
	config = cfg.ReadConfig()

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// getCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// getCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	getCmd.Flags().StringP("banks", "b", "wellsfargo,fidelity,costcocitivisa,chasevisa", "The desired bank CSV files to read")
	getCmd.Flags().IntP("copies", "c", 0, "The number of times to copy the last 2 rows")
	getCmd.Flags().Int64P("start", "s", config.RegisterStartRow, "The last used row in the spreadsheet")
	getCmd.Flags().Int64P("end", "e", config.RegisterEndRow, "The last used row in the spreadsheet")
	getCmd.Flags().StringP("id", "i", config.SpreadsheetID, "The Google spreadsheet id")
}

func doThis() {
	fmt.Println("doThis()")
}

func getTrans(cmd *cobra.Command, args []string) {
	var err error

	banksOpt, _ := cmd.Flags().GetString("banks")
	banks := strings.Split(banksOpt, ",")

	ssID, _ := cmd.Flags().GetString("id")
	startRow, _ := cmd.Flags().GetInt64("star")
	endRow, _ := cmd.Flags().GetInt64("end")
	copies, _ := cmd.Flags().GetInt("copies")

	client := &banking.Client{
		Keys: &banking.Keys{
			Products:     "transactions",
			CountryCodes: "US",
		},
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

	fmt.Println("Getting transactions...")
	os.Exit(0)

	for _, cfg := range config.BankInfo {
		fmt.Printf("    %s...", cfg.Name)
		client.SetBank(cfg)
		transResp := client.GetTransactions(cfg, config.StartDate, config.EndDate)
		client.WriteCSV(cfg.FileName, transResp.Transactions)
		fmt.Println("done")
	}

	srv := &sheets.SheetService{
		Service:       sheets.NewService(),
		SpreadsheetID: ssID,
	}

	reg := sheets.NewRegisterSheet(srv, startRow, endRow)
	fmt.Printf("Reading Register...\n")
	reg.ID, err = srv.GetSheetID(config.TabNames["register"])
	checkError(err)
	reg.Read()

	if copies != 0 {
		fmt.Printf("Copying rows %d times...\n", copies)
		reg.CopyRows(copies)
		os.Exit(0)
	}

	rows := []*csv.Row{}
	csvRows := []*csv.Row{}
	for _, bank := range banks {
		bankFile := config.FinanceDir + config.BankInfo[bank].FileName
		fmt.Printf("Reading %s...\n", bankFile)
		switch bank {
		case "wellsfargo":
			rows = csv.ReadWellsFargoCSV(bankFile)
		case "fidelity":
			rows = csv.ReadFidelityCSV(bankFile)
		case "costcocitivisa":
			rows = csv.ReadCitiCSV(bankFile)
		case "chasevisa":
			rows = csv.ReadChaseCSV(bankFile)
		default:
			rows = []*csv.Row{}
			fmt.Printf("could not determine CSV file for %s\n", bank)
			os.Exit(0)
		}
		csvRows = append(csvRows, rows...)
	}
	reg.CSV = csvRows

	fmt.Printf("Sorting...\n")
	reg.SortByCSVDate()

	fmt.Printf("Reading Budget...\n")
	bud := sheets.NewBudgetSheet(srv, config.BudgetStartRow, config.BudgetEndRow)
	bud.ID, err = srv.GetSheetID(config.TabNames["budget"])
	checkError(err)
	bud.Read()
	reg.CategoriesMap = bud.CategoriesMap

	fmt.Printf("Filtering rows...\n")
	reg.CSV = reg.FilterCSVRows(reg.CSV)

	fmt.Printf("Transactions...\n")
	for i, r := range reg.CSV {
		fmt.Printf("    [%2d] %-5s %-10s %8.2f %s\n", i, r.Source, r.Date, r.Amount, r.Name)
	}

	fmt.Printf("Updating spreadheet...\n")
	reg.UpdateRows()
}

func checkError(err error) {
	if err != nil {
		panic(err)
	}
}
