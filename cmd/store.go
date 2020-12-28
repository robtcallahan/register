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
	"github.com/dustin/go-humanize"
	"github.com/plaid/plaid-go/plaid"
	"register/pkg/banking"
	cfg "register/pkg/config"

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

type Investments struct {
	Firm        string
	AccountName string
	Balance     float64
}

func store() {
	bankClient := banking.New(&banking.ClientOptions{
		StartDate:     config.StartDate,
		EndDate:       config.EndDate,
		BankInfo:      config.BankInfo,
		Debug:         options.Debug,
		PlaidClientID: config.PlaidClientID,
		PlaidSecret:   config.PlaidSecret,
	})

	//bankClient.SetBank(config.BankInfo["sofi"])
	//resp := bankClient.GetAccounts()
	//for _, a := range resp.Accounts {
	//	fmt.Printf("%s, %s, %.2f\n", a.Name, a.Type, a.Balances.Current)
	//}

	fmt.Println("Getting investment balances...")
	investments := getInvestmentTotals(bankClient)
	fmt.Printf("%d accounts found\n\n", len(investments))

	var sum float64
	for _, i := range investments {
		sum += i.Balance
		fmt.Printf("%-12s %-18s $ %10s\n", i.Firm, i.AccountName, humanize.FormatFloat("#,###.##", i.Balance))
	}
	fmt.Printf("%31s $ %10s\n", " ", humanize.FormatFloat("#,###.##", sum))

	//acc := bankClient.GetAccount(config.BankInfo["wellsfargo"], "depository")
	//fmt.Printf("current: %.2f, avail: %.2f\n", acc.Balances.Current, acc.Balances.Available)

	//sheetsService, err := sheets_service.New(sheets_provider.New(options.SpreadsheetID))
	//checkError(err)

	//fmt.Printf("Reading Register...\n")
	//err = sheetsService.NewRegisterSheet(config)
	//checkError(err)

	//fmt.Printf("Reading Budget...\n")
	//err = sheetsService.NewBudgetSheet(config)
	//checkError(err)

	//_, err = sheetsService.ReadBudgetSheet()
	//checkError(err)

	//dir := "/Users/rob/ws/go/src/register/api/services/sheets_service/json/"
	//j, err := json.Marshal(sheetsService.RegisterSheet)
	//checkError(err)
	//err = ioutil.WriteFile(dir + "ReadRegisterSheet.json", j, 0644)
	//checkError(err)

	//val := sheetsService.ReadCell("A1", "string")
	//fmt.Printf("val: %s\n", val)

	//conn, err := driver.ConnectSQL(&driver.ConnectParams{
	//	DBType: driver.DBType(config.DBType),
	//	Host:   config.DBHost,
	//	Port:   config.DBPort,
	//	DBName: config.DBName,
	//	User:   config.DBUsername,
	//	Pass:   config.DBPassword,
	//})
	//checkError(err)
	//qHandler := handler.NewQueryHandler(conn)

}

func getInvestmentTotals(bankClient *banking.Client) []*Investments {
	var (
		invest []*Investments
		resp plaid.GetAccountsResponse
	)

	fmt.Println("    Fidelity")
	bankClient.SetBank(config.BankInfo["fidelity"])
	resp = bankClient.GetAccounts()
	for _, a := range resp.Accounts {
		if a.Name == "CROWDSTRIKE, INC." {
			invest = append(invest, &Investments{
				Firm:        "Fidelity",
				AccountName: "CrowdStrike 401k",
				Balance:     a.Balances.Current,
			})
			break
		}
	}

	fmt.Println("    E*Trade")
	bankClient.SetBank(config.BankInfo["etrade"])
	resp = bankClient.GetAccounts()
	for _, a := range resp.Accounts {
		if a.Name == "Stock Plan (CRWD)" {
			invest = append(invest, &Investments{
				Firm:        "E*Trade",
				AccountName: "CrowdStrike Stock",
				Balance:     a.Balances.Current,
			})
			break
		}
	}

	fmt.Println("    Betterment")
	bankClient.SetBank(config.BankInfo["betterment"])
	resp = bankClient.GetAccounts()
	a := resp.Accounts[0]
	invest = append(invest, &Investments{
		Firm:        "Betterment",
		AccountName: "Personal IRA",
		Balance:     a.Balances.Current,
	})

	fmt.Println("    Ally")
	bankClient.SetBank(config.BankInfo["ally"])
	resp = bankClient.GetAccounts()
	for _, a := range resp.Accounts {
		if a.Name == "Online Savings" {
			invest = append(invest, &Investments{
				Firm:        "Ally Bank",
				AccountName: "Online Savings",
				Balance:     a.Balances.Current,
			})
			break
		}
	}

	fmt.Println("    Sofi Loans")
	bankClient.SetBank(config.BankInfo["sofi"])
	resp = bankClient.GetAccounts()
	for _, a := range resp.Accounts {
		if a.Name == "SoFi Personal Loan" {
			invest = append(invest, &Investments{
				Firm:        "SoFi Loans",
				AccountName: "Personal Loan",
				Balance:     -a.Balances.Current,
			})
			break
		}
	}

	return invest
}

//func fixTransactionTable() {
//conn, err := driver.ConnectSQL(&driver.ConnectParams{
//	DBType: driver.DBType(config.DBType),
//	Host:   config.DBHost,
//	Port:   config.DBPort,
//	DBName: config.DBName,
//	User:   config.DBUsername,
//	Pass:   config.DBPassword,
//})
//if err != nil {
//	panic(err)
//}
//qHandler := handler.NewQueryHandler(conn)

//trans := qHandler.GetTransactions()
//for _, t := range trans {
//	tri := strings.Split(t.Key, ":")
//	if tri[0] == "-" {
//		tri[0] = "WellsFargo"
//	} else {
//		tri[0] = strings.Title(strings.ToLower(tri[0]))
//	}
//	t.Key = strings.Join(tri, ":")
//
//	if t.Source == "-" {
//		t.Source = "WellsFargo"
//	} else {
//		t.Source = strings.Title(strings.ToLower(t.Source))
//	}
//
//	qHandler.SaveTransaction(&t)
//	fmt.Println(t.Key)
//}
//}

//func printTables(h *handler.Query) {
//	fmt.Println("Columns...")
//	cols := h.GetColumns()
//	printColumns(cols, 20)
//	fmt.Println("")
//
//	fmt.Println("Merchants...")
//	merch := h.GetMerchants()
//	printMerchants(merch, 20)
//	fmt.Println("")
//}

//func printMerchants(merch []models.Merchant, num int) {
//	for i := 0; i < num; i++ {
//		m := merch[i]
//		fmt.Printf("    %-35s %s\n", m.CreatedAt, m.BankName)
//	}
//}

//func printColumns(cols []models.Column, num int) {
//	for i := 0; i < num; i++ {
//		c := cols[i]
//		fmt.Printf("    %-35s %s\n", c.CreatedAt, c.Name)
//	}
//}
