package sheets_service

import (
	"fmt"
	"strings"
)

// rowDecoder maps header names to column positions, built once from a header
// row. It replaces positional values[index] reads: a missing, renamed, or
// moved header is a loud error, not a silently wrong column.
type rowDecoder struct {
	index map[string]int
}

// normalizeHeader canonicalizes a header label for matching: surrounding
// whitespace is ignored and matching is case-insensitive, so ' Withdrawals '
// and 'withdrawals' are the same header.
func normalizeHeader(h string) string {
	return strings.ToLower(strings.TrimSpace(h))
}

// newRowDecoder builds a decoder from a header row. Empty header cells are
// skipped; a duplicated header name is an error.
func newRowDecoder(header []interface{}) (*rowDecoder, error) {
	d := &rowDecoder{index: make(map[string]int)}
	for i, h := range header {
		name := normalizeHeader(fmt.Sprintf("%s", h))
		if name == "" {
			continue
		}
		if _, dup := d.index[name]; dup {
			return nil, fmt.Errorf("duplicate header %q in header row", h)
		}
		d.index[name] = i
	}
	return d, nil
}

// indexOf returns the 0-based column index for a header name, or -1.
func (d *rowDecoder) indexOf(name string) int {
	if i, ok := d.index[normalizeHeader(name)]; ok {
		return i
	}
	return -1
}

// require returns the column index for a header name, or a loud error naming
// the missing header.
func (d *rowDecoder) require(name string) (int, error) {
	if i := d.indexOf(name); i >= 0 {
		return i, nil
	}
	return -1, fmt.Errorf("expected header %q not found", name)
}

// cell returns the cell at the named column, or "" when the row is short.
func (d *rowDecoder) cell(row []interface{}, name string) interface{} {
	i := d.indexOf(name)
	if i < 0 || i >= len(row) {
		return ""
	}
	return row[i]
}
