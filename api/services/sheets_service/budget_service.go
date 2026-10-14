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
	BiWeekly float64
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
			StartRow: cfg.BudgetStartRow,
			EndRow:   cfg.BudgetEndRow,
			// EndColumnName/Index are detected from the header row below.
		},
	}
	props, err := ss.getSheetProperties(BudgetTabName)
	if err != nil {
		return err
	}
	ss.BudgetSheet.ID = props.SheetId

	// Detect the rightmost column from the header row so adding columns
	// to the Budget tab doesn't require a code change. We read a generous
	// range (A1:Z1) and find the last non-empty header.
	headerRange := fmt.Sprintf("%s!A1:Z1", BudgetTabName)
	headerResp, err := ss.Provider.GetValues(headerRange)
	if err != nil {
		return fmt.Errorf("could not read budget header row for width detection: %w", err)
	}
	if len(headerResp.Values) == 0 || len(headerResp.Values[0]) == 0 {
		return errors.New("no header row found on Budget tab")
	}
	headers := headerResp.Values[0]
	lastIdx := 0
	for i, h := range headers {
		if s := fmt.Sprintf("%v", h); strings.TrimSpace(s) != "" {
			lastIdx = i
		}
	}
	ss.BudgetSheet.SheetCoords.EndColumnIndex = int64(lastIdx)
	ss.BudgetSheet.SheetCoords.EndColumnName = models.ColumnLetter(lastIdx)
	return nil
}

func (ss *SheetsService) ReadBudgetSheet() (*BudgetSheet, error) {
	range_ := fmt.Sprintf("%s!A%d:%s%d", ss.BudgetSheet.TabName, ss.BudgetSheet.SheetCoords.StartRow, ss.BudgetSheet.SheetCoords.EndColumnName, ss.BudgetSheet.SheetCoords.EndRow)
	resp, err := ss.Provider.GetValues(range_)
	if err != nil {
		return nil, fmt.Errorf("could not get sheet values: %w\n", err)
	}
	if len(resp.Values) == 0 {
		return nil, errors.New("no values found")
	}

	decoder, err := ss.budgetHeaderDecoder()
	if err != nil {
		return nil, err
	}
	// Fail fast: the columns we read must exist under their expected headers.
	if _, err := decoder.require("Weekly"); err != nil {
		return nil, err
	}
	if _, err := decoder.require("Bi-Weekly"); err != nil {
		return nil, err
	}
	if _, err := decoder.require("Monthly"); err != nil {
		return nil, err
	}

	var entries []*BudgetEntry
	var categoriesMap = make(map[string]*BudgetEntry)
	for _, values := range resp.Values {
		if ss.isEmptyBudgetRow(values) {
			continue
		}
		entry, err := ss.populateBudgetEntry(decoder, values)
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

// BudgetCheckResult is a single line of the budget layout check report.
type BudgetCheckResult struct {
	Ok      bool
	Message string
}

// CheckBudgetLayout validates the Budget tab headers and returns a
// human-readable report. Unlike ReadBudgetSheet (which fails on the first
// problem), this collects all issues so they can be fixed in one pass.
func (ss *SheetsService) CheckBudgetLayout() ([]BudgetCheckResult, error) {
	var results []BudgetCheckResult

	decoder, err := ss.budgetHeaderDecoder()
	if err != nil {
		return nil, fmt.Errorf("could not read budget headers: %w", err)
	}

	// Required headers for the current code.
	required := []string{"Bi-Weekly", "Monthly"}
	for _, h := range required {
		if _, err := decoder.require(h); err != nil {
			results = append(results, BudgetCheckResult{
				Ok:      false,
				Message: fmt.Sprintf("Header %q not found \u2014 did you rename it?", h),
			})
		} else {
			results = append(results, BudgetCheckResult{
				Ok:      true,
				Message: fmt.Sprintf("Header %q found", h),
			})
		}
	}

	return results, nil
}

func (ss *SheetsService) isEmptyBudgetRow(values []interface{}) bool {
	return getStringField(values, 0) == ""
}

func (ss *SheetsService) getBudgetCategory(values []interface{}) string {
	return fmt.Sprintf("%s", values[0])
}

func (ss *SheetsService) populateBudgetEntry(decoder *rowDecoder, values []interface{}) (*BudgetEntry, error) {
	// The category column is A-anchored (index 0); Weekly/Monthly come from
	// the header row, so a moved or renamed column is a loud error.
	weekly, err := readDollarsValue(decoder.cell(values, "Weekly"))
	if err != nil {
		return nil, err
	}
	biweekly, err := readDollarsValue(decoder.cell(values, "Bi-Weekly"))
	if err != nil {
		return nil, err
	}
	monthly, err := readDollarsValue(decoder.cell(values, "Monthly"))
	if err != nil {
		return nil, err
	}
	return &BudgetEntry{
		Category: ss.getBudgetCategory(values),
		Weekly:   weekly,
		BiWeekly: biweekly,
		Monthly:  monthly,
	}, nil
}

// budgetHeaderDecoder reads the Budget tab header row (row 1) and builds a
// name→column decoder from it.
func (ss *SheetsService) budgetHeaderDecoder() (*rowDecoder, error) {
	headerRange := fmt.Sprintf("%s!A1:%s1", ss.BudgetSheet.TabName, ss.BudgetSheet.SheetCoords.EndColumnName)
	headerResp, err := ss.Provider.GetValues(headerRange)
	if err != nil {
		return nil, fmt.Errorf("could not read budget header row: %w", err)
	}
	if len(headerResp.Values) == 0 {
		return nil, errors.New("no header row found on Budget tab")
	}
	return newRowDecoder(headerResp.Values[0])
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
	rng := fmt.Sprintf("%s!A%d:A%d", ss.BudgetSheet.TabName,
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
	rng := fmt.Sprintf("%s!A%d:%s%d", ss.BudgetSheet.TabName, row,
		ss.BudgetSheet.SheetCoords.EndColumnName, row)
	resp, err := ss.Provider.GetValues(rng)
	if err != nil {
		return nil, false, fmt.Errorf("could not read budget row %d: %w", row, err)
	}
	if len(resp.Values) == 0 {
		return nil, false, nil
	}
	decoder, err := ss.budgetHeaderDecoder()
	if err != nil {
		return nil, false, err
	}
	entry, err := ss.populateBudgetEntry(decoder, resp.Values[0])
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
	rng := fmt.Sprintf("%s!A%d", ss.BudgetSheet.TabName, row)
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
	rng := fmt.Sprintf("%s!A%d", ss.BudgetSheet.TabName, totalsRow)
	if _, err := ss.Provider.Update(rng, &sheets.ValueRange{Values: [][]interface{}{{name}}}); err != nil {
		return fmt.Errorf("could not label budget row %d: %w", totalsRow, err)
	}

	// The totals row moved to totalsRow+1. Extend any of its SUM ranges
	// that ended at the group's old last row (totalsRow-1) to include
	// the new row — Sheets does not widen a range for a row inserted
	// immediately above its formula.
	newTotalsRow := totalsRow + 1
	frng := fmt.Sprintf("%s!A%d:G%d", ss.BudgetSheet.TabName, newTotalsRow, newTotalsRow)
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
			string(rune('A'+j)), newTotalsRow)
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

// MoveBudgetCategoryAfter moves the Budget row of name so it sits
// immediately after the Budget row of afterName. The caller is
// responsible for keeping the row inside its color group.
func (ss *SheetsService) MoveBudgetCategoryAfter(name, afterName string) error {
	row, found, err := ss.BudgetRowFor(name)
	if err != nil || !found {
		if !found {
			return fmt.Errorf("no budget row for %q", name)
		}
		return err
	}
	afterRow, found, err := ss.BudgetRowFor(afterName)
	if err != nil || !found {
		if !found {
			return fmt.Errorf("no budget row for %q", afterName)
		}
		return err
	}
	if row == afterRow+1 {
		return nil // already there
	}
	return ss.moveBudgetRow(row, afterRow)
}

// MoveBudgetCategoryBefore moves the Budget row of name so it sits
// immediately above the Budget row of beforeName.
func (ss *SheetsService) MoveBudgetCategoryBefore(name, beforeName string) error {
	row, found, err := ss.BudgetRowFor(name)
	if err != nil || !found {
		if !found {
			return fmt.Errorf("no budget row for %q", name)
		}
		return err
	}
	beforeRow, found, err := ss.BudgetRowFor(beforeName)
	if err != nil || !found {
		if !found {
			return fmt.Errorf("no budget row for %q", beforeName)
		}
		return err
	}
	if row == beforeRow-1 {
		return nil // already there
	}
	return ss.moveBudgetRow(row, beforeRow-1)
}

// moveBudgetRow relocates the 1-based Budget row to the given
// MoveDimension destination (pre-removal coordinates).
func (ss *SheetsService) moveBudgetRow(row, destinationIndex int) error {
	moveReq := &sheets.BatchUpdateSpreadsheetRequest{
		Requests: []*sheets.Request{{
			MoveDimension: &sheets.MoveDimensionRequest{
				Source: &sheets.DimensionRange{
					SheetId:    ss.BudgetSheet.ID,
					Dimension:  "ROWS",
					StartIndex: int64(row - 1),
					EndIndex:   int64(row),
				},
				DestinationIndex: int64(destinationIndex),
			},
		}},
	}
	if _, err := ss.Provider.BatchUpdate(moveReq); err != nil {
		return fmt.Errorf("could not move budget row %d: %w", row, err)
	}
	return nil
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
