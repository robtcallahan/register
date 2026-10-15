package mysql

import (
	"os"
	"testing"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"register/pkg/models"
)

// testRepo returns a mysqlQueryRepo against the test DB, or skips the test
// if REGISTER_TEST_MYSQL_DSN is not set.
func testRepo(t *testing.T) *mysqlQueryRepo {
	t.Helper()
	dsn := os.Getenv("REGISTER_TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("REGISTER_TEST_MYSQL_DSN not set; skipping DB test")
	}
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("could not connect to MySQL: %v", err)
	}
	return &mysqlQueryRepo{Conn: db}
}

// TestGetColumns verifies columns come back ordered by column_index.
func TestGetColumns(t *testing.T) {
	r := testRepo(t)
	cols, err := r.GetColumns()
	if err != nil {
		t.Fatalf("GetColumns() error = %v", err)
	}
	if len(cols) == 0 {
		t.Errorf("GetColumns() returned no columns")
	}
	for i := 1; i < len(cols); i++ {
		if cols[i].ColumnIndex < cols[i-1].ColumnIndex {
			t.Errorf("columns not ordered: index %d (%s) after %d (%s)",
				cols[i].ColumnIndex, cols[i].Name, cols[i-1].ColumnIndex, cols[i-1].Name)
			break
		}
	}
}

// TestAddRenameDeleteColumn exercises the column lifecycle: add shifts
// later columns up, rename changes the name, delete removes it and shifts
// later columns down.
func TestAddRenameDeleteColumn(t *testing.T) {
	r := testRepo(t)

	// Add a column at index 11 (first category slot)
	newCol := &models.Column{Name: "TestCol1013", Color: "blue", ColumnIndex: 11}
	if err := r.AddColumn(newCol); err != nil {
		t.Fatalf("AddColumn() error = %v", err)
	}

	// Find it
	cols, err := r.GetColumns()
	if err != nil {
		t.Fatalf("GetColumns() error = %v", err)
	}
	var added *models.Column
	for i := range cols {
		if cols[i].Name == "TestCol1013" {
			added = &cols[i]
			break
		}
	}
	if added == nil {
		t.Fatal("AddColumn() did not create the column")
	}
	if added.ColumnIndex != 11 {
		t.Errorf("added column index = %d, want 11", added.ColumnIndex)
	}

	// Rename it
	if err := r.RenameColumn(int(added.ID), "TestCol1013Renamed"); err != nil {
		t.Fatalf("RenameColumn() error = %v", err)
	}
	cols, _ = r.GetColumns()
	found := false
	for _, c := range cols {
		if c.Name == "TestCol1013Renamed" {
			found = true
			added.ID = c.ID
			break
		}
	}
	if !found {
		t.Fatal("RenameColumn() did not rename the column")
	}

	// Delete it
	if err := r.DeleteColumn(int(added.ID)); err != nil {
		t.Fatalf("DeleteColumn() error = %v", err)
	}
	cols, _ = r.GetColumns()
	for _, c := range cols {
		if c.Name == "TestCol1013Renamed" {
			t.Errorf("DeleteColumn() did not delete the column")
			break
		}
	}
}

// TestMoveColumn verifies a column can be repositioned.
func TestMoveColumn(t *testing.T) {
	r := testRepo(t)

	// Add two test columns
	c1 := &models.Column{Name: "TestMoveA", Color: "blue", ColumnIndex: 11}
	c2 := &models.Column{Name: "TestMoveB", Color: "blue", ColumnIndex: 12}
	if err := r.AddColumn(c1); err != nil {
		t.Fatalf("AddColumn() error = %v", err)
	}
	if err := r.AddColumn(c2); err != nil {
		t.Fatalf("AddColumn() error = %v", err)
	}

	// Find their IDs
	cols, _ := r.GetColumns()
	var idA, idB int
	for _, c := range cols {
		if c.Name == "TestMoveA" {
			idA = c.ID
		}
		if c.Name == "TestMoveB" {
			idB = c.ID
		}
	}

	// Move A to position 12 (should swap with B)
	if err := r.MoveColumn(int(idA), 12); err != nil {
		t.Fatalf("MoveColumn() error = %v", err)
	}

	cols, _ = r.GetColumns()
	var idxA, idxB int
	for _, c := range cols {
		if c.Name == "TestMoveA" {
			idxA = c.ColumnIndex
		}
		if c.Name == "TestMoveB" {
			idxB = c.ColumnIndex
		}
	}
	if idxA != 12 {
		t.Errorf("TestMoveA index = %d, want 12", idxA)
	}
	if idxB != 11 {
		t.Errorf("TestMoveB index = %d, want 11", idxB)
	}

	// Cleanup
	r.DeleteColumn(int(idA))
	r.DeleteColumn(int(idB))

	// Move to fixed block should fail
	if err := r.MoveColumn(int(idA), 5); err == nil {
		t.Errorf("MoveColumn() to index 5 should fail (fixed block)")
	}
}

// TestCreateAndGetMerchants verifies merchant CRUD.
func TestCreateAndGetMerchants(t *testing.T) {
	r := testRepo(t)

	m := &models.Merchant{
		BankName:  "TEST BANK 1013",
		Name:      "Test Merchant",
		ColumnID:  11,
		MatchType: "exact",
		Pattern:   "TEST BANK 1013",
		Priority:  0,
	}
	if err := r.CreateMerchant(m); err != nil {
		t.Fatalf("CreateMerchant() error = %v", err)
	}

	merchants, err := r.GetMerchants()
	if err != nil {
		t.Fatalf("GetMerchants() error = %v", err)
	}
	found := false
	for _, mm := range merchants {
		if mm.BankName == "TEST BANK 1013" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("CreateMerchant() did not persist the merchant")
	}

	// Cleanup: delete via raw SQL to avoid affecting other tests
	r.Conn.Unscoped().Where("bank_name = ?", "TEST BANK 1013").Delete(&models.Merchant{})
}

// TestGetMaxMerchantPriority verifies the priority query.
func TestGetMaxMerchantPriority(t *testing.T) {
	r := testRepo(t)
	max, err := r.GetMaxMerchantPriority()
	if err != nil {
		t.Fatalf("GetMaxMerchantPriority() error = %v", err)
	}
	// Should not error; value depends on data
	t.Logf("max merchant priority = %d", max)
}

// TestGetMerchantsByColumn verifies filtering by column.
func TestGetMerchantsByColumn(t *testing.T) {
	r := testRepo(t)
	merchants, err := r.GetMerchantsByColumn(11)
	if err != nil {
		t.Fatalf("GetMerchantsByColumn() error = %v", err)
	}
	for _, m := range merchants {
		if m.ColumnID != 11 {
			t.Errorf("GetMerchantsByColumn(11) returned merchant with ColumnID %d", m.ColumnID)
		}
	}
}

// TestSaveAndGetTransactions verifies transaction persistence.
func TestSaveAndGetTransactions(t *testing.T) {
	r := testRepo(t)

	// Ensure the table exists (the dump may not include it)
	if err := r.Conn.AutoMigrate(&models.Transaction{}); err != nil {
		t.Fatalf("AutoMigrate() error = %v", err)
	}

	tx := &models.Transaction{
		Key:      "test:1013:1",
		Source:   "TestBank",
		Date:     "10/13/26",
		Name:     "Test Transaction",
		Amount:   42.00,
		Withdrawal: 42.00,
	}
	if err := r.SaveTransaction(tx); err != nil {
		t.Fatalf("SaveTransaction() error = %v", err)
	}

	trans, err := r.GetTransactions()
	if err != nil {
		t.Fatalf("GetTransactions() error = %v", err)
	}
	found := false
	for _, tr := range trans {
		if tr.Key == "test:1013:1" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("SaveTransaction() did not persist the transaction")
	}

	// Cleanup
	r.Conn.Unscoped().Where("`key` = ?", "test:1013:1").Delete(&models.Transaction{})
}

// TestGetLookupData verifies the lookup data query.
func TestGetLookupData(t *testing.T) {
	r := testRepo(t)
	rows, err := r.GetLookupData()
	if err != nil {
		t.Fatalf("GetLookupData() error = %v", err)
	}
	// Should not error; may be empty
	t.Logf("lookup data rows = %d", len(rows))
}

// TestGetNameMapToColumn verifies the name-to-column mapping.
func TestGetNameMapToColumn(t *testing.T) {
	r := testRepo(t)
	m, err := r.GetNameMapToColumn()
	if err != nil {
		t.Fatalf("GetNameMapToColumn() error = %v", err)
	}
	if len(m) == 0 {
		t.Errorf("GetNameMapToColumn() returned empty map")
	}
}
