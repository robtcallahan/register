package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/plaid/plaid-go/plaid"
)

var config *Config

func main() {
	var err error

	config = readConfig()
	options := parseOptions()
	banks := strings.Split(*options.Banks, ",")

	client := &Client{
		PlaidKeys: &PlaidKeys{
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
		checkErr(err)
		return client
	}()

	fmt.Println("Getting transactions...")
	for _, cfg := range config.BankInfo {
		fmt.Printf("    %s...", cfg.Name)
		client.setBank(cfg)
		transResp := client.getTransactions(cfg, config.StartDate, config.EndDate)
		writeCSV(cfg.FileName, transResp.Transactions)
		fmt.Println("done")
	}

	os.Exit(0)

	srv := &SheetService{
		Service:       newService(),
		SpreadsheetID: *options.SpreadsheetID,
	}

	reg := newRegisterSheet(srv, *options.RegisterStartRow, *options.RegisterEndRow)
	fmt.Printf("Reading Register...\n")
	reg.ID, err = srv.getSheetID(config.TabNames["register"])
	checkError(err)
	reg.read()

	if *options.NumberOfCopies != 0 {
		fmt.Printf("Copying rows %d times...\n", *options.NumberOfCopies)
		reg.copyRows(*options.NumberOfCopies)
		os.Exit(0)
	}

	rows := []*CSVRow{}
	csvRows := []*CSVRow{}
	for _, bank := range banks {
		bankFile := config.FinanceDir + config.BankInfo[bank].FileName
		fmt.Printf("Reading %s...\n", bankFile)
		switch bank {
		case "wellsfargo":
			rows = readWellsFargoCSV(bankFile)
		case "fidelity":
			rows = readFidelityCSV(bankFile)
		case "costcocitivisa":
			rows = readCitiCSV(bankFile)
		case "chasevisa":
			rows = readChaseCSV(bankFile)
		default:
			rows = []*CSVRow{}
			fmt.Printf("could not determine CSV file for %s\n", bank)
			os.Exit(0)
		}
		csvRows = append(csvRows, rows...)
	}
	reg.CSV = csvRows

	fmt.Printf("Sorting...\n")
	reg.sortByCSVDate()

	fmt.Printf("Reading Budget...\n")
	bud := newBudgetSheet(srv, config.BudgetStartRow, config.BudgetEndRow)
	bud.ID, err = srv.getSheetID(config.TabNames["budget"])
	checkError(err)
	bud.read()
	reg.CategoriesMap = bud.CategoriesMap

	fmt.Printf("Filtering rows...\n")
	reg.CSV = reg.filterCSVRows(reg.CSV)

	fmt.Printf("Transactions...\n")
	for i, r := range reg.CSV {
		fmt.Printf("    [%2d] %-5s %-10s %8.2f %s\n", i, r.Source, r.Date, r.Amount, r.Name)
	}

	fmt.Printf("Updating spreadheet...\n")
	reg.updateRows()
}
