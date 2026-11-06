package models

import (
	"strings"
	"testing"
)

func TestColumnSet(t *testing.T) {
	columns := []Column{
		{ID: 1, Name: "Recon", ColumnIndex: 1},
		{ID: 10, Name: "Delta", ColumnIndex: 10},
		{ID: 58, Name: "Emergency Fund", ColumnIndex: 42},
		{ID: 61, Name: "Last", ColumnIndex: 46},
	}
	set := NewColumnSet(columns)

	if idx, ok := set.IndexFor("Emergency Fund"); !ok || idx != 42 {
		t.Errorf("IndexFor(Emergency Fund) = %d, %v; want 42, true", idx, ok)
	}
	if _, ok := set.IndexFor("Nope"); ok {
		t.Error("IndexFor(Nope) ok = true; want false")
	}
	if idx, ok := set.IndexForID(58); !ok || idx != 42 {
		t.Errorf("IndexForID(58) = %d, %v; want 42, true", idx, ok)
	}
	if _, ok := set.IndexForID(999); ok {
		t.Error("IndexForID(999) ok = true; want false")
	}
	if col, ok := set.ByIndex(10); !ok || col.Name != "Delta" {
		t.Errorf("ByIndex(10) = %v, %v; want Delta, true", col, ok)
	}
	if end, ok := set.End(); !ok || end.ColumnIndex != 46 {
		t.Errorf("End() = %v, %v; want index 46, true", end, ok)
	}
	if _, ok := NewColumnSet(nil).End(); ok {
		t.Error("End() on empty set ok = true; want false")
	}
	for index, want := range map[int]string{1: "A", 26: "Z", 27: "AA", 42: "AP", 46: "AT"} {
		if got := set.LetterFor(index); got != want {
			t.Errorf("LetterFor(%d) = %q; want %q", index, got, want)
		}
	}
	if got := set.SliceOffset(42); got != 41 {
		t.Errorf("SliceOffset(42) = %d; want 41", got)
	}
}

func TestColumnSetValidate(t *testing.T) {
	if problems := NewColumnSet([]Column{
		{ID: 1, Name: "A", ColumnIndex: 1},
		{ID: 2, Name: "B", ColumnIndex: 2},
	}).Validate(); len(problems) != 0 {
		t.Errorf("Validate() on gapless set = %v; want no problems", problems)
	}

	problems := NewColumnSet([]Column{
		{ID: 1, Name: "A", ColumnIndex: 1},
		{ID: 2, Name: "B", ColumnIndex: 1},
		{ID: 3, Name: "D", ColumnIndex: 4},
	}).Validate()
	if len(problems) != 3 { // duplicate index 1, missing 2, missing 3
		t.Errorf("Validate() = %v; want 3 problems", problems)
	}
}

func TestColumnSetValidateNonPositive(t *testing.T) {
	tests := []struct {
		name    string
		columns []Column
		wantLen int
		wantSub string // substring expected in one of the problems ("" = no check)
	}{
		{
			name:    "empty set has no problems",
			columns: nil,
			wantLen: 0,
		},
		{
			name:    "zero index",
			columns: []Column{{ID: 1, Name: "Bad", ColumnIndex: 0}},
			wantLen: 1,
			wantSub: `column "Bad" has non-positive index 0`,
		},
		{
			name:    "negative index",
			columns: []Column{{ID: 1, Name: "Neg", ColumnIndex: -3}},
			wantLen: 1,
			wantSub: `column "Neg" has non-positive index -3`,
		},
		{
			// A non-positive column is skipped, so it neither fills a gap
			// nor extends the range: only its own problem plus the gap at 2.
			name: "non-positive skipped for gaps",
			columns: []Column{
				{ID: 1, Name: "A", ColumnIndex: 1},
				{ID: 2, Name: "Bad", ColumnIndex: 0},
				{ID: 3, Name: "C", ColumnIndex: 3},
			},
			wantLen: 2,
			wantSub: "non-positive index 0",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			problems := NewColumnSet(tt.columns).Validate()
			if len(problems) != tt.wantLen {
				t.Fatalf("Validate() = %v; want %d problems", problems, tt.wantLen)
			}
			if tt.wantSub != "" {
				found := false
				for _, p := range problems {
					if strings.Contains(p, tt.wantSub) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Validate() = %v; want a problem containing %q", problems, tt.wantSub)
				}
			}
		})
	}
}

func TestColumnSetColumns(t *testing.T) {
	columns := []Column{
		{ID: 1, Name: "Recon", ColumnIndex: 1},
		{ID: 10, Name: "Delta", ColumnIndex: 10},
		{ID: 58, Name: "Emergency Fund", ColumnIndex: 42},
	}
	got := NewColumnSet(columns).Columns()
	if len(got) != len(columns) {
		t.Fatalf("Columns() len = %d, want %d", len(got), len(columns))
	}
	for i, want := range columns {
		if got[i] != want {
			t.Errorf("Columns()[%d] = %+v, want %+v", i, got[i], want)
		}
	}
	if got := NewColumnSet(nil).Columns(); len(got) != 0 {
		t.Errorf("Columns() on empty set len = %d, want 0", len(got))
	}
}
