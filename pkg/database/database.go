package database

import (
	"fmt"
	// "sort"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// Merchant ...
type Merchant struct {
	gorm.Model
	BankName string
	Name     string
	ColumnID int
	Column   Column
}

// Column ...
type Column struct {
	ID          int
	Name        string
	Color       string
	ColumnIndex int
	Letter      string
	IsCategory  bool
}

// DataRow ...
type DataRow struct {
	Name        string
	BankName    string
	ColumnName  string
	ColumnIndex int
	Color       string
	IsCategory  bool
}

// ConfigParams ...
type ConfigParams struct {
	Debug      bool
	DBName     string
	DBUsername string
	DBPassword string
}

// Client ...
type Client struct {
	Debug      bool
	DBName     string
	DBUsername string
	DBPassword string
	DSN        string
	DB         *gorm.DB
}

// New ...
func New(c ConfigParams) *Client {
	return &Client{
		DBName:     c.DBName,
		DBUsername: c.DBUsername,
		DBPassword: c.DBPassword,
		DSN:        c.DBUsername + ":" + c.DBPassword + "@tcp(127.0.0.1:3306)/" + c.DBName + "?charset=utf8mb4&parseTime=True&loc=Local",
		Debug:      c.Debug,
	}
}

func (c *Client) connect() {
	if c.DB == nil {
		db, err := gorm.Open(mysql.Open(c.DSN), &gorm.Config{})
		if err != nil {
			panic("failed to connect database")
		}
		c.DB = db
	}
}

// CreateMerchant ...
func (c *Client) CreateMerchant(m *Merchant) {
	result := c.DB.Create(&Merchant{
		Name:     m.Name,
		BankName: m.BankName,
		ColumnID: m.ColumnID,
	})
	if result.Error != nil {
		panic(result.Error)
	}
}

// GetLookupData ...
func (c *Client) GetLookupData() []*DataRow {
	var merchants []Merchant

	c.connect()
	c.DB.Preload("Column").Find(&merchants)

	var data = []*DataRow{}
	for _, m := range merchants {
		data = append(data, &DataRow{
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
func (c *Client) GetColumns() []Column {
	var cols []Column

	c.connect()
	c.DB.Order("column_index").Find(&cols)

	// sort.Slice(cols, func(i, j int) bool {
	// 	return cols[i].ColumnIndex == cols[j].ColumnIndex
	// })

	return cols
}

// GetNameMapToColumn creates a map lookup from trans name to budget category/column names
func (c *Client) GetNameMapToColumn() map[string]string {
	c.connect()
	cols := c.GetLookupData()

	nameToCol := make(map[string]string)
	for _, c := range cols {
		nameToCol[c.Name] = c.ColumnName
	}
	return nameToCol
}

// PrintData ...
func (c *Client) PrintData() {
	c.connect()

	var merchants []Merchant
	c.DB.Preload("Column").Find(&merchants)

	fmt.Printf("[Num] %-35s %-30s %-30s %-s\n", "Bank Name", "Name", "Column Name", "Column Index")
	for i, m := range merchants {
		fmt.Printf("[%3d] %-35s %-30s %-30s %2d\n", i+1, m.BankName, m.Name, m.Column.Name, m.Column.ColumnIndex)
	}
}

// PrintTable ...
func (c *Client) PrintTable(table string) {
	c.connect()

	switch table {
	case "merchants":
		var merchants []Merchant
		result := c.DB.Find(&merchants)
		fmt.Printf("%d rows found\n", result.RowsAffected)
		for _, m := range merchants {
			fmt.Printf("%d %s %s %s\n", m.ID, m.BankName, m.Name, m.Column.Name)
		}
	}
}
