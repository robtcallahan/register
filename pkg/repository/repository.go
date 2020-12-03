package repository

import (
	"register/pkg/models"
)

// Repository represent the repositories
type Repository interface {
	GetColumns() []models.Column
	CreateMerchant(m *models.Merchant)
	GetLookupData() []*models.DataRow
	GetNameMapToColumn() map[string]string
	PrintData()
	PrintTable(table string)
}
