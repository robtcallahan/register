package models

import (
	"reflect"
	"strings"
	"testing"
)

// TestTransactionKey_Size255 verifies the transactions.key
// column must  be 255 characters in length
func TestTransactionKey_Size255(t *testing.T) {
	field, ok := reflect.TypeFor[Transaction]().FieldByName("Key")
	if !ok {
		t.Fatal("Transaction has no Key field")
	}

	tag := field.Tag.Get("gorm")
	if !strings.Contains(tag, "size:255") {
		t.Errorf("Transaction.Key gorm tag = %q, want it to contain size:255", tag)
	}
}
