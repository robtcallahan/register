package models

import (
	"time"

	"gorm.io/gorm"
)

// Transaction ...
type Transaction struct {
	Key            string `gorm:"size:255;uniqueIndex"`
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
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      gorm.DeletedAt
}

// Merchant ...
type Merchant struct {
	ID            int
	BankName      string
	Name          string
	ColumnID      int
	Column        Column
	TaxDeductible bool
	Priority      int    // when several rules match, the highest priority wins
	MatchType     string // "exact", "substring", or "regex"; "" is treated as "substring" (pre-Phase-4 rows)
	Pattern       string `gorm:"column:match_pattern"` // what the matcher compares; BankName stays the raw observed name
	CreatedAt     time.Time
	UpdatedAt     time.Time
	DeletedAt     gorm.DeletedAt
}

// Column ...
type Column struct {
	ID          int
	Name        string
	Color       string
	ColumnIndex int
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt
}

// FirstCategoryIndex is the first 1-based Register position that holds a
// budget category; positions 1-10 (A-J) are the fixed register block.
const FirstCategoryIndex = 11

// Column colors: stored on each columns row, and the names of the cell
// background palette. Grey is reserved for the payoff columns (Credit
// Cards, AppleCard) and is never offered for a new category.
const (
	CheckTransactionName = "CHECK"
	CheckRegisterName    = "Check"
	IncomeColumnName     = "Emergency Fund" // column_index 42 in his tables today

	WellsFargoID        = "wellsfargo"
	FidelityID          = "fidelity"
	ChaseID             = "chase"
	BoAID               = "boa"
	CheckingBalanceCell = "G1" // Wells Fargo ending balance
	CheckingDeltaCell   = "G2" // running check: balance minus cleared total

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
	Pattern       string
}

// IsCategory reports whether the column sits in the category block. It
// replaces the old is_category database flag: position alone decides, so
// the two can never disagree.
func (c Column) IsCategory() bool {
	return c.ColumnIndex >= FirstCategoryIndex
}
