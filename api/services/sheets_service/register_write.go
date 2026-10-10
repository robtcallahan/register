package sheets_service

// The register write path: turning transactions into sheet rows.

import (
	"strings"
	"register/pkg/models"

	"google.golang.org/api/sheets/v4"
)


func (ss *SheetsService) UpdateRows(columns []models.Column, transNameToColName map[string]string, transactions []*models.Transaction) error {
	var requests []*sheets.Request

	rows, err := ss.populateCells(columns, transNameToColName, transactions)
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

func (ss *SheetsService) populateCells(columns []models.Column, transNameToColName map[string]string, transactions []*models.Transaction) ([]*sheets.RowData, error) {
	var rows []*sheets.RowData

	rowIndex := ss.RegisterSheet.SheetCoords.FirstRowToUpdate
	for _, trans := range transactions {
		var cells []*sheets.CellData

		bgColor := getBackgroundColor(trans, ss.isIncomeName(trans.Name))
		cells, err := addSourceDateNameCells(cells, trans, bgColor)
		if err != nil {
			return nil, err
		}
		cells = addAmountCell(cells, trans, bgColor)

		totalsFormulas, err := ss.readRangeFormulas(getRegisterToDeltaReadRange(rowIndex))
		if err != nil {
			return nil, err
		}
		if ss.isIncomeName(trans.Name) {
			cells = ss.addSalaryCells(cells, columns, totalsFormulas, ss.incomeFrequency(trans.Name))
		} else {
			cells = addCategoryCells(cells, trans, columns, transNameToColName, totalsFormulas)
		}

		rows = append(rows, &sheets.RowData{Values: cells})

		if hasNote(trans) {
			rows = append(rows, ss.makeNoteRow(trans.Note))
		} else {
			var emptyCells []*sheets.CellData
			rows = append(rows, &sheets.RowData{
				Values: emptyCells,
			})
		}
		rowIndex += 2
	}
	return rows, nil
}

func (ss *SheetsService) makeNoteRow(note string) *sheets.RowData {
	var cells = make([]*sheets.CellData, 0, ss.RegisterSheet.SheetCoords.EndColumnIndex)
	for i := 1; i < 4; i++ {
		cells = append(cells, mkCellDataString("", "left", "lightgrey", false))
	}
	cells = append(cells, mkCellDataString(note, "left", "lightgrey", false))
	return &sheets.RowData{
		Values: cells,
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

func (ss *SheetsService) addSalaryCells(cells []*sheets.CellData, columns []models.Column, totalsFormulas []string, frequency string) []*sheets.CellData {
	// Same placement rule as addCategoryCells: cells are positioned by each
	// column's own ColumnIndex, so a gap in the columns table leaves an empty
	// cell instead of shifting the amounts after it.
	nextIndex := BankRegister + 1 // 1-based index of the next cell to fill (8 = H)
	for _, col := range columns {
		if col.ColumnIndex < nextIndex {
			continue // part of the fixed A-G block, already appended
		}
		for ; nextIndex < col.ColumnIndex; nextIndex++ {
			cells = append(cells, mkCellDataDollars(0.00, "left", "", true))
		}
		// i is this column's offset within the category block (0 = H)
		i := col.ColumnIndex - BankRegister - 1
		entry := ss.BudgetSheet.CategoriesMap[col.Name]

		if isRegisterClearedOrDeltaColumn(i) {
			// first 3 columns are Register, Cleared & Delta. We copied the cell formulas above and are pasting here
			cells = append(cells, mkCellDataFormula(totalsFormulas[i], "right", col.Color, false))
		} else if isBudgetColumn(col.Name) && entry != nil {
			// enter the budgeted amount for this cadence in this category column
			cells = append(cells, mkCellDataDollars(allocationFor(entry, frequency), "left", col.Color, true))
		} else {
			// this cell doesn't apply. Just create an empty (opaque) cell.
			cells = append(cells, mkCellDataDollars(0.00, "left", col.Color, true))
		}
		nextIndex++
	}
	return cells
}
