package mysql

import (
	"fmt"
	"register/pkg/models"
	repo "register/pkg/repository"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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

func (r *mysqlQueryRepo) UpdateTransactionTables(transactions []*models.Transaction) error {
	return r.Conn.Transaction(func(tx *gorm.DB) error {
		for _, t := range transactions {
			if result := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(t); result.Error != nil {
				return result.Error
			}
		}
		return nil
	})
}

// GetColumns ...
func (r *mysqlQueryRepo) GetColumns() ([]models.Column, error) {
	var cols []models.Column
	if result := r.Conn.Order("column_index").Find(&cols); result.Error != nil {
		return nil, fmt.Errorf("unable to get columns: %w", result.Error)
	}
	return cols, nil
}

// CreateMerchant ...
func (r *mysqlQueryRepo) CreateMerchant(m *models.Merchant) error {
	if m.Pattern == "" {
		m.Pattern = m.BankName
	}
	result := r.Conn.Create(m)
	if result.Error != nil {
		return fmt.Errorf("unable to create merchant: %w", result.Error)
	}
	return nil
}

// GetMaxMerchantPriority returns the highest rule priority in the merchants
// table, or 0 when the table is empty.
func (r *mysqlQueryRepo) GetMaxMerchantPriority() (int, error) {
	var max int
	if result := r.Conn.Model(&models.Merchant{}).Select("COALESCE(MAX(priority), 0)").Scan(&max); result.Error != nil {
		return 0, fmt.Errorf("unable to get max merchant priority: %w", result.Error)
	}
	return max, nil
}

// GetLookupData queries the columns and merchants tables using foreign keys and creates
// and array of models.DataRow that contains an aggregate of both tables.
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
			Pattern:       m.Pattern,
		})
	}
	return data, nil
}

// GetNameMapToColumn creates a map lookup from trans name to budget category/column names
func (r *mysqlQueryRepo) GetNameMapToColumn() (map[string]string, error) {
	cols, err := r.GetLookupData()
	if err != nil {
		return nil, fmt.Errorf("get lookup data for name-to-column map: %w", err)
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
		row := map[string]any{
			"name":         col.Name,
			"color":        col.Color,
			"column_index": col.ColumnIndex,
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

// MoveColumn repositions a column to newIndex, shifting the columns
// between its old and new indexes one step toward the vacated slot. One
// transaction: column_index is UNIQUE, so the moved row is parked above
// every live index first, keeping each intermediate state collision-free.
func (r *mysqlQueryRepo) MoveColumn(id int, newIndex int) error {
	if newIndex < models.FirstCategoryIndex {
		return fmt.Errorf("cannot move a column to index %d: indexes 1-%d are the fixed register block",
			newIndex, models.FirstCategoryIndex-1)
	}
	return r.Conn.Transaction(func(tx *gorm.DB) error {
		var col models.Column
		if err := tx.First(&col, id).Error; err != nil {
			return fmt.Errorf("unable to find column %d: %w", id, err)
		}
		old := col.ColumnIndex
		if newIndex == old {
			return nil
		}
		var maxIndex int
		if err := tx.Model(&models.Column{}).Select("COALESCE(MAX(column_index), 0)").Scan(&maxIndex).Error; err != nil {
			return fmt.Errorf("unable to find max column index: %w", err)
		}
		if newIndex > maxIndex {
			return fmt.Errorf("cannot move column %s to index %d: last column is %d", col.Name, newIndex, maxIndex)
		}
		if err := tx.Model(&models.Column{}).Where("id = ?", id).
			Update("column_index", maxIndex+1).Error; err != nil {
			return fmt.Errorf("unable to park column %s: %w", col.Name, err)
		}
		lo, hi, delta, order := old+1, newIndex, -1, "column_index asc"
		if newIndex < old {
			lo, hi, delta, order = newIndex, old-1, 1, "column_index desc"
		}
		var shifting []models.Column
		if err := tx.Where("column_index >= ? AND column_index <= ?", lo, hi).
			Order(order).Find(&shifting).Error; err != nil {
			return fmt.Errorf("unable to shift columns: %w", err)
		}
		for i := range shifting {
			if err := tx.Model(&models.Column{}).Where("id = ?", shifting[i].ID).
				Update("column_index", shifting[i].ColumnIndex+delta).Error; err != nil {
				return fmt.Errorf("unable to shift column %s: %w", shifting[i].Name, err)
			}
		}
		if err := tx.Model(&models.Column{}).Where("id = ?", id).
			Update("column_index", newIndex).Error; err != nil {
			return fmt.Errorf("unable to place column %s: %w", col.Name, err)
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
