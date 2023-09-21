package handler

import (
	"register/pkg/driver"
	"register/pkg/models"
	"register/pkg/repository"
	"register/pkg/repository/mysql"
	"register/pkg/repository/postgres"
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
	case driver.PostgreSQL:
		repo = postgres.NewPostgreSQLQueryRepo(db.SQL)
	}
	return &Query{
		repo: repo,
	}
}

func (q *Query) GetTransactions() []models.Transaction {
	return q.repo.GetTransactions()
}

func (q *Query) SaveTransaction(trans *models.Transaction) {
	q.repo.SaveTransaction(trans)
}

func (q *Query) UpdateTransactionTables(trans []*models.Transaction) {
	q.repo.UpdateTransactionTables(trans)
}

// GetColumns get all columns
func (q *Query) GetColumns() []models.Column {
	return q.repo.GetColumns()
}

// GetMerchants get all merchants
func (q *Query) GetMerchants() []models.Merchant {
	return q.repo.GetMerchants()
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
