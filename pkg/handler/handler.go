package handler

import (
	"register/pkg/driver"
	"register/pkg/models"
	"register/pkg/repository"
	"register/pkg/repository/mysql"
)

// Query ...
type Query struct {
	repo repository.Repository
}

// NewQueryHandler ...
func NewQueryHandler(db *driver.DB) *Query {
	return &Query{
		repo: mysql.NewQueryRepo(db.SQL),
	}
}

// GetColumns get all columns
func (q *Query) GetColumns() []models.Column {
	return q.repo.GetColumns()
}

// CreateMerchant ...
func (q *Query) CreateMerchant(m *models.Merchant) {
	q.repo.CreateMerchant(m)
}

// GetLookupData ...
func (q *Query) GetLookupData() []*models.DataRow {
	return q.repo.GetLookupData()
}

// GetNameMapToColumn creates a map lookup from trans name to budget category/column names
func (q *Query) GetNameMapToColumn() map[string]string {
	return q.repo.GetNameMapToColumn()
}

// PrintData ...
func (q *Query) PrintData() {
	q.repo.PrintData()
}

// PrintTable ...
func (q *Query) PrintTable(table string) {
	q.repo.PrintTable(table)
}
