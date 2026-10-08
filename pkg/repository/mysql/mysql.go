package mysql

import (
	"fmt"
	"gorm.io/gorm/clause"
	"register/pkg/models"
	repo "register/pkg/repository"

	"gorm.io/gorm"
)

type mysqlQueryRepo struct {
	Conn *gorm.DB
}

// NewMySQLQueryRepo ...
func NewMySQLQueryRepo(conn *gorm.DB) repo.QueryRepo {
	return &mysqlQueryRepo{
		Conn: conn,
	}
}

func (r *mysqlQueryRepo) GetTransactions() ([]models.Transaction, error) {
	var trans []models.Transaction
	if result := r.Conn.Order("date").Find(&trans); result.Error != nil {
		return nil, fmt.Errorf("unable to get transactions: %w", result.Error)
	}
	return trans, nil
}

func (r *mysqlQueryRepo) SaveTransaction(trans *models.Transaction) {
	r.Conn.Save(trans)
}

// UpdateTransactionTables ...
func (r *mysqlQueryRepo) UpdateTransactionTables(trans []*models.Transaction) {
	_ = r.Conn.AutoMigrate(&models.Transaction{})

	for _, t := range trans {
		result := r.Conn.Clauses(clause.OnConflict{
			UpdateAll: true,
		}).Create(t)
		if result.Error != nil {
			panic(result.Error)
		}
	}
}

// CreateDB ...
func (r *mysqlQueryRepo) CreateDB(dbName string) (*gorm.DB, error) {
	db := r.Conn.Exec("CREATE DATABASE " + dbName)
	return db, db.Error
}

// GetColumns ...
func (r *mysqlQueryRepo) GetColumns() ([]models.Column, error) {
	var cols []models.Column
	if result := r.Conn.Order("column_index").Find(&cols); result.Error != nil {
		return nil, fmt.Errorf("unable to get columns: %w", result.Error)
	}
	return cols, nil
}

// GetMerchants ...
func (r *mysqlQueryRepo) GetMerchants() ([]models.Merchant, error) {
	var merch []models.Merchant
	if result := r.Conn.Order("name").Find(&merch); result.Error != nil {
		return nil, fmt.Errorf("unable to get merchants: %w", result.Error)
	}
	return merch, nil
}

// CreateMerchant ...
func (r *mysqlQueryRepo) CreateMerchant(m *models.Merchant) {
	result := r.Conn.Create(&models.Merchant{
		Name:     m.Name,
		BankName: m.BankName,
		ColumnID: m.ColumnID,
	})
	if result.Error != nil {
		panic(result.Error)
	}
}

// GetLookupData ...
func (r *mysqlQueryRepo) GetLookupData() ([]*models.DataRow, error) {
	var merchants []models.Merchant

	if result := r.Conn.Preload("Column").Find(&merchants); result.Error != nil {
		return nil, fmt.Errorf("unable to get lookup data: %w", result.Error)
	}

	var data []*models.DataRow
	for _, m := range merchants {
		data = append(data, &models.DataRow{
			Name:          m.Name,
			BankName:      m.BankName,
			ColumnName:    m.Column.Name,
			ColumnIndex:   m.Column.ColumnIndex,
			Color:         m.Column.Color,
			IsCategory:    m.Column.IsCategory,
			TaxDeductible: m.TaxDeductible,
		})
	}
	return data, nil
}

// GetNameMapToColumn creates a map lookup from trans name to budget category/column names
func (r *mysqlQueryRepo) GetNameMapToColumn() (map[string]string, error) {
	cols, err := r.GetLookupData()
	if err != nil {
		return nil, err
	}

	transNameToColName := make(map[string]string)
	for _, c := range cols {
		transNameToColName[c.Name] = c.ColumnName
	}
	return transNameToColName, nil
}

// PrintData ...
func (r *mysqlQueryRepo) PrintData() error {
	var merchants []models.Merchant
	if result := r.Conn.Preload("Column").Find(&merchants); result.Error != nil {
		return fmt.Errorf("unable to get merchants: %w", result.Error)
	}

	fmt.Printf("[Num] %-35s %-30s %-30s %-s\n", "Bank Name", "Name", "Column Name", "Column Index")
	for i, m := range merchants {
		fmt.Printf("[%3d] %-35s %-30s %-30s %2d\n", i+1, m.BankName, m.Name, m.Column.Name, m.Column.ColumnIndex)
	}
	return nil
}

// PrintTable ...
func (r *mysqlQueryRepo) PrintTable(table string) error {
	switch table {
	case "merchants":
		var merchants []models.Merchant
		result := r.Conn.Find(&merchants)
		if result.Error != nil {
			return fmt.Errorf("unable to get merchants: %w", result.Error)
		}
		fmt.Printf("%d rows found\n", result.RowsAffected)
		for _, m := range merchants {
			fmt.Printf("%d %s %s %s\n", m.ID, m.BankName, m.Name, m.Column.Name)
		}
	}
	return nil
}
