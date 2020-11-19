package main

import (
	"encoding/json"
	"io/ioutil"
)

// BudgetCategory ...
type BudgetCategory struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

// Config ...
type Config struct {
	SpreadsheetID     string            `json:"spreadsheet_id"`
	SpreadsheetTestID string            `json:"spreadsheet_test_id"`
	FinanceDir        string            `json:"finance_dir"`
	BankFiles         map[string]string `json:"bank_files"`
	PaycheckName      string            `json:"paycheck_name"`
	BudgetStartRow    int64             `json:"budget_start_row"`
	BudgetEndRow      int64             `json:"budget_end_row"`
	RegisterStartRow  int64             `json:"register_start_row"`
	RegisterEndRow    int64             `json:"register_end_row"`
	BudgetCategories  []BudgetCategory  `json:"budget_categories"`
	Merchants         map[string]string `json:"merchants"`
	TabNames          map[string]string `json:"tab_names"`
	RegisterIndexes   map[string]int    `json:"register_indexes"`
	ColumnIndexes     map[string]int64  `json:"column_indexes"`
}

func readConfig() *Config {
	contents, err := ioutil.ReadFile("/Users/rob/workspace/go/src/budget/config.json")
	checkError(err)

	var config Config = Config{}
	err = json.Unmarshal([]byte(contents), &config)
	checkError(err)
	return &config
}

// "budget_categories": {
// 	"Cash": "K",
// 	"Dining Out": "L",
// 	"Gas": "M",
// 	"Grocery": "N",
// 	"Misc": "O",
// 	"Vape Supplies": "P",
// 	"AT&T Cell Phone": "Q",
// 	"Content Subscriptions": "R",
// 	"Comcast/Xfinity Internet": "S",
// 	"Washington Gas": "U",
// 	"Dominion Power": "V",
// 	"Hair Cut": "W",
// 	"Insurance, Auto": "Y",
// 	"Massage": "AA",
// 	"Loudoun Heights Rent": "AB",
// 	"Insurance, Renters": "AC",
// 	"Storage Rental": "AD",
// 	"Personal Loan": "AG",
// 	"Car Loan": "AH",
// 	"IRS": "AI",
// 	"Car Expenses": "AL",
// 	"Car Property Tax": "AM",
// 	"Clothing & Household": "AN",
// 	"Extra": "AO",
// 	"Gifts": "AP",
// 	"Gandalf": "AT",
// 	"Mental Health": "AU",
// 	"Medical (SoberLink)": "AV",
// 	"Vision": "AW",
// 	"Emergency": "AY",
// 	"Exercise Equipment": "BA",
// 	"General Savings (court fines)": "BB"
// },
