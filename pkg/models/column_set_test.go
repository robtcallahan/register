package models

import "testing"

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
