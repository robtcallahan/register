package models

import (
	"reflect"
	"strings"
	"testing"
)

// TestTransactionKey_UniqueIndex verifies the 9.1 fix: the transactions.key
// column must have a unique index, otherwise UpdateTransactionTables'
// OnConflict(UpdateAll) never conflicts and duplicates accumulate.
func TestTransactionKey_UniqueIndex(t *testing.T) {
	field, ok := reflect.TypeOf(Transaction{}).FieldByName("Key")
	if !ok {
		t.Fatal("Transaction has no Key field")
	}

	tag := field.Tag.Get("gorm")
	if !strings.Contains(tag, "uniqueIndex") {
		t.Errorf("Transaction.Key gorm tag = %q, want it to contain uniqueIndex", tag)
	}
	if !strings.Contains(tag, "size:255") {
		t.Errorf("Transaction.Key gorm tag = %q, want it to contain size:255", tag)
	}
}
