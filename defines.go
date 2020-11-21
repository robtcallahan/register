package main

import "google.golang.org/api/sheets/v4"

// SheetService ...
type SheetService struct {
	Service       *sheets.Service
	SpreadsheetID string
}

// BudgetEntry ...
type BudgetEntry struct {
	Category     string
	Weekly       float32
	Monthly      float32
	Every2Weeks  float32
	TwiceMonthly float32
	Yearly       float32
	// RegisterColumnName string
}

// BudgetSheet ...
type BudgetSheet struct {
	Service        *sheets.Service
	ID             int64
	SpreadsheetID  string
	StartRow       int64
	EndRow         int64
	LastRow        int64
	EndColumnName  string
	EndColumnIndex int64
	Spreadsheet    sheets.Spreadsheet
	BudgetEntries  []*BudgetEntry
	CategoriesMap  map[string]*BudgetEntry
}

// RegisterEntry ...
type RegisterEntry struct {
	Reconciled   string
	Source       string
	Date         string
	Description  string
	Amount       float32
	Withdrawl    string
	Deposit      string
	CreditCard   string
	BankRegister string
	Cleared      string
	Delta        string
}

// RegisterSheet ...
type RegisterSheet struct {
	Service          *sheets.Service
	ID               int64
	SpreadsheetID    string
	StartRow         int64
	EndRow           int64
	FirstRowToUpdate int64
	LastRow          int64
	EndColumnName    string
	EndColumnIndex   int64
	Spreadsheet      sheets.Spreadsheet
	RegisterEntries  []*RegisterEntry
	CSV              []*CSVRow
	CategoriesMap    map[string]*BudgetEntry
	ValuesMap        map[string][]interface{}
}

// Name ...
type Name struct {
	string
}

// FidelityVisa ...
type FidelityVisa struct {
	Date        string  `csv:"Date"`
	Transaction string  `csv:"Transaction"`
	Name        Name    `csv:"Name"`
	Memo        string  `csv:"Memo"`
	Amount      float32 `csv:"Amount"`
}

// CostcoCitiVisa ...
type CostcoCitiVisa struct {
	Status      string  `csv:"Status"`
	Date        string  `csv:"Date"`
	Description Name    `csv:"Description"`
	Debit       float32 `csv:"Debit"`
	Credit      float32 `csv:"Credit"`
	MemberName  string  `csv:"Member Name"`
}

// ChaseVisa ...
type ChaseVisa struct {
	TransactionDate string  `csv:"Transaction Date"`
	PostDate        string  `csv:"Post Date"`
	Description     Name    `csv:"Description"`
	Category        string  `csv:"Category"`
	Type            string  `csv:"Type"`
	Amount          float32 `csv:"Amount"`
}

// WellsFargo ...
type WellsFargo struct {
	Date   string  `csv:"Date"`
	Amount float32 `csv:"Amount"`
	Dummy1 string
	Dummy2 string
	Name   Name `csv:"Name"`
}

// CSVRow ...
type CSVRow struct {
	Key    string
	Source string
	Date   string
	Amount float32
	Name   string
}
