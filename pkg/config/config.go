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
	DBType                      string              `json:"db_type"`
	DBHost                      string              `json:"db_host"`
	DBPort                      string              `json:"db_port"`
	DBName                      string              `json:"db_name"`
	DBUsername                  string              `json:"db_username"`
	DBPassword                  string              `json:"db_password"`
	StartDate                   string              `json:"start_date"`
	EndDate                     string              `json:"end_date"`
	PlaidClientID               string              `json:"plaid_client_id"`
	PlaidSecret                 string              `json:"plaid_secret"`
	BankInfo                    map[string]BankInfo `json:"bank_info"`
	SpreadsheetID               string              `json:"spreadsheet_id"`
	SpreadsheetTestID           string              `json:"spreadsheet_test_id"`
	CreditCardColumnName        string              `json:"credit_card_column_name"`
	RegisterSheetID             string              `json:"register_sheet_id"`
	BudgetSheetID               string              `json:"budget_sheet_id"`
	FinanceDir                  string              `json:"finance_dir"`
	PaycheckName                string              `json:"paycheck_name"`
	BudgetStartRow              int64               `json:"budget_start_row"`
	BudgetEndRow                int64               `json:"budget_end_row"`
	RegisterStartRow            int64               `json:"register_start_row"`
	RegisterEndRow              int64               `json:"register_end_row"`
	TabNames                    map[string]string   `json:"tab_names"`
	RegisterCategoryStartColumn string              `json:"register_category_start_column"`
	RegisterCategoryEndColumn   string              `json:"register_category_end_column"`
	RegisterIndexes             map[string]int      `json:"register_indexes"`
	ColumnIndexes               map[string]int64    `json:"column_indexes"`
}

// ReadConfig ...
func ReadConfig() *Config {
	contents, err := ioutil.ReadFile("config/config.json")
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
