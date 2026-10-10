package repository

import (
	"register/pkg/models"

	"gorm.io/gorm"
)

// QueryRepo represent the repositories
type QueryRepo interface {
	CreateDB(dbName string) (*gorm.DB, error)

	GetColumns() ([]models.Column, error)
	AddColumn(col *models.Column) error
	RenameColumn(id int, name string) error
	DeleteColumn(id int) error
	MoveColumn(id int, newIndex int) error
	GetMerchantsByColumn(columnID int) ([]models.Merchant, error)

	GetMerchants() ([]models.Merchant, error)
	CreateMerchant(m *models.Merchant) error

	GetTransactions() ([]models.Transaction, error)
	SaveTransaction(trans *models.Transaction) error
	UpdateTransactionTables(trans []*models.Transaction) error

	GetLookupData() ([]*models.DataRow, error)
	GetNameMapToColumn() (map[string]string, error)

	PrintData() error
	PrintTable(table string) error
}

func ColumnNames(cats []models.Column) *[]string {
	names := make([]string, 0, len(cats))
	for _, c := range cats {
		names = append(names, c.Name)
	}
	return &names
}
