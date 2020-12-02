package mysql

import (
	"fmt"

	repo "register/pkg/repository"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	// _ "github.com/go-sql-driver/mysql"
)

// Repository ...
type Repository struct {
	db *gorm.DB
}

// NewRepository ...
func (r *repo.Repository) NewRepository(n repo.NewRepositoryParams) (*repo.Repository, error) {
	dsn := n.DBUsername + ":" + n.DBPassword + "@tcp(127.0.0.1:3306)/" + n.DBName + "?charset=utf8mb4&parseTime=True&loc=Local"

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	return &Repository{db: db}, nil
}

// CreateMerchant ...
func (r *Repository) CreateMerchant(m *repo.Merchant) {
	result := r.db.Create(&repo.Merchant{
		Name:     m.Name,
		BankName: m.BankName,
		ColumnID: m.ColumnID,
	})
	if result.Error != nil {
		panic(result.Error)
	}
}

// GetLookupData ...
func (r *Repository) GetLookupData() []*repo.DataRow {
	var merchants []repo.Merchant

	r.db.Preload("Column").Find(&merchants)

	var data = []*repo.DataRow{}
	for _, m := range merchants {
		data = append(data, &repo.DataRow{
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

// GetColumns ...
func (r *Repository) GetColumns() []repo.Column {
	var cols []repo.Column

	r.db.Order("column_index").Find(&cols)

	return cols
}

// GetNameMapToColumn creates a map lookup from trans name to budget category/column names
func (r *Repository) GetNameMapToColumn() map[string]string {
	cols := r.GetLookupData()

	nameToCol := make(map[string]string)
	for _, c := range cols {
		nameToCol[c.Name] = c.ColumnName
	}
	return nameToCol
}

// PrintData ...
func (r *Repository) PrintData() {
	var merchants []repo.Merchant
	r.db.Preload("Column").Find(&merchants)

	fmt.Printf("[Num] %-35s %-30s %-30s %-s\n", "Bank Name", "Name", "Column Name", "Column Index")
	for i, m := range merchants {
		fmt.Printf("[%3d] %-35s %-30s %-30s %2d\n", i+1, m.BankName, m.Name, m.Column.Name, m.Column.ColumnIndex)
	}
}

// PrintTable ...
func (r *Repository) PrintTable(table string) {
	switch table {
	case "merchants":
		var merchants []repo.Merchant
		result := r.db.Find(&merchants)
		fmt.Printf("%d rows found\n", result.RowsAffected)
		for _, m := range merchants {
			fmt.Printf("%d %s %s %s\n", m.ID, m.BankName, m.Name, m.Column.Name)
		}
	}
}
