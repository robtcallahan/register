package sheets_service

import (
	"register/pkg/config"
	"register/pkg/models"

	"google.golang.org/api/sheets/v4"
)

// RowBuilder assembles sheet rows from transactions. It holds no API client:
// everything it needs is data, plus the injected ReadFormulas for the one
// sheet read row-building requires (the totals formulas copied into the
// Register/Cleared/Delta cells).
type RowBuilder struct {
	Columns            []models.Column
	TransNameToColName map[string]string
	CategoriesMap      map[string]*BudgetEntry
	IncomeSources      []config.IncomeSource
	FirstRowToUpdate   int64
	EndColumnIndex     int64
	ReadFormulas       func(rowIndex int64) ([]string, error)
}

func (rb *RowBuilder) populateCells(transactions []*models.Transaction) ([]*sheets.RowData, error) {
	var rows []*sheets.RowData

	rowIndex := rb.FirstRowToUpdate
	for _, trans := range transactions {
		var cells []*sheets.CellData

		bgColor := getBackgroundColor(trans, isIncomeName(trans.Name, rb.IncomeSources))
		cells, err := addSourceDateNameCells(cells, trans, bgColor)
		if err != nil {
			return nil, err
		}
		cells = addAmountCell(cells, trans, bgColor)

		totalsFormulas, err := rb.ReadFormulas(rowIndex)
		if err != nil {
			return nil, err
		}
		if isIncomeName(trans.Name, rb.IncomeSources) {
			cells = rb.addSalaryCells(cells, totalsFormulas, incomeFrequency(trans.Name, rb.IncomeSources))
		} else {
			cells = addCategoryCells(cells, trans, rb.Columns, rb.TransNameToColName, totalsFormulas)
		}

		rows = append(rows, &sheets.RowData{Values: cells})

		if hasNote(trans) {
			rows = append(rows, rb.makeNoteRow(trans.Note))
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

func (rb *RowBuilder) makeNoteRow(note string) *sheets.RowData {
	var cells = make([]*sheets.CellData, 0, rb.EndColumnIndex)
	for i := 1; i < 4; i++ {
		cells = append(cells, mkCellDataString("", "left", "lightgrey", false))
	}
	cells = append(cells, mkCellDataString(note, "left", "lightgrey", false))
	return &sheets.RowData{
		Values: cells,
	}
}

func (rb *RowBuilder) addSalaryCells(cells []*sheets.CellData, totalsFormulas []string, frequency string) []*sheets.CellData {
	// Same placement rule as addCategoryCells: cells are positioned by each
	// column's own ColumnIndex, so a gap in the columns table leaves an empty
	// cell instead of shifting the amounts after it.
	nextIndex := BankRegister + 1 // 1-based index of the next cell to fill (8 = H)
	for _, col := range rb.Columns {
		if col.ColumnIndex < nextIndex {
			continue // part of the fixed A-G block, already appended
		}
		for ; nextIndex < col.ColumnIndex; nextIndex++ {
			cells = append(cells, mkCellDataDollars(0.00, "left", "", true))
		}
		// i is this column's offset within the category block (0 = H)
		i := col.ColumnIndex - BankRegister - 1
		entry := rb.CategoriesMap[col.Name]

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

// isIncomeName reports whether a register row's display name is one of the
// configured income sources.
func isIncomeName(name string, sources []config.IncomeSource) bool {
	for _, src := range sources {
		if src.Name == name {
			return true
		}
	}
	return false
}

func incomeFrequency(name string, sources []config.IncomeSource) string {
	for _, src := range sources {
		if src.Name == name {
			return src.Frequency
		}
	}
	return ""
}
