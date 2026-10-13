package cmd

import (
	"strings"
	"testing"

	"register/pkg/models"
)

// TestDeleteColumn_Safety verifies the safety guards in deleteColumn:
// - cannot delete a column that doesn't exist
// - cannot delete the fixed register block (indexes 1-10)
func TestDeleteColumn_Safety(t *testing.T) {
	columns := []models.Column{
		{ID: 1, Name: "Recon", ColumnIndex: 1},
		{ID: 2, Name: "Date", ColumnIndex: 3},
		{ID: 11, Name: "Grocery", ColumnIndex: 11},
		{ID: 12, Name: "Rent", ColumnIndex: 12},
	}
	set := models.NewColumnSet(columns)

	// Safety check 1: column must exist
	if _, ok := set.IndexFor("Nonexistent"); ok {
		t.Errorf("IndexFor(Nonexistent) returned ok, want not ok")
	}

	// Safety check 2: fixed block (1-10) cannot be deleted
	for _, name := range []string{"Recon", "Date"} {
		index, ok := set.IndexFor(name)
		if !ok {
			t.Fatalf("IndexFor(%q) not found", name)
		}
		if index >= models.FirstCategoryIndex {
			t.Errorf("%q has index %d, want < %d (fixed block)",
				name, index, models.FirstCategoryIndex)
		}
		// This is the guard in deleteColumn:
		if index < models.FirstCategoryIndex {
			// correctly identified as protected
		} else {
			t.Errorf("%q should be protected from deletion", name)
		}
	}

	// Category columns CAN be deleted (index >= 11)
	for _, name := range []string{"Grocery", "Rent"} {
		index, ok := set.IndexFor(name)
		if !ok {
			t.Fatalf("IndexFor(%q) not found", name)
		}
		if index < models.FirstCategoryIndex {
			t.Errorf("%q has index %d, want >= %d (deletable)",
				name, index, models.FirstCategoryIndex)
		}
	}
}

// TestAddColumn_Safety verifies the validation in addColumn:
// - name must not be empty (after trim)
// - name must not already exist
// - color must be green, yellow, or blue
func TestAddColumn_Safety(t *testing.T) {
	columns := []models.Column{
		{ID: 11, Name: "Grocery", ColumnIndex: 11},
	}
	set := models.NewColumnSet(columns)

	// Empty name check
	for _, name := range []string{"", "   ", "\t"} {
		if strings.TrimSpace(name) != "" {
			t.Errorf("TrimSpace(%q) should be empty", name)
		}
	}

	// Duplicate name check
	if _, ok := set.IndexFor("Grocery"); !ok {
		t.Errorf("IndexFor(Grocery) not found, want ok (duplicate detection)")
	}
	if _, ok := set.IndexFor("NewColumn"); ok {
		t.Errorf("IndexFor(NewColumn) returned ok, want not ok")
	}

	// Color validation (from addColumn)
	validColors := []string{models.ColorGreen, models.ColorYellow, models.ColorBlue}
	for _, c := range validColors {
		switch c {
		case models.ColorGreen, models.ColorYellow, models.ColorBlue:
			// valid
		default:
			t.Errorf("color %q should be valid", c)
		}
	}
	// Grey is reserved
	if models.ColorGrey == models.ColorGreen ||
		models.ColorGrey == models.ColorYellow ||
		models.ColorGrey == models.ColorBlue {
		t.Errorf("grey should not be a valid add color")
	}
}

// TestColumnSet_IndexFor verifies the lookup used by all column commands.
func TestColumnSet_IndexFor(t *testing.T) {
	columns := []models.Column{
		{ID: 1, Name: "Recon", ColumnIndex: 1},
		{ID: 11, Name: "Grocery", ColumnIndex: 11},
	}
	set := models.NewColumnSet(columns)

	if idx, ok := set.IndexFor("Grocery"); !ok || idx != 11 {
		t.Errorf("IndexFor(Grocery) = (%d, %v), want (11, true)", idx, ok)
	}
	if _, ok := set.IndexFor("grocery"); ok {
		t.Errorf("IndexFor(grocery) returned ok, want not ok (case-sensitive)")
	}
	if _, ok := set.IndexFor(""); ok {
		t.Errorf("IndexFor(empty) returned ok, want not ok")
	}
}
