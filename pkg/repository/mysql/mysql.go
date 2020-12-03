package mysql

import (
	"fmt"

	"register/pkg/models"
	repo "register/pkg/repository"

	"gorm.io/gorm"
)

type queryRepo struct {
	Conn *gorm.DB
}

// NewQueryRepo ...
func NewQueryRepo(conn *gorm.DB) repo.Repository {
	return &queryRepo{
		Conn: conn,
	}
}

// GetColumns ...
func (r *queryRepo) GetColumns() []models.Column {
	var cols []models.Column

	r.Conn.Order("column_index").Find(&cols)

	return cols
}

// CreateMerchant ...
func (r *queryRepo) CreateMerchant(m *models.Merchant) {
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
func (r *queryRepo) GetLookupData() []*models.DataRow {
	var merchants []models.Merchant

	r.Conn.Preload("Column").Find(&merchants)

	var data []*models.DataRow
	for _, m := range merchants {
		data = append(data, &models.DataRow{
			Name:        m.Name,
			BankName:    m.BankName,
			ColumnName:  m.Column.Name,
			ColumnIndex: m.Column.ColumnIndex,
			Color:       m.Column.Color,
			IsCategory:  m.Column.IsCategory,
		})
	}
	return data
}

// GetNameMapToColumn creates a map lookup from trans name to budget category/column names
func (r *queryRepo) GetNameMapToColumn() map[string]string {
	cols := r.GetLookupData()

	nameToCol := make(map[string]string)
	for _, c := range cols {
		nameToCol[c.Name] = c.ColumnName
	}
	return nameToCol
}

// PrintData ...
func (r *queryRepo) PrintData() {
	var merchants []models.Merchant
	r.Conn.Preload("Column").Find(&merchants)

	fmt.Printf("[Num] %-35s %-30s %-30s %-s\n", "Bank Name", "Name", "Column Name", "Column Index")
	for i, m := range merchants {
		fmt.Printf("[%3d] %-35s %-30s %-30s %2d\n", i+1, m.BankName, m.Name, m.Column.Name, m.Column.ColumnIndex)
	}
}

// PrintTable ...
func (r *queryRepo) PrintTable(table string) {
	switch table {
	case "merchants":
		var merchants []models.Merchant
		result := r.Conn.Find(&merchants)
		fmt.Printf("%d rows found\n", result.RowsAffected)
		for _, m := range merchants {
			fmt.Printf("%d %s %s %s\n", m.ID, m.BankName, m.Name, m.Column.Name)
		}
	}
}
