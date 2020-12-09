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
	"log"
	cfg "register/pkg/config"
	"register/pkg/driver"
	"register/pkg/handler"
	"register/pkg/models"
	"strings"

	"github.com/spf13/cobra"
)

// storeCmd represents the store command
var storeCmd = &cobra.Command{
	Use:   "store",
	Short: "Just a placeholder. Doesn't do anything yet.",
	Long:  `Just a placeholder. Doesn't do anything yet.`,
	Run: func(cmd *cobra.Command, args []string) {
		store()
	},
}

func init() {
	config = cfg.ReadConfig()
	rootCmd.AddCommand(storeCmd)

	storeCmd.Flags().StringVarP(&options.SpreadsheetID, "id", "i", config.SpreadsheetID, "The Google spreadsheet id")
	storeCmd.Flags().Int64VarP(&options.StartRow, "start", "s", config.RegisterStartRow, "The last used row in the spreadsheet")
	storeCmd.Flags().Int64VarP(&options.EndRow, "end", "e", config.RegisterEndRow, "The last used row in the spreadsheet")

	storeCmd.Flags().BoolVarP(&options.Test, "test", "t", false, "Test mode; no updates performed")
	storeCmd.Flags().BoolVarP(&options.Debug, "debug", "d", false, "Debug mode")
}

func store() {




	//bankClient := banking.New(&banking.ClientOptions{
	//	StartDate:     config.StartDate,
	//	EndDate:       config.EndDate,
	//	BankInfo:      config.BankInfo,
	//	Debug:         options.Debug,
	//	PlaidClientID: config.PlaidClientID,
	//	PlaidSecret:   config.PlaidSecret,
	//})
	//
	//acc := bankClient.GetAccount(config.BankInfo["wellsfargo"], "depository")
	//fmt.Printf("current: %.2f, avail: %.2f\n", acc.Balances.Current, acc.Balances.Available)

	//sheetsService, err := sheets_service.New(options.SpreadsheetID, options.Verbose)
	//checkError(err)

	//fmt.Printf("Reading Register...\n")
	//regSrv := sheets.NewRegisterSheet(sheetService, config, options, config.RegisterStartRow, config.RegisterEndRow)
	//regSrv.ID, err = sheetService.GetSheetID(config.TabNames["register"])
	//checkError(err)
	//regSrv.Register, regSrv.KeysMap, _ = regSrv.Read()


	//curBal := regSrv.ReadDollarsCell("G1")
	//fmt.Printf("Current Balance: %.2f\n", curBal)

	// Mon Jan 2 15:04:05 -0700 MST 2006
	//curDate := time.Now().Format("01/02/2006")
	//curDate := regSrv.ReadDateCell("F1")
	//fmt.Printf("Current Date: %s\n", curDate)

	//reconFormula := "=SUM(G1-I6406)"
	//reconValue := regSrv.ReadDollarsCell("G2")
	//fmt.Printf("Recon Value: %.2f\n", reconValue)

	//regSrv.WriteCell("G2", "=SUM(G1-I6468)")
	//regSrv.WriteCell("F1", time.Now().Format("01/02/2006"))
	//regSrv.WriteCell("G1", acc.Balances.Available)
}

func checkErr(err error) {
	if err != nil {
		log.Fatalf("unable to retrieve data from sheet: %v", err)
	}
}

func fixTransactionTable() {
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

	trans := qHandler.GetTransactions()
	for _, t := range trans {
		tri := strings.Split(t.Key, ":")
		if tri[0] == "-" {
			tri[0] = "WellsFargo"
		} else {
			tri[0] = strings.Title(strings.ToLower(tri[0]))
		}
		t.Key = strings.Join(tri, ":")

		if t.Source == "-" {
			t.Source = "WellsFargo"
		} else {
			t.Source = strings.Title(strings.ToLower(t.Source))
		}

		qHandler.SaveTransaction(&t)
		fmt.Println(t.Key)
	}
}

func printTables(h *handler.Query) {
	fmt.Println("Columns...")
	cols := h.GetColumns()
	printColumns(cols, 20)
	fmt.Println("")

	fmt.Println("Merchants...")
	merch := h.GetMerchants()
	printMerchants(merch, 20)
	fmt.Println("")
}

func printMerchants(merch []models.Merchant, num int) {
	for i := 0; i < num; i++ {
		m := merch[i]
		fmt.Printf("    %-35s %s\n", m.CreatedAt, m.BankName)
	}
}

func printColumns(cols []models.Column, num int) {
	for i := 0; i < num; i++ {
		c := cols[i]
		fmt.Printf("    %-35s %s\n", c.CreatedAt, c.Name)
	}
}
