package sheets_service

// The register write path: turning transactions into sheet rows.

import (
	"strings"
	"register/pkg/models"

	"google.golang.org/api/sheets/v4"
)


func (ss *SheetsService) UpdateRows(columns []models.Column, transNameToColName map[string]string, transactions []*models.Transaction) error {
	var requests []*sheets.Request

	rb := &RowBuilder{
		Columns:            columns,
		TransNameToColName: transNameToColName,
		CategoriesMap:      ss.BudgetSheet.CategoriesMap,
		IncomeSources:      ss.IncomeSources,
		FirstRowToUpdate:   ss.RegisterSheet.SheetCoords.FirstRowToUpdate,
		EndColumnIndex:     ss.RegisterSheet.SheetCoords.EndColumnIndex,
		ReadFormulas: func(rowIndex int64) ([]string, error) {
			return ss.readRangeFormulas(getRegisterToDeltaReadRange(rowIndex))
		},
	}
	rows, err := rb.populateCells(transactions)
	if err != nil {
		return err
	}

	gridCoordinate := ss.getGridCoordinate()
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

func (ss *SheetsService) getGridCoordinate() *sheets.GridCoordinate {
	return &sheets.GridCoordinate{
		SheetId:     ss.RegisterSheet.ID,
		RowIndex:    ss.RegisterSheet.SheetCoords.FirstRowToUpdate,
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
