package models

import (
	"gorm.io/gorm"
)

// Transaction ...
type Transaction struct {
	gorm.Model
	Key            string
	Source         string
	Date           string
	Name           string
	BankName       string
	Note           string
	Amount         float64 // The actual transaction value
	Withdrawal     float64 // the amount that goes in the Withdrawal column (positive)
	Deposit        float64 // the amount that goes in the Deposit column (positive)
	CreditPurchase float64 // the amount that goes in the Credit Purchases column (positive)
	Budget         float64 // the amount that goes in the Budget category column (negative)
	CreditCard     float64 // the amount that goes in the Credit Card column (positive)
	ColumnIndex    int
	Color          string
	IsCategory     bool
	TaxDeductible  bool
	IsCheck        bool
}

// Merchant ...
type Merchant struct {
	gorm.Model
	ID            int
	BankName      string
	Name          string
	ColumnID      int
	Column        Column
	TaxDeductible bool
}

// Column ...
type Column struct {
	gorm.Model
	ID          int
	Name        string
	Color       string
	ColumnIndex int
	Letter      string
	IsCategory  bool
}

// DataRow ...
type DataRow struct {
	ID            int
	Name          string
	BankName      string
	ColumnName    string
	ColumnIndex   int
	Color         string
	IsCategory    bool
	TaxDeductible bool
}
