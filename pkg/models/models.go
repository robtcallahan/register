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
	Priority      int    // when several rules match, the highest priority wins
	MatchType     string // "exact", "substring", or "regex"; "" is treated as "substring" (pre-Phase-4 rows)
}

// Column ...
type Column struct {
	gorm.Model
	ID          int
	Name        string
	Color       string
	ColumnIndex int
}

// FirstCategoryIndex is the first 1-based Register position that holds a
// budget category; positions 1-10 (A-J) are the fixed register block.
const FirstCategoryIndex = 11

// IsCategory reports whether the column sits in the category block. It
// replaces the old is_category database flag: position alone decides, so
// the two can never disagree.
func (c Column) IsCategory() bool {
	return c.ColumnIndex >= FirstCategoryIndex
}

// Column colors: stored on each columns row, and the names of the cell
// background palette. Black is reserved for the payoff columns (Credit
// Cards, AppleCard, IRS) and is never offered for a new category.
const (
	ColorBlack     = "black"
	ColorWhite     = "white"
	ColorGreen     = "green"
	ColorYellow    = "yellow"
	ColorBlue      = "blue"
	ColorGrey      = "grey"
	ColorLightGrey = "lightgrey"
)

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
	Priority      int
	MatchType     string
}
