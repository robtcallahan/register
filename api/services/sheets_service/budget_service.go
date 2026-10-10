package sheets_service

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"register/pkg/config"
	"register/pkg/models"

	"google.golang.org/api/sheets/v4"
)

// BudgetEntry is one category row of the Budget tab: the raw allocation at
// each of the two cadences Rob gets paid on. (The Yearly column on the sheet
// is display-only; the app never reads it.)
type BudgetEntry struct {
	Category string
	Weekly   float64
	Monthly  float64
}

type BudgetSheet struct {
	ID            int64
	SpreadsheetID string
	TabName       string
	Spreadsheet   sheets.Spreadsheet
	SheetCoords   SheetCoords
	BudgetEntries []*BudgetEntry
	CategoriesMap map[string]*BudgetEntry
}

func (ss *SheetsService) NewBudgetSheet(cfg *config.Config) error {
	ss.BudgetSheet = &BudgetSheet{
		TabName: BudgetTabName,
		SheetCoords: SheetCoords{
			StartRow:       cfg.BudgetStartRow,
			EndRow:         cfg.BudgetEndRow,
			EndColumnName:  "J",
			EndColumnIndex: 9,
		},
	}
	props, err := ss.getSheetProperties(BudgetTabName)
	if err != nil {
		return err
	}
	ss.BudgetSheet.ID = props.SheetId
	return nil
}

func (ss *SheetsService) ReadBudgetSheet() (*BudgetSheet, error) {
	range_ := fmt.Sprintf("%s!B%d:%s%d", ss.BudgetSheet.TabName, ss.BudgetSheet.SheetCoords.StartRow, ss.BudgetSheet.SheetCoords.EndColumnName, ss.BudgetSheet.SheetCoords.EndRow)
	resp, err := ss.Provider.GetValues(range_)
	if err != nil {
		return nil, fmt.Errorf("could not get sheet values: %s\n", err.Error())
	}
	if len(resp.Values) == 0 {
		return nil, errors.New("no values found")
	}

	var entries []*BudgetEntry
	var categoriesMap = make(map[string]*BudgetEntry)
	for _, values := range resp.Values {
		if ss.isEmptyBudgetRow(values) {
			continue
		}
		entry, err := ss.populateBudgetEntry(values)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
		categoriesMap[ss.getBudgetCategory(values)] = entry
	}
	ss.BudgetSheet.BudgetEntries = entries
	ss.BudgetSheet.CategoriesMap = categoriesMap
	return ss.BudgetSheet, nil
}

func (ss *SheetsService) isEmptyBudgetRow(values []interface{}) bool {
	if len(values) < 7 || ss.getBudgetCategory(values) == "" {
		return true
	}
	return false
}

func (ss *SheetsService) getBudgetCategory(values []interface{}) string {
	return fmt.Sprintf("%s", values[0])
}

func (ss *SheetsService) populateBudgetEntry(values []interface{}) (*BudgetEntry, error) {
	weekly, err := readDollarsValue(values[2])
	if err != nil {
		return nil, err
	}
	monthly, err := readDollarsValue(values[3])
	if err != nil {
		return nil, err
	}
	return &BudgetEntry{
		Category: ss.getBudgetCategory(values),
		Weekly:   weekly,
		Monthly:  monthly,
	}, nil
}

// --- Budget tab category surgery (Phase 6.5) -------------------------------
// The columns CLI keeps the Budget tab in step with the Register columns:
// the salary fan-out reads budget rows by name, so a register category with
// no budget row silently stops receiving allocations.

// budgetGroupName maps a register column color to the label of the Budget
// group whose totals row ends its block.
func budgetGroupName(color string) (string, bool) {
	switch color {
	case models.ColorGreen:
		return "Discretionary", true
	case models.ColorYellow:
		return "Non-Discretionary", true
	case models.ColorBlue:
		return "Savings Categories", true
	}
	return "", false
}

// BudgetRowFor returns the 1-based sheet row of the Budget category with
// the given name, or false if the tab has no such row. Reads fresh: the
// columns CLI is short-lived and the tab may have been edited by hand.
func (ss *SheetsService) BudgetRowFor(name string) (int, bool, error) {
	rng := fmt.Sprintf("%s!B%d:B%d", ss.BudgetSheet.TabName,
		ss.BudgetSheet.SheetCoords.StartRow, ss.BudgetSheet.SheetCoords.EndRow)
	resp, err := ss.Provider.GetValues(rng)
	if err != nil {
		return 0, false, fmt.Errorf("could not read budget categories: %w", err)
	}
	for i, row := range resp.Values {
		if len(row) > 0 && strings.TrimSpace(fmt.Sprintf("%s", row[0])) == name {
			return int(ss.BudgetSheet.SheetCoords.StartRow) + i, true, nil
		}
	}
	return 0, false, nil
}

// BudgetEntryFor returns the Weekly/Monthly allocations of the named
// Budget category.
func (ss *SheetsService) BudgetEntryFor(name string) (*BudgetEntry, bool, error) {
	row, found, err := ss.BudgetRowFor(name)
	if err != nil || !found {
		return nil, found, err
	}
	rng := fmt.Sprintf("%s!B%d:%s%d", ss.BudgetSheet.TabName, row,
		ss.BudgetSheet.SheetCoords.EndColumnName, row)
	resp, err := ss.Provider.GetValues(rng)
	if err != nil {
		return nil, false, fmt.Errorf("could not read budget row %d: %w", row, err)
	}
	if len(resp.Values) == 0 {
		return nil, false, nil
	}
	values := resp.Values[0]
	for len(values) < 7 { // short rows: trailing cells trimmed by the API
		values = append(values, "")
	}
	entry, err := ss.populateBudgetEntry(values)
	if err != nil {
		return nil, false, err
	}
	return entry, true, nil
}

// RenameBudgetCategory renames the Budget row of a category. Reports
// whether a row existed (payoff columns have none).
func (ss *SheetsService) RenameBudgetCategory(oldName, newName string) (bool, error) {
	row, found, err := ss.BudgetRowFor(oldName)
	if err != nil || !found {
		return found, err
	}
	rng := fmt.Sprintf("%s!B%d", ss.BudgetSheet.TabName, row)
	if _, err := ss.Provider.Update(rng, &sheets.ValueRange{Values: [][]interface{}{{newName}}}); err != nil {
		return false, fmt.Errorf("could not rename budget row %d: %w", row, err)
	}
	return true, nil
}

// InsertBudgetCategory adds a zeroed Budget row for a new register
// category: inside its color group, just before the group's totals row,
// with the totals formulas extended to cover the new row.
func (ss *SheetsService) InsertBudgetCategory(name, color string) error {
	group, ok := budgetGroupName(color)
	if !ok {
		return fmt.Errorf("no budget group for color %q", color)
	}
	totalsRow, found, err := ss.BudgetRowFor(group)
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("budget tab has no %q totals row to insert before", group)
	}

	insertReq := &sheets.BatchUpdateSpreadsheetRequest{
		Requests: []*sheets.Request{{
			InsertDimension: &sheets.InsertDimensionRequest{
				Range: &sheets.DimensionRange{
					SheetId:    ss.BudgetSheet.ID,
					Dimension:  "ROWS",
					StartIndex: int64(totalsRow - 1),
					EndIndex:   int64(totalsRow),
				},
				InheritFromBefore: true,
			},
		}},
	}
	if _, err := ss.Provider.BatchUpdate(insertReq); err != nil {
		return fmt.Errorf("could not insert budget row: %w", err)
	}

	// the new row sits at totalsRow; label it
	rng := fmt.Sprintf("%s!B%d", ss.BudgetSheet.TabName, totalsRow)
	if _, err := ss.Provider.Update(rng, &sheets.ValueRange{Values: [][]interface{}{{name}}}); err != nil {
		return fmt.Errorf("could not label budget row %d: %w", totalsRow, err)
	}

	// The totals row moved to totalsRow+1. Extend any of its SUM ranges
	// that ended at the group's old last row (totalsRow-1) to include
	// the new row — Sheets does not widen a range for a row inserted
	// immediately above its formula.
	newTotalsRow := totalsRow + 1
	frng := fmt.Sprintf("%s!B%d:H%d", ss.BudgetSheet.TabName, newTotalsRow, newTotalsRow)
	resp, err := ss.Provider.GetFormula(frng)
	if err != nil {
		return fmt.Errorf("could not read budget totals row: %w", err)
	}
	if len(resp.Values) == 0 {
		return nil
	}
	for j, cell := range resp.Values[0] {
		formula, ok := cell.(string)
		if !ok || !strings.HasPrefix(formula, "=") {
			continue
		}
		extended, changed := extendRangeEnd(formula, totalsRow-1, totalsRow)
		if !changed {
			continue
		}
		cellRef := fmt.Sprintf("%s!%s%d", ss.BudgetSheet.TabName,
			string(rune('B'+j)), newTotalsRow)
		if _, err := ss.Provider.Update(cellRef, &sheets.ValueRange{Values: [][]interface{}{{extended}}}); err != nil {
			return fmt.Errorf("could not extend budget totals formula at %s: %w", cellRef, err)
		}
	}
	return nil
}

// DeleteBudgetCategory removes the Budget row of a category. Reports
// whether a row existed. Deleting inside the group's SUM range is safe:
// Sheets shrinks the range itself.
func (ss *SheetsService) DeleteBudgetCategory(name string) (bool, error) {
	row, found, err := ss.BudgetRowFor(name)
	if err != nil || !found {
		return found, err
	}
	deleteReq := &sheets.BatchUpdateSpreadsheetRequest{
		Requests: []*sheets.Request{{
			DeleteDimension: &sheets.DeleteDimensionRequest{
				Range: &sheets.DimensionRange{
					SheetId:    ss.BudgetSheet.ID,
					Dimension:  "ROWS",
					StartIndex: int64(row - 1),
					EndIndex:   int64(row),
				},
			},
		}},
	}
	if _, err := ss.Provider.BatchUpdate(deleteReq); err != nil {
		return false, fmt.Errorf("could not delete budget row %d: %w", row, err)
	}
	return true, nil
}

// rangeRef matches a cell range like D3:D8 or $D$3:$D$8 inside a formula.
var rangeRef = regexp.MustCompile(`(\$?[A-Za-z]+\$?)(\d+):(\$?[A-Za-z]+\$?)(\d+)`)

// extendRangeEnd rewrites every range in a formula whose end row is
// oldEnd so it ends at newEnd instead — how a group totals formula grows
// to include a freshly inserted category row.
func extendRangeEnd(formula string, oldEnd, newEnd int) (string, bool) {
	changed := false
	out := rangeRef.ReplaceAllStringFunc(formula, func(m string) string {
		parts := rangeRef.FindStringSubmatch(m)
		end, err := strconv.Atoi(parts[4])
		if err != nil || end != oldEnd {
			return m
		}
		changed = true
		return parts[1] + parts[2] + ":" + parts[3] + strconv.Itoa(newEnd)
	})
	return out, changed
}
