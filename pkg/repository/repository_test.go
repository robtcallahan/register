package repository

import (
	"testing"

	"register/pkg/models"
)

func TestColumnNames(t *testing.T) {
	columns := []models.Column{
		{Name: "Grocery"},
		{Name: "Dining Out"},
		{Name: "Emergency Fund"},
	}
	got := ColumnNames(columns)
	want := []string{"Grocery", "Dining Out", "Emergency Fund"}
	if len(got) != len(want) {
		t.Fatalf("ColumnNames() = %v, want %v", got, want)
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("ColumnNames()[%d] = %q, want %q", i, got[i], w)
		}
	}
	if got := ColumnNames(nil); len(got) != 0 {
		t.Errorf("ColumnNames(nil) len = %d, want 0", len(got))
	}
}
