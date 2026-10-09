package handler

import (
	"register/pkg/driver"
	"register/pkg/models"
	"register/pkg/repository"
	"register/pkg/repository/mysql"
)

// Query ...
type Query struct {
	repo repository.QueryRepo
}

// NewQueryHandler ...
func NewQueryHandler(db *driver.DB) *Query {
	var repo repository.QueryRepo

	switch db.DBType {
	case driver.MySQL:
		repo = mysql.NewMySQLQueryRepo(db.SQL)
	}
	return &Query{
		repo: repo,
	}
}

func (q *Query) GetTransactions() ([]models.Transaction, error) {
	return q.repo.GetTransactions()
}

func (q *Query) SaveTransaction(trans *models.Transaction) error {
	return q.repo.SaveTransaction(trans)
}

func (q *Query) UpdateTransactionTables(trans []*models.Transaction) error {
	return q.repo.UpdateTransactionTables(trans)
}

// GetColumns get all columns
func (q *Query) GetColumns() ([]models.Column, error) {
	return q.repo.GetColumns()
}

// AddColumn inserts a column at its index, shifting later columns up
func (q *Query) AddColumn(col *models.Column) error {
	return q.repo.AddColumn(col)
}

// RenameColumn changes a column's display name
func (q *Query) RenameColumn(id int, name string) error {
	return q.repo.RenameColumn(id, name)
}

// DeleteColumn removes a column, its merchant rules, and renumbers
func (q *Query) DeleteColumn(id int) error {
	return q.repo.DeleteColumn(id)
}

// GetMerchantsByColumn lists the merchant rules pointing at a column
func (q *Query) GetMerchantsByColumn(columnID int) ([]models.Merchant, error) {
	return q.repo.GetMerchantsByColumn(columnID)
}

// GetMerchants get all merchants
func (q *Query) GetMerchants() ([]models.Merchant, error) {
	return q.repo.GetMerchants()
}

// CreateMerchant ...
func (q *Query) CreateMerchant(m *models.Merchant) error {
	return q.repo.CreateMerchant(m)
}

// GetLookupData ...
func (q *Query) GetLookupData() ([]*models.DataRow, error) {
	return q.repo.GetLookupData()
}

// GetNameMapToColumn creates a map lookup from trans name to budget category/column names
func (q *Query) GetNameMapToColumn() (map[string]string, error) {
	return q.repo.GetNameMapToColumn()
}

// PrintData ...
func (q *Query) PrintData() error {
	return q.repo.PrintData()
}

// PrintTable ...
func (q *Query) PrintTable(table string) error {
	return q.repo.PrintTable(table)
}
