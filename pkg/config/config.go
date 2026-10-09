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
	"fmt"
	"os"
)

// BudgetCategory ...
type BudgetCategory struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

type Bank struct {
	ID          string `json:"id"`
	Firm        string `json:"firm"`
	Name        string `json:"name"`
	Source      string `json:"source"`
	CSVFileName string `json:"csv_file_name"`
	Institution string `json:"institution"`
	AccountID   string `json:"account_id"`
	AccessToken string `json:"access_token"`
}

// DBConfig ...
type DBConfig struct {
	Type     string `json:"type"`
	Host     string `json:"host"`
	Port     string `json:"port"`
	Name     string `json:"name"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// Config ...
type Config struct {
	UserID                      string                   `json:"user_id"`
	JSONDir                     string                   `json:"json_dir"`
	DB                          DBConfig                 `json:"db"`
	StartDate                   string                   `json:"start_date"`
	EndDate                     string                   `json:"end_date"`
	PlaidClientID               string                   `json:"plaid_client_id"`
	PlaidEnvironment            string                   `json:"plaid_environment"`
	PlaidEnvSecrets             map[string]string        `json:"plaid_env_secrets"`
	PlaidTokensDir              string                   `json:"plaid_tokens_dir"`
	KeyFile                     string                   `json:"key_file"`
	CertFile                    string                   `json:"cert_file"`
	GoogleCredentialsFile       string                   `json:"google_credentials_file"`
	Banks                       map[string]Bank          `json:"banks"`
	SpreadsheetID               string                   `json:"spreadsheet_id"`
	SpreadsheetTestID           string                   `json:"spreadsheet_test_id"`
	CreditCardColumnName        string                   `json:"credit_card_column_name"`
	RegisterSheetID             string                   `json:"register_sheet_id"`
	BudgetSheetID               string                   `json:"budget_sheet_id"`
	FinanceDir                  string                   `json:"finance_dir"`
	BudgetStartRow              int64                    `json:"budget_start_row"`
	BudgetEndRow                int64                    `json:"budget_end_row"`
	RegisterStartRow            int64                    `json:"register_start_row"`
	RegisterEndRow              int64                    `json:"register_end_row"`
	MonthlyStartRow             int64                    `json:"monthly_start_row"`
	MonthlyEndRow               int64                    `json:"monthly_end_row"`
	TabNames                    map[string]string        `json:"tab_names"`
	RegisterCategoryStartColumn string                   `json:"register_category_start_column"`
	RegisterCategoryEndColumn   string                   `json:"register_category_end_column"`
	RegisterIndexes             map[string]int           `json:"register_indexes"`
	ColumnIndexes               map[string]int64         `json:"column_indexes"`
	IncomeSources               []IncomeSource           `json:"income_sources"`
}

// IncomeSource describes one configured source of income (a paycheck, a
// benefit deposit, ...): how to recognize it in the bank feed and what to
// call it in the register. Instances live in config.json; matching and the
// sheet write are wired up in 5.2/5.3.
type IncomeSource struct {
	Name      string `json:"name"`       // display name in the register
	Match     string `json:"match"`      // bank-name pattern to recognize the deposit
	MatchType string `json:"match_type"` // "exact" | "substring" | "regex" ("" = substring)
	Frequency string `json:"frequency"`  // "weekly" | "biweekly" | "monthly"
}

// ReadConfig ...
func ReadConfig(file string) (*Config, error) {
	contents, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("could not read config file: %s", err.Error())
	}

	var config = Config{}
	err = json.Unmarshal(contents, &config)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal config file: %s", err.Error())
	}
	return &config, nil
}
