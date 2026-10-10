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
