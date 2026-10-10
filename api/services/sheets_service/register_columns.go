package sheets_service

// Register column surgery and drift checking: keeping the sheet's
// columns in step with the columns table.

import (
	"fmt"
	"regexp"
	"strings"
	"register/pkg/models"

	"google.golang.org/api/sheets/v4"
)

// CheckRegisterColumns reports drift between the columns table and the
// Register tab: structural problems in the DB indexes, a sheet grid wider
// or narrower than the DB's last column, and row-4 header names that
// disagree with the DB. Report-only; it never writes anything.
func (ss *SheetsService) CheckRegisterColumns(columns []models.Column) ([]string, error) {
	set := models.NewColumnSet(columns)
	problems := set.Validate()

	headerRange := fmt.Sprintf("%s!A4:%s4", ss.RegisterSheet.TabName, ss.RegisterSheet.SheetCoords.EndColumnName)
	resp, err := ss.Provider.GetValues(headerRange)
	if err != nil {
		return nil, fmt.Errorf("could not read header row: %w", err)
	}
	var headerRow []interface{}
	if len(resp.Values) > 0 {
		headerRow = resp.Values[0]
	}
	problems = append(problems, compareColumnHeaders(columns, headerRow)...)

	if end, ok := set.End(); ok {
		if count := ss.RegisterSheet.SheetCoords.ColumnCount; count > 0 && int64(end.ColumnIndex) != count {
			problems = append(problems, fmt.Sprintf(
				"column count mismatch: DB ends at %s (index %d) but the sheet grid has %d columns",
				set.LetterFor(end.ColumnIndex), end.ColumnIndex, count))
		}
	}
	return problems, nil
}

// compareColumnHeaders matches each category column's DB name against the
// sheet's row-4 header at its position. Only category columns (K, index 11
// and up) are compared: the fixed block's sheet labels ("Register") predate
// its DB names ("BankRegister") and are the code's business, not drift.
func compareColumnHeaders(columns []models.Column, headerRow []interface{}) []string {
	var problems []string
	set := models.NewColumnSet(columns)
	for _, col := range columns {
		if col.ColumnIndex < models.FirstCategoryIndex {
			continue
		}
		header := ""
		if offset := set.SliceOffset(col.ColumnIndex); offset >= 0 && offset < len(headerRow) {
			if s, ok := headerRow[offset].(string); ok {
				header = strings.TrimSpace(s)
			}
		}
		if header != col.Name {
			problems = append(problems, fmt.Sprintf("header mismatch at %s (index %d): DB has %q, sheet has %q",
				set.LetterFor(col.ColumnIndex), col.ColumnIndex, col.Name, header))
		}
	}
	return problems
}

// InsertRegisterColumn inserts a blank sheet column at the given 1-based
// index, labels it in the row-4 header, and fills its running-total
// formulas by translating the left neighbor column's formulas. The
// position math is the caller's (DB) job; this keeps the sheet in step.
func (ss *SheetsService) InsertRegisterColumn(index int, name string) error {
	insertReq := sheets.BatchUpdateSpreadsheetRequest{
		Requests: []*sheets.Request{{
			InsertDimension: &sheets.InsertDimensionRequest{
				Range: &sheets.DimensionRange{
					SheetId:    ss.RegisterSheet.ID,
					Dimension:  "COLUMNS",
					StartIndex: int64(index - 1),
					EndIndex:   int64(index),
				},
				InheritFromBefore: true,
			},
		}},
	}
	if _, err := ss.Provider.BatchUpdate(&insertReq); err != nil {
		return fmt.Errorf("could not insert column: %w", err)
	}

	rowCount := ss.RegisterSheet.SheetCoords.RowCount
	newLetter := models.ColumnLetter(index)
	values := make([][]interface{}, rowCount)
	for i := range values {
		values[i] = []interface{}{""}
	}
	if rowCount > 3 {
		values[3] = []interface{}{name} // row 4 holds the column name
	}
	if index > 1 && rowCount > 0 {
		neighborLetter := models.ColumnLetter(index - 1)
		formulas, err := ss.readColumnFormulas(neighborLetter, rowCount)
		if err != nil {
			return err
		}
		// totals formulas live in the data rows (row 5 down); rows 1-4 are
		// titles, the totals block, the group header, and the name
		for row := 4; row < len(formulas); row++ {
			if formulas[row] != "" {
				values[row] = []interface{}{shiftFormulaColumn(formulas[row], neighborLetter, newLetter)}
			}
		}
	}
	writeRange := fmt.Sprintf("%s!%s1:%s%d", ss.RegisterSheet.TabName, newLetter, newLetter, rowCount)
	if _, err := ss.Provider.Update(writeRange, &sheets.ValueRange{Values: values}); err != nil {
		return fmt.Errorf("could not write column %s: %w", newLetter, err)
	}
	return nil
}

// DeleteRegisterColumn removes the sheet column at the given 1-based index.
func (ss *SheetsService) DeleteRegisterColumn(index int) error {
	deleteReq := sheets.BatchUpdateSpreadsheetRequest{
		Requests: []*sheets.Request{{
			DeleteDimension: &sheets.DeleteDimensionRequest{
				Range: &sheets.DimensionRange{
					SheetId:    ss.RegisterSheet.ID,
					Dimension:  "COLUMNS",
					StartIndex: int64(index - 1),
					EndIndex:   int64(index),
				},
			},
		}},
	}
	if _, err := ss.Provider.BatchUpdate(&deleteReq); err != nil {
		return fmt.Errorf("could not delete column %s: %w", models.ColumnLetter(index), err)
	}
	return nil
}

// MoveRegisterColumn moves the sheet column at fromIndex (1-based) so it
// ends up immediately after the column currently at afterIndex (1-based),
// carrying the column's values, formulas, and header color with it.
// MoveDimension's DestinationIndex counts positions in pre-removal
// coordinates, which is exactly what "after column N" means here.
func (ss *SheetsService) MoveRegisterColumn(fromIndex, afterIndex int) error {
	moveReq := sheets.BatchUpdateSpreadsheetRequest{
		Requests: []*sheets.Request{{
			MoveDimension: &sheets.MoveDimensionRequest{
				Source: &sheets.DimensionRange{
					SheetId:    ss.RegisterSheet.ID,
					Dimension:  "COLUMNS",
					StartIndex: int64(fromIndex - 1),
					EndIndex:   int64(fromIndex),
				},
				DestinationIndex: int64(afterIndex),
			},
		}},
	}
	if _, err := ss.Provider.BatchUpdate(&moveReq); err != nil {
		return fmt.Errorf("could not move column %s: %w", models.ColumnLetter(fromIndex), err)
	}
	return nil
}

// PlanColumnMove works out where the column at index `from` lands when
// moved to just after index `after` (after == 0 means "to the end"), and
// whether the move is allowed: a color-group crossing — landing outside
// the moved column's own color run, or splitting another color's run —
// is refused, because both the Register's grouping and the Budget tab's
// group totals depend on colors staying in contiguous runs. changed is
// false for a no-op move. Columns must be ordered by ColumnIndex.
func PlanColumnMove(columns []models.Column, from, after int) (to int, changed bool, err error) {
	byIndex := make(map[int]models.Column, len(columns))
	for _, c := range columns {
		byIndex[c.ColumnIndex] = c
	}
	moved, ok := byIndex[from]
	if !ok {
		return 0, false, fmt.Errorf("no column at index %d", from)
	}
	if from < models.FirstCategoryIndex {
		return 0, false, fmt.Errorf("cannot move %q: indexes 1-%d are the fixed register block",
			moved.Name, models.FirstCategoryIndex-1)
	}
	to = len(columns)
	if after != 0 {
		if _, ok := byIndex[after]; !ok {
			return 0, false, fmt.Errorf("no column at index %d", after)
		}
		if after == from {
			return 0, false, fmt.Errorf("cannot move %q after itself", moved.Name)
		}
		if after < models.FirstCategoryIndex-1 {
			return 0, false, fmt.Errorf("cannot move %q after %q: that lands in the fixed register block",
				moved.Name, byIndex[after].Name)
		}
		to = after
		if from > after {
			to = after + 1
		}
	}
	changed = to != from
	if !changed {
		return to, false, nil
	}
	final := ApplyColumnMove(columns, from, to)
	for color := range colorsPresent(columns) {
		if contiguous(final, color) || !contiguous(columns, color) {
			continue
		}
		return 0, false, fmt.Errorf("moving %q to index %d would break up the %s color group — columns stay inside their color group so the Budget totals keep meaning the same thing",
			moved.Name, to, colorOrNone(color))
	}
	return to, true, nil
}

func colorOrNone(color string) string {
	if color == "" {
		return "uncolored"
	}
	return color
}

func colorsPresent(columns []models.Column) map[string]bool {
	present := make(map[string]bool, 4)
	for _, c := range columns {
		present[c.Color] = true
	}
	return present
}

// contiguous reports whether every column of the given color sits in one
// uninterrupted run of the (index-ordered) slice.
func contiguous(columns []models.Column, color string) bool {
	first, last, count := -1, -1, 0
	for i, c := range columns {
		if c.Color != color {
			continue
		}
		if first < 0 {
			first = i
		}
		last, count = i, count+1
	}
	return count == 0 || last-first+1 == count
}

// ApplyColumnMove returns the ordering that results from moving the
// column at 1-based index `from` to 1-based index `to`.
func ApplyColumnMove(columns []models.Column, from, to int) []models.Column {
	out := make([]models.Column, 0, len(columns))
	var moved models.Column
	for _, c := range columns {
		if c.ColumnIndex == from {
			moved = c
			continue
		}
		out = append(out, c)
	}
	pos := to - 1 // 0-based slot in the shrunk list
	if pos > len(out) {
		pos = len(out)
	}
	out = append(out, models.Column{})
	copy(out[pos+1:], out[pos:])
	out[pos] = moved
	return out
}

// RenameRegisterColumnHeader rewrites the row-4 name cell of the column at
// the given 1-based index.
func (ss *SheetsService) RenameRegisterColumnHeader(index int, name string) error {
	_, err := ss.WriteCell(models.ColumnLetter(index)+"4", name)
	return err
}

// readColumnFormulas returns the formulas of one sheet column, one entry per
// row ("" where the cell holds no formula).
func (ss *SheetsService) readColumnFormulas(letter string, rowCount int64) ([]string, error) {
	rng := fmt.Sprintf("%s!%s1:%s%d", ss.RegisterSheet.TabName, letter, letter, rowCount)
	resp, err := ss.Provider.GetFormula(rng)
	if err != nil {
		return nil, fmt.Errorf("could not read formulas from column %s: %w", letter, err)
	}
	formulas := make([]string, rowCount)
	for i, row := range resp.Values {
		if i >= int(rowCount) {
			break
		}
		if len(row) > 0 {
			if s, ok := row[0].(string); ok && strings.HasPrefix(s, "=") {
				formulas[i] = s
			}
		}
	}
	return formulas, nil
}

var cellRefRegex = regexp.MustCompile(`\$?([A-Z]+)\$?(\d+)`)

// shiftFormulaColumn rewrites a formula so references to fromLetter point
// at toLetter instead. Register totals formulas reference only their own
// column, so this moves a copied formula into its new column wholesale.
func shiftFormulaColumn(formula, fromLetter, toLetter string) string {
	return cellRefRegex.ReplaceAllStringFunc(formula, func(ref string) string {
		sub := cellRefRegex.FindStringSubmatch(ref)
		if sub[1] != fromLetter {
			return ref
		}
		return strings.Replace(ref, sub[1], toLetter, 1)
	})
}
