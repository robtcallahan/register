package models

import (
	"gorm.io/gorm"
)

// Merchant ...
type Merchant struct {
	gorm.Model
	BankName string
	Name     string
	ColumnID int
	Column   Column
}

// Column ...
type Column struct {
	ID          int
	Name        string
	Color       string
	ColumnIndex int
	Letter      string
	IsCategory  bool
}

// DataRow ...
type DataRow struct {
	Name        string
	BankName    string
	ColumnName  string
	ColumnIndex int
	Color       string
	IsCategory  bool
}
