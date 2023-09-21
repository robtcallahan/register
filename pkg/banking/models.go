package banking

import "gorm.io/gorm"

// Transaction ...
type Transaction struct {
	gorm.Model
	Key           string
	Source        string
	Date          string
	Name          string
	BankName      string
	Note          string
	Amount        float64
	Withdrawal    float64
	Deposit       float64
	CreditCard    float64
	ColumnIndex   int
	Color         string
	IsCategory    bool
	TaxDeductible bool
	IsCheck       bool
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
