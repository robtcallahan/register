package repository

import (
	"register/pkg/models"

	"gorm.io/gorm"
)

// QueryRepo represent the repositories
type QueryRepo interface {
	CreateDB(dbName string) (*gorm.DB, error)
	UpdateTransactionTables(trans []*models.Transaction)
	GetColumns() []models.Column
	GetMerchants() []models.Merchant
	CreateMerchant(m *models.Merchant)
	GetLookupData() []*models.DataRow
	GetNameMapToColumn() map[string]string
	PrintData()
	PrintTable(table string)
}
