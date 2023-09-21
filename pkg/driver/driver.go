package driver

import (
	"fmt"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// DBType ...
type DBType string

const (
	// MySQL ...
	MySQL = "MySQL"
	// PostgreSQL ...
	PostgreSQL = "PostgreSQL"
)

// DB ...
type DB struct {
	DBType DBType
	SQL    *gorm.DB
}

// ConnectParams ...
type ConnectParams struct {
	DBType DBType
	Host   string
	Port   string
	DBName string
	User   string
	Pass   string
}

// ConnectSQL ...
func ConnectSQL(c *ConnectParams) (*DB, error) {
	var err error
	db := &gorm.DB{}

	switch c.DBType {
	case MySQL:
		dsn := fmt.Sprintf(
			"%s:%s@tcp(%s:%s)/%s?charset=utf8&parseTime=true",
			c.User,
			c.Pass,
			c.Host,
			c.Port,
			c.DBName,
		)
		db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	case PostgreSQL:
		dsn := fmt.Sprintf(
			"user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=America/New_York",
			c.User,
			c.Pass,
			c.DBName,
			c.Port,
		)
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	default:
		panic(fmt.Errorf("unknown DBType %s", c.DBType))
	}
	if err != nil {
		return nil, err
	}
	dbConn := &DB{
		DBType: c.DBType,
		SQL:    db,
	}
	return dbConn, err
}
