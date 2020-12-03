package models

// Merchant ...
type Merchant struct {
	//gorm.Model
	ID       int
	BankName string
	Name     string
	ColumnID int
	Column   Column
}

// Column ...
type Column struct {
	//gorm.Model
	ID          int
	Name        string
	Color       string
	ColumnIndex int
	Letter      string
	IsCategory  bool
}

// DataRow ...
type DataRow struct {
	//gorm.Model
	ID          int
	Name        string
	BankName    string
	ColumnName  string
	ColumnIndex int
	Color       string
	IsCategory  bool
}
