package repository

import (
	"fmt"

	"register/pkg/config"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// Connect opens the MySQL database described by db and returns the GORM handle.
//
// This replaces the old pkg/driver package: with postgres gone, the DBType
// switch had exactly one branch left, so a whole package for "pick a driver"
// was just ceremony around building a DSN string.
func Connect(db *config.DBConfig) (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?charset=utf8&parseTime=true",
		db.Username,
		db.Password,
		db.Host,
		db.Port,
		db.Name,
	)
	return gorm.Open(mysql.Open(dsn), &gorm.Config{})
}
