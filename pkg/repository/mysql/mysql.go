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

func (r *mysqlQueryRepo) SaveTransaction(trans *models.Transaction) error {
	if result := r.Conn.Save(trans); result.Error != nil {
		return fmt.Errorf("unable to save transaction: %w", result.Error)
	}
	return nil
}

// UpdateTransactionTables ...
func (r *mysqlQueryRepo) UpdateTransactionTables(trans []*models.Transaction) error {
	if err := r.Conn.AutoMigrate(&models.Transaction{}); err != nil {
		return fmt.Errorf("unable to migrate transactions table: %w", err)
	}

	for _, t := range trans {
		result := r.Conn.Clauses(clause.OnConflict{
			UpdateAll: true,
		}).Create(t)
		if result.Error != nil {
			return fmt.Errorf("unable to create transaction: %w", result.Error)
		}
	}
	return nil
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
func (r *mysqlQueryRepo) CreateMerchant(m *models.Merchant) error {
	result := r.Conn.Create(&models.Merchant{
		Name:     m.Name,
		BankName: m.BankName,
		ColumnID: m.ColumnID,
	})
	if result.Error != nil {
		return fmt.Errorf("unable to create merchant: %w", result.Error)
	}
	return nil
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
			ID:            m.ID,
			Name:          m.Name,
			BankName:      m.BankName,
			ColumnName:    m.Column.Name,
			ColumnIndex:   m.Column.ColumnIndex,
			Color:         m.Column.Color,
			IsCategory:    m.Column.IsCategory(),
			TaxDeductible: m.TaxDeductible,
			Priority:      m.Priority,
			MatchType:     m.MatchType,
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

// AddColumn inserts col at its ColumnIndex, shifting the columns at or
// after that index up by one. One transaction: column_index is UNIQUE, and
// descending per-row updates keep every intermediate state collision-free.
func (r *mysqlQueryRepo) AddColumn(col *models.Column) error {
	return r.Conn.Transaction(func(tx *gorm.DB) error {
		var shifting []models.Column
		if err := tx.Where("column_index >= ?", col.ColumnIndex).Order("column_index desc").Find(&shifting).Error; err != nil {
			return fmt.Errorf("unable to shift columns: %w", err)
		}
		for i := range shifting {
			if err := tx.Model(&models.Column{}).Where("id = ?", shifting[i].ID).
				Update("column_index", shifting[i].ColumnIndex+1).Error; err != nil {
				return fmt.Errorf("unable to shift column %s: %w", shifting[i].Name, err)
			}
		}
		// is_category is still NOT NULL in the DDL (until Rob drops the
		// column), so the insert writes the value derived from position.
		row := map[string]interface{}{
			"name":         col.Name,
			"color":        col.Color,
			"column_index": col.ColumnIndex,
			"is_category":  col.IsCategory(),
		}
		if err := tx.Table("columns").Create(row).Error; err != nil {
			return fmt.Errorf("unable to create column %s: %w", col.Name, err)
		}
		return nil
	})
}

// RenameColumn ...
func (r *mysqlQueryRepo) RenameColumn(id int, name string) error {
	if result := r.Conn.Model(&models.Column{}).Where("id = ?", id).Update("name", name); result.Error != nil {
		return fmt.Errorf("unable to rename column: %w", result.Error)
	}
	return nil
}

// DeleteColumn removes a column and the merchant rules pointing at it, then
// shifts later columns down by one. Hard deletes throughout: a soft-deleted
// row would keep occupying its column_index under the UNIQUE constraint.
func (r *mysqlQueryRepo) DeleteColumn(id int) error {
	return r.Conn.Transaction(func(tx *gorm.DB) error {
		var col models.Column
		if err := tx.First(&col, id).Error; err != nil {
			return fmt.Errorf("unable to find column %d: %w", id, err)
		}
		if err := tx.Unscoped().Where("column_id = ?", id).Delete(&models.Merchant{}).Error; err != nil {
			return fmt.Errorf("unable to delete merchants for column %s: %w", col.Name, err)
		}
		if err := tx.Unscoped().Delete(&models.Column{}, id).Error; err != nil {
			return fmt.Errorf("unable to delete column %s: %w", col.Name, err)
		}
		var shifting []models.Column
		if err := tx.Where("column_index > ?", col.ColumnIndex).Order("column_index asc").Find(&shifting).Error; err != nil {
			return fmt.Errorf("unable to shift columns: %w", err)
		}
		for i := range shifting {
			if err := tx.Model(&models.Column{}).Where("id = ?", shifting[i].ID).
				Update("column_index", shifting[i].ColumnIndex-1).Error; err != nil {
				return fmt.Errorf("unable to shift column %s: %w", shifting[i].Name, err)
			}
		}
		return nil
	})
}

// GetMerchantsByColumn ...
func (r *mysqlQueryRepo) GetMerchantsByColumn(columnID int) ([]models.Merchant, error) {
	var merchants []models.Merchant
	if result := r.Conn.Where("column_id = ?", columnID).Order("name").Find(&merchants); result.Error != nil {
		return nil, fmt.Errorf("unable to get merchants for column: %w", result.Error)
	}
	return merchants, nil
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
