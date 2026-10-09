package models

import "fmt"

// Validate reports structural problems in the set: non-positive, duplicate,
// or missing indexes. An empty result means the indexes run gapless from 1 —
// the precondition the sheet write path relies on.
func (s *ColumnSet) Validate() []string {
	var problems []string
	seen := make(map[int]string)
	max := 0
	for _, col := range s.columns {
		if col.ColumnIndex < 1 {
			problems = append(problems, fmt.Sprintf("column %q has non-positive index %d", col.Name, col.ColumnIndex))
			continue
		}
		if prev, dup := seen[col.ColumnIndex]; dup {
			problems = append(problems, fmt.Sprintf("index %d is shared by %q and %q", col.ColumnIndex, prev, col.Name))
		}
		seen[col.ColumnIndex] = col.Name
		if col.ColumnIndex > max {
			max = col.ColumnIndex
		}
	}
	for i := 1; i <= max; i++ {
		if _, ok := seen[i]; !ok {
			problems = append(problems, fmt.Sprintf("no column at index %d (%s)", i, ColumnLetter(i)))
		}
	}
	return problems
}

// ColumnSet wraps the columns-table rows fetched for a run and is the single
// place that knows how DB column indexes map to spreadsheet positions.
//
// The DB stores ColumnIndex 1-based (A=1). Conversions out of that base —
// 0-based slice offsets and A1 column letters — happen here and nowhere else,
// so the rest of the code never has to remember which base it is holding.
type ColumnSet struct {
	columns []Column
	byIndex map[int]*Column
	byName  map[string]*Column
	byID    map[int]*Column
}

// NewColumnSet builds a ColumnSet from rows as returned by GetColumns
// (ordered by ColumnIndex).
func NewColumnSet(columns []Column) *ColumnSet {
	s := &ColumnSet{
		columns: columns,
		byIndex: make(map[int]*Column, len(columns)),
		byName:  make(map[string]*Column, len(columns)),
		byID:    make(map[int]*Column, len(columns)),
	}
	for i := range s.columns {
		col := &s.columns[i]
		s.byIndex[col.ColumnIndex] = col
		s.byName[col.Name] = col
		s.byID[col.ID] = col
	}
	return s
}

// Columns returns the wrapped rows in their original order.
func (s *ColumnSet) Columns() []Column { return s.columns }

// ByIndex returns the column at a 1-based DB index.
func (s *ColumnSet) ByIndex(index int) (*Column, bool) {
	col, ok := s.byIndex[index]
	return col, ok
}

// IndexFor returns the 1-based DB index of the named column.
func (s *ColumnSet) IndexFor(name string) (int, bool) {
	col, ok := s.byName[name]
	if !ok {
		return 0, false
	}
	return col.ColumnIndex, true
}

// IndexForID returns the 1-based DB index of the column with the given ID.
func (s *ColumnSet) IndexForID(id int) (int, bool) {
	col, ok := s.byID[id]
	if !ok {
		return 0, false
	}
	return col.ColumnIndex, true
}

// End returns the column with the highest index — the register's last column.
func (s *ColumnSet) End() (*Column, bool) {
	if len(s.columns) == 0 {
		return nil, false
	}
	end := &s.columns[0]
	for i := range s.columns {
		if s.columns[i].ColumnIndex > end.ColumnIndex {
			end = &s.columns[i]
		}
	}
	return end, true
}

// LetterFor converts a 1-based column index to its A1 letter (1=A, 27=AA).
func (s *ColumnSet) LetterFor(index int) string {
	return ColumnLetter(index)
}

// SliceOffset converts a 1-based column index to a 0-based slice offset.
// This is the one place the 1-based DB index becomes a 0-based position.
func (s *ColumnSet) SliceOffset(index int) int {
	return index - 1
}

// ColumnLetter converts a 1-based column index to its A1 letter (1=A, 27=AA).
func ColumnLetter(index int) string {
	letter := ""
	for index > 0 {
		index--
		letter = string(rune('A'+index%26)) + letter
		index /= 26
	}
	return letter
}
