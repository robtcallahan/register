package sheets_service

// The register write path: turning transactions into sheet rows.

import (
	"fmt"
	"register/pkg/models"
	"strings"

	"google.golang.org/api/sheets/v4"
)

func (ss *SheetsService) UpdateRows(registerSheet RegisterSheet, budgetSheet BudgetSheet, columns []models.Column, transNameToColName map[string]string, transactions []*models.Transaction) error {
	var requests []*sheets.Request

	rb := &RowBuilder{
		Columns:            columns,
		TransNameToColName: transNameToColName,
		CategoriesMap:      budgetSheet.CategoriesMap,
		IncomeSources:      ss.IncomeSources,
		FirstRowToUpdate:   registerSheet.SheetCoords.FirstRowToUpdate,
		EndColumnIndex:     registerSheet.SheetCoords.EndColumnIndex,
		ReadFormulas: func(rowIndex int64) ([]string, error) {
			return ss.readRangeFormulas(getRegisterToDeltaReadRange(rowIndex))
		},
	}
	rows, err := rb.populateCells(transactions)
	if err != nil {
		return err
	}

	gridCoordinate := ss.getGridCoordinate(registerSheet)
	updateCellsRequest := sheets.UpdateCellsRequest{
		Fields: "*",
		Rows:   rows,
		Start:  gridCoordinate,
	}

	request := sheets.Request{
		UpdateCells: &updateCellsRequest,
	}
	requests = append(requests, &request)

	batchUpdateRequest := sheets.BatchUpdateSpreadsheetRequest{
		Requests: requests,
	}

	_, err = ss.Provider.BatchUpdate(&batchUpdateRequest)
	if err != nil {
		return err
	}
	return nil
}

// Private methods

func (ss *SheetsService) getGridCoordinate(registerSheet RegisterSheet) *sheets.GridCoordinate {
	return &sheets.GridCoordinate{
		SheetId:     registerSheet.ID,
		RowIndex:    registerSheet.SheetCoords.FirstRowToUpdate,
		ColumnIndex: 0,
	}
}

// allocationFor returns a category's budgeted amount at the income source's
// cadence: "weekly" deposits allocate the Weekly amounts, "monthly" the
// Monthly ones. Any other frequency allocates nothing (visible zeros), so a
// misconfigured frequency is obvious rather than quietly wrong.
func allocationFor(entry *BudgetEntry, frequency string) float64 {
	if entry == nil {
		return 0
	}
	switch strings.ToLower(frequency) {
	case "weekly":
		return entry.Weekly
	case "biweekly":
		return entry.BiWeekly
	case "monthly":
		return entry.Monthly
	}
	return 0
}

func addDeposit(amount float64, bgColor string, cells []*sheets.CellData) []*sheets.CellData {
	cells = append(cells, mkCellDataString("", "left", bgColor, false))
	cells = append(cells, mkCellDataDollars(amount, "right", bgColor, false))
	return cells
}

func addWithdrawal(amount float64, bgColor string, cells []*sheets.CellData) []*sheets.CellData {
	cells = append(cells, mkCellDataDollars(amount, "right", bgColor, false))
	cells = append(cells, mkCellDataString("", "left", bgColor, false))
	return cells
}

func addCheckingTransaction(trans *models.Transaction, bgColor string, cells []*sheets.CellData) []*sheets.CellData {
	if trans.Deposit > 0 {
		cells = addDeposit(trans.Deposit, bgColor, cells)
	} else {
		cells = addWithdrawal(trans.Withdrawal, bgColor, cells)
	}
	// add credit card cell
	cells = append(cells, mkCellDataString("", "left", bgColor, false))
	return cells
}

func addCCTransaction(trans *models.Transaction, bgColor string, cells []*sheets.CellData) []*sheets.CellData {
	cells = append(cells, mkCellDataString("", "left", bgColor, false))
	cells = append(cells, mkCellDataString("", "left", bgColor, false))
	cells = append(cells, mkCellDataDollars(trans.CreditPurchase, "right", bgColor, false))
	return cells
}

func addSourceDateNameCells(cells []*sheets.CellData, trans *models.Transaction, bgColor string) ([]*sheets.CellData, error) {
	cells = append(cells, getCellDataReconcileColumn("X", "center", bgColor, false))
	cells = append(cells, mkCellDataString(trans.Source, "center", bgColor, false))
	dateCell, err := getCellDataDate(trans.Date, "center", bgColor, false)
	if err != nil {
		return nil, err
	}
	cells = append(cells, dateCell)
	cells = append(cells, mkCellDataString(trans.Name, "left", bgColor, false))
	return cells, nil
}

func addAmountCell(cells []*sheets.CellData, trans *models.Transaction, bgColor string) []*sheets.CellData {
	if isCheckingAccount(trans) {
		cells = addCheckingTransaction(trans, bgColor, cells)
	} else {
		cells = addCCTransaction(trans, bgColor, cells)
	}
	return cells
}

func addCategoryCells(cells []*sheets.CellData, trans *models.Transaction, columns []models.Column, transNameToColName map[string]string, totalsFormulas []string) []*sheets.CellData {
	// Cells for cols A-G (DB indexes 1-7) are already appended. Category cells
	// are placed by each column's own ColumnIndex, not by slice position, so a
	// gap in the columns table leaves an empty cell instead of shifting every
	// amount after it one column left.
	nextIndex := BankRegister + 1 // 1-based index of the next cell to fill (8 = H)
	for _, col := range columns {
		if col.ColumnIndex < nextIndex {
			continue // part of the fixed A-G block, already appended
		}
		for ; nextIndex < col.ColumnIndex; nextIndex++ {
			cells = append(cells, mkCellDataEmpty("left", "", true))
		}
		// i is this column's offset within the category block (0 = H)
		i := col.ColumnIndex - BankRegister - 1
		if isRegisterClearedOrDeltaColumn(i) {
			// first 3 columns are Register, Cleared & Delta. We copied the cell formulas above and are pasting here
			cells = append(cells, mkCellDataFormula(totalsFormulas[i], "right", col.Color, false))
		} else if isCreditCardTransaction(trans.Source, col.Name) {
			// enter a positive value in the credit card column
			cells = append(cells, mkCellDataDollars(trans.CreditCard, "left", models.ColorGrey, true))
		} else if trans.ColumnIndex != 0 && trans.ColumnIndex == col.ColumnIndex {
			cells = append(cells, mkCellDataDollars(trans.Budget, "left", col.Color, true))
		} else if isCorrectBudgetColumn(trans.Name, col.Name, transNameToColName) {
			// enter the value in the budget category column
			cells = append(cells, mkCellDataDollars(trans.Budget, "left", col.Color, true))
		} else {
			// this cell doesn't apply. Just create an empty cell.
			cells = append(cells, mkCellDataEmpty("left", col.Color, true))
		}
		nextIndex++
	}
	return cells
}

// CategoryMove relocates one row's category amount to another category column.
type CategoryMove struct {
	Key       string  // transaction key, for the companion DB update
	RowID     int64   // 1-based sheet row of the transaction's data row
	FromIndex int     // 1-based column index being vacated
	ToIndex   int     // 1-based column index receiving the amount
	Value     float64 // the amount to move, as read from the sheet
	Color     string  // background color for the receiving cell
}

// PlanCategoryMoves scans the register sheet for data rows whose description
// contains any of the match strings (case-insensitive) and plans moving each
// row's category amount into the target column. Rows with no (or ambiguous)
// category amounts are reported in notes and left alone.
func (ss *SheetsService) PlanCategoryMoves(registerSheet RegisterSheet, matches []string, target models.Column, cols []models.Column) ([]CategoryMove, []string) {
	var moves []CategoryMove
	var notes []string

	lowered := make([]string, len(matches))
	for i, m := range matches {
		lowered[i] = strings.ToLower(m)
	}

	for _, e := range registerSheet.Register {
		matched := false
		name := strings.ToLower(e.Name)
		for _, m := range lowered {
			if strings.Contains(name, m) {
				matched = true
				break
			}
		}
		if !matched {
			continue
		}

		// RowID = StartRow + offset, so the offset indexes RangeValues
		// directly: the two-rows-per-entry stepping is already in RowID.
		rowVals := getRowValues(registerSheet.RangeValues, int(e.RowID-registerSheet.SheetCoords.StartRow))

		var fromIndex int
		var value float64
		hits := 0
		for _, col := range cols {
			// categories only, and not the black payoff columns: those
			// accumulate card balances, not spend (mirrors Aggregate)
			if !col.IsCategory() || col.Color == models.ColorGrey {
				continue
			}
			v, err := getDollarsCellByIndex(rowVals, col.ColumnIndex-1)
			if err != nil {
				continue
			}
			if v != 0 {
				hits++
				fromIndex = col.ColumnIndex
				value = v
			}
		}

		desc := fmt.Sprintf("row %d (%s %s)", e.RowID, e.Date, e.Name)
		switch {
		case hits == 0:
			notes = append(notes, desc+": no category amount found, skipping")
		case hits > 1:
			notes = append(notes, desc+": multiple category amounts, skipping")
		case fromIndex == target.ColumnIndex:
			notes = append(notes, desc+": already in "+target.Name+", skipping")
		default:
			moves = append(moves, CategoryMove{
				Key:       e.Key,
				RowID:     e.RowID,
				FromIndex: fromIndex,
				ToIndex:   target.ColumnIndex,
				Value:     value,
				Color:     target.Color,
			})
		}
	}
	return moves, notes
}

// MoveCategoryCells executes planned category moves in a single batch update:
// each amount is written into its new column and cleared from the old one.
func (ss *SheetsService) MoveCategoryCells(registerSheet RegisterSheet, moves []CategoryMove) error {
	var requests []*sheets.Request
	for _, m := range moves {
		write := &sheets.UpdateCellsRequest{
			Fields: "*",
			Start: &sheets.GridCoordinate{
				SheetId:     registerSheet.ID,
				RowIndex:    m.RowID - 1,
				ColumnIndex: int64(m.ToIndex - 1),
			},
			Rows: []*sheets.RowData{
				{Values: []*sheets.CellData{mkCellDataDollars(m.Value, "right", m.Color, false)}},
			},
		}
		requests = append(requests, &sheets.Request{UpdateCells: write})

		clear := &sheets.UpdateCellsRequest{
			Fields: "*",
			Start: &sheets.GridCoordinate{
				SheetId:     registerSheet.ID,
				RowIndex:    m.RowID - 1,
				ColumnIndex: int64(m.FromIndex - 1),
			},
			Rows: []*sheets.RowData{
				{Values: []*sheets.CellData{{}}},
			},
		}
		requests = append(requests, &sheets.Request{UpdateCells: clear})
	}

	_, err := ss.Provider.BatchUpdate(&sheets.BatchUpdateSpreadsheetRequest{Requests: requests})
	if err != nil {
		return fmt.Errorf("clear register rows: %w", err)
	}
	return nil
}
