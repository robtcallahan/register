package mysql

import (
	"os"
	"testing"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"register/pkg/models"
)

// TestUpdateTransactionTables_Idempotent verifies that running
// UpdateTransactionTables twice with the same transactions does not
// create duplicates. Requires MySQL; skips if not available.
//
// To run: set REGISTER_TEST_MYSQL_DSN, e.g.
//   REGISTER_TEST_MYSQL_DSN="user:pass@tcp(localhost:3306)/register_test?parseTime=true" go test -run TestUpdateTransactionTables_Idempotent ./pkg/repository/mysql/
func TestUpdateTransactionTables_Idempotent(t *testing.T) {
	dsn := os.Getenv("REGISTER_TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("REGISTER_TEST_MYSQL_DSN not set; skipping DB idempotence test")
	}

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("could not connect to MySQL: %v", err)
	}

	// Use a temp table to avoid touching real data
	if err := db.Migrator().DropTable(&models.Transaction{}); err != nil {
		t.Fatalf("could not drop table: %v", err)
	}
	if err := db.AutoMigrate(&models.Transaction{}); err != nil {
		t.Fatalf("could not migrate: %v", err)
	}
	defer db.Migrator().DropTable(&models.Transaction{})

	repo := &mysqlQueryRepo{Conn: db}

	trans := []*models.Transaction{
		{Key: "wellsfargo:12/07/20:10.00", Source: "WellsFargo", Date: "12/07/20", Amount: 10.00},
		{Key: "wellsfargo:12/07/20:20.00", Source: "WellsFargo", Date: "12/07/20", Amount: 20.00},
	}

	// First run
	if err := repo.UpdateTransactionTables(trans); err != nil {
		t.Fatalf("first UpdateTransactionTables() error = %v", err)
	}

	var count1 int64
	db.Model(&models.Transaction{}).Count(&count1)
	if count1 != 2 {
		t.Fatalf("after first run, count = %d, want 2", count1)
	}

	// Second run with same transactions — should not create duplicates
	if err := repo.UpdateTransactionTables(trans); err != nil {
		t.Fatalf("second UpdateTransactionTables() error = %v", err)
	}

	var count2 int64
	db.Model(&models.Transaction{}).Count(&count2)
	if count2 != 2 {
		t.Errorf("after second run, count = %d, want 2 (idempotence violated)", count2)
	}
}
