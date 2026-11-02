package models

import "testing"

func TestColumnIsCategory(t *testing.T) {
	tests := []struct {
		name  string
		index int
		want  bool
	}{
		{"zero index", 0, false},
		{"last fixed position", 10, false},
		{"first category position", FirstCategoryIndex, true},
		{"later category", 42, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			col := Column{ColumnIndex: tt.index}
			if got := col.IsCategory(); got != tt.want {
				t.Errorf("Column{ColumnIndex: %d}.IsCategory() = %v, want %v", tt.index, got, tt.want)
			}
		})
	}
}
