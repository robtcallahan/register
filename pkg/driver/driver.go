package driver

import (
	"fmt"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// DB ...
type DB struct {
	SQL *gorm.DB
	// Mgo *mgo.database
}

// ConnectParams ...
type ConnectParams struct {
	Host   string
	Port   string
	DBName string
	User   string
	Pass   string
}

// DBConn ...
var dbConn = &DB{}

// ConnectSQL ...
func ConnectSQL(c *ConnectParams) (*DB, error) {
	dbSource := fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?charset=utf8",
		c.User,
		c.Pass,
		c.Host,
		c.Port,
		c.DBName,
	)
	db, err := gorm.Open(mysql.Open(dbSource), &gorm.Config{})
	if err != nil {
		panic(err)
	}
	dbConn.SQL = db
	return dbConn, err
}
