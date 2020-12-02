package repository

import (
	"gorm.io/gorm"
)

// Repository represent the repositories
type Repository interface {
	NewRepository(n NewRepositoryParams) *Repository
	CreateMerchant(m *Merchant)
	GetLookupData() []*DataRow
	GetColumns() []Column
	GetNameMapToColumn() map[string]string
	PrintData()
	PrintTable(table string)
}

// Repo ...
type Repo struct {
	repo Repository
}

// NewRepository ...
func (r *Repo) NewRepository(n NewRepositoryParams) *Repository

// CreateMerchant ...
func (r *Repo) CreateMerchant(m *Merchant)

// GetLookupData ...
func (r *Repo) GetLookupData() []*DataRow

// GetColumns ...
func (r *Repo) GetColumns() []Column

// GetNameMapToColumn ...
func (r *Repo) GetNameMapToColumn() map[string]string

// PrintData ...
func (r *Repo) PrintData()

// PrintTable ...
func (r *Repo) PrintTable(table string)

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

// NewRepositoryParams ...
type NewRepositoryParams struct {
	Debug      bool
	DBName     string
	DBUsername string
	DBPassword string
}
