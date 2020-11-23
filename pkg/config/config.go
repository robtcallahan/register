package config

import (
	"encoding/json"
	"io/ioutil"
)

// BudgetCategory ...
type BudgetCategory struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

// BankInfo ...
type BankInfo struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	FileName         string `json:"file_name"`
	PlaidItemID      string `json:"plaid_item_id"`
	PlaidAccessToken string `json:"plaid_access_token"`
	PlaidAccountID   string `json:"plaid_account_id"`
}

// Config ...
type Config struct {
	StartDate         string              `json:"start_date"`
	EndDate           string              `json:"end_date"`
	PlaidClientID     string              `json:"plaid_client_id"`
	PlaidSecret       string              `json:"plaid_secret"`
	BankInfo          map[string]BankInfo `json:"bank_info"`
	SpreadsheetID     string              `json:"spreadsheet_id"`
	SpreadsheetTestID string              `json:"spreadsheet_test_id"`
	FinanceDir        string              `json:"finance_dir"`
	PaycheckName      string              `json:"paycheck_name"`
	BudgetStartRow    int64               `json:"budget_start_row"`
	BudgetEndRow      int64               `json:"budget_end_row"`
	RegisterStartRow  int64               `json:"register_start_row"`
	RegisterEndRow    int64               `json:"register_end_row"`
	BudgetCategories  []BudgetCategory    `json:"budget_categories"`
	Merchants         map[string]string   `json:"merchants"`
	TabNames          map[string]string   `json:"tab_names"`
	RegisterIndexes   map[string]int      `json:"register_indexes"`
	ColumnIndexes     map[string]int64    `json:"column_indexes"`
}

// ReadConfig ...
func ReadConfig() *Config {
	contents, err := ioutil.ReadFile("config.json")
	checkError(err)

	var config Config = Config{}
	err = json.Unmarshal([]byte(contents), &config)
	checkError(err)
	return &config
}

func checkError(err error) {
	if err != nil {
		panic(err)
	}
}
