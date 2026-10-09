package sheets_service

import (
	"errors"
	"fmt"
	"regexp"
	"register/pkg/config"
	"register/pkg/models"
	"strings"

	"google.golang.org/api/sheets/v4"
)

type SheetCoords struct {
	StartRow         int64
	EndRow           int64
	LastRow          int64
	RowCount         int64
	ColumnCount      int64
	FirstRowToUpdate int64
	EndColumnName    string
	EndColumnIndex   int64
}

type RegisterEntry struct {
	RowID        int64
	IsCheck      bool
	Key          string
	Reconciled   string
	Source       string
	Date         string
	Name         string
	Amount       float64
	Withdrawal   float64
	Deposit      float64
	CreditCard   float64
	BankRegister float64
	Cleared      float64
	Delta        float64
}

type RegisterSheet struct {
	ID          int64
	TabName     string
	Spreadsheet sheets.Spreadsheet
	SheetCoords SheetCoords
	Register    []*RegisterEntry
	KeysMap     map[string]bool
	RangeValues [][]interface{}
}

// Public methods

func (ss *SheetsService) NewRegisterSheet(cfg *config.Config, columns []models.Column) error {
	// The register's last column comes from the columns table (its
	// highest-indexed column). The config values remain as a fallback for
	// callers that don't read the DB (register copy) until 6.6 retires them.
	endColumnName := cfg.RegisterCategoryEndColumn
	endColumnIndex := cfg.ColumnIndexes[cfg.RegisterCategoryEndColumn]
	if len(columns) > 0 {
		set := models.NewColumnSet(columns)
		if end, ok := set.End(); ok {
			endColumnName = set.LetterFor(end.ColumnIndex)
			endColumnIndex = int64(set.SliceOffset(end.ColumnIndex))
		}
	}
	ss.RegisterSheet = &RegisterSheet{
		TabName: "Register",
		SheetCoords: SheetCoords{
			StartRow:       cfg.RegisterStartRow,
			EndRow:         cfg.RegisterEndRow,
			EndColumnName:  endColumnName,
			EndColumnIndex: endColumnIndex,
		},
	}

	props, err := ss.getSheetProperties("Register")
	if err != nil {
		return err
	}
	ss.RegisterSheet.ID = props.SheetId
	if props.GridProperties != nil {
		ss.RegisterSheet.SheetCoords.RowCount = props.GridProperties.RowCount
		ss.RegisterSheet.SheetCoords.ColumnCount = props.GridProperties.ColumnCount
	}
	return nil
}

func (ss *SheetsService) ReadRegisterSheet() (*RegisterSheet, error) {
	var register []*RegisterEntry
	var i int64

	var readRange = ss.getReadRange()
	resp, err := ss.Provider.GetValues(readRange)
	if err != nil {
		return nil, fmt.Errorf("could not get sheet values: %s\n", err.Error())
	}
	if len(resp.Values) == 0 {
		return nil, fmt.Errorf("no data found for read range: %s", ss.getReadRange())
	}

	// determine last used row in the spreadsheet
	ss.RegisterSheet.SheetCoords.LastRow = ss.getLastRow(resp.Values)
	keysMap := make(map[string]bool)

	for i = 0; i <= ss.RegisterSheet.SheetCoords.LastRow; i += 2 {
		row := getRowValues(resp.Values, int(i))
		if ss.isEmptyRow(row) {
			break
		}
		transactionKey := getTransactionKey(row)
		keysMap[transactionKey] = true
		registerEntry, err := ss.populateRegisterEntry(row)
		if err != nil {
			return nil, err
		}
		registerEntry.RowID = ss.getRowID(i)
		register = append(register, registerEntry)
	}
	ss.RegisterSheet.SheetCoords.FirstRowToUpdate = ss.getFirstRowToUpdate(i)
	ss.RegisterSheet.Register = register
	ss.RegisterSheet.KeysMap = keysMap
	ss.RegisterSheet.RangeValues = resp.Values

	return ss.RegisterSheet, nil
}

func (ss *SheetsService) ReadCell(cell string, cellDataType CellDataType) (interface{}, error) {
	var resp *sheets.ValueRange
	var err error

	readRange := fmt.Sprintf("%s!%s:%s", ss.RegisterSheet.TabName, cell, cell)

	if cellDataType == CellDataFormula {
		resp, err = ss.Provider.GetFormula(readRange)
	} else {
		resp, err = ss.Provider.GetValues(readRange)
	}
	if err != nil {
		return nil, err
	}
	row := getRowValues(resp.Values, 0)
	if len(row) == 0 {
		return nil, fmt.Errorf("no data found for cell: %s", cell)
	}
	return row[0], nil
}

// errSheetFull means the copy would run past the end of the sheet's grid.
var errSheetFull = errors.New("register sheet is full")

func (ss *SheetsService) CopyRows(numCopies int) error {
	if err := ss.checkRowsFit(numCopies); err != nil {
		return err
	}
	updateReq := ss.copyRowsBatchUpdateRequest(numCopies)
	_, err := ss.Provider.BatchUpdate(&updateReq)
	if err != nil {
		if strings.Contains(err.Error(), "grid limits") {
			return fmt.Errorf("%w — add rows and re-run", errSheetFull)
		}
		return fmt.Errorf("could not perform copy: %v", err)
	}
	return nil
}

func (ss *SheetsService) checkRowsFit(numCopies int) error {
	coords := ss.RegisterSheet.SheetCoords
	if coords.RowCount == 0 {
		return nil // grid size unknown; let the API be the judge
	}
	lastNeeded := coords.LastRow + 2*int64(numCopies)
	if lastNeeded <= coords.RowCount {
		return nil
	}
	shortfall := lastNeeded - coords.RowCount
	add := (shortfall + 999) / 1000 * 1000
	return fmt.Errorf("%w — add %d rows and re-run (copy needs row %d, sheet has %d rows)", errSheetFull, add, lastNeeded, coords.RowCount)
}

func (ss *SheetsService) ReadStringCell(cell string) (string, error) {
	v, err := ss.ReadCell(cell, CellDataString)
	if err != nil {
		return "", err
	}
	return readStringValue(v), nil
}

func (ss *SheetsService) ReadDollarsCell(cell string) (float64, error) {
	v, err := ss.ReadCell(cell, CellDataDollars)
	if err != nil {
		return 0, err
	}
	return readDollarsValue(v)
}

func (ss *SheetsService) ReadFormulaCell(cell string) (string, error) {
	v, err := ss.ReadCell(cell, CellDataFormula)
	if err != nil {
		return "", err
	}
	return readStringValue(v), nil
}

func (ss *SheetsService) ReadDateCell(cell string) (string, error) {
	v, err := ss.ReadCell(cell, CellDataDate)
	if err != nil {
		return "", err
	}
	return readDateValue(v), nil
}

func (ss *SheetsService) WriteCell(cell string, value interface{}) (*sheets.UpdateValuesResponse, error) {
	var v [][]interface{}
	v = append(v, []interface{}{value})

	writeRange := fmt.Sprintf("%s!%s:%s", ss.RegisterSheet.TabName, cell, cell)
	vRange := &sheets.ValueRange{
		Values: v,
	}
	resp, err := ss.Provider.Update(writeRange, vRange)
	if err != nil {
		return nil, fmt.Errorf("unable to write cell data: %s", err.Error())
	}
	return resp, nil
}

// firstCategoryColumnIndex is the first 1-based position past the fixed
// A-J register block (K) — where category columns begin.
const firstCategoryColumnIndex = 11

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
		if col.ColumnIndex < firstCategoryColumnIndex {
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

func (ss *SheetsService) isEmptyRow(values []interface{}) bool {
	if getStringField(values, Date) == "" {
		return true
	}
	return false
}

func (ss *SheetsService) getFirstRowToUpdate(i int64) int64 {
	return ss.RegisterSheet.SheetCoords.StartRow + i - 1
}

func (ss *SheetsService) getLastRow(values [][]interface{}) int64 {
	return int64(len(values)) + ss.RegisterSheet.SheetCoords.StartRow - 2
}

func (ss *SheetsService) getReadRange() string {
	return fmt.Sprintf("%s!A%d:%s%d", ss.RegisterSheet.TabName, ss.RegisterSheet.SheetCoords.StartRow,
		ss.RegisterSheet.SheetCoords.EndColumnName, ss.RegisterSheet.SheetCoords.EndRow)
}

func (ss *SheetsService) getRowID(i int64) int64 {
	return ss.RegisterSheet.SheetCoords.StartRow + i
}

func (ss *SheetsService) populateRegisterEntry(values []interface{}) (*RegisterEntry, error) {
	withdrawal, err := getDollarsCellByIndex(values, Withdrawals)
	if err != nil {
		return nil, err
	}
	deposit, err := getDollarsCellByIndex(values, Deposits)
	if err != nil {
		return nil, err
	}
	creditCard, err := getDollarsCellByIndex(values, CreditCards)
	if err != nil {
		return nil, err
	}
	bankRegister, err := getDollarsCellByIndex(values, BankRegister)
	if err != nil {
		return nil, err
	}
	cleared, err := getDollarsCellByIndex(values, Cleared)
	if err != nil {
		return nil, err
	}
	delta, err := getDollarsCellByIndex(values, Delta)
	if err != nil {
		return nil, err
	}
	entry := &RegisterEntry{
		Key:          getTransactionKey(values),
		Reconciled:   getStringField(values, Reconciled),
		Source:       getSourceField(values),
		Date:         getDateField(values),
		Name:         getNameField(values),
		Withdrawal:   withdrawal,
		Deposit:      deposit,
		CreditCard:   creditCard,
		BankRegister: bankRegister,
		Cleared:      cleared,
		Delta:        delta,
	}
	return entry, nil
}

func (ss *SheetsService) getSheetProperties(tabName string) (*sheets.SheetProperties, error) {
	spreadsheet, err := ss.Provider.GetSpreadsheet()
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve spreadsheet: %w", err)
	}

	for _, sheet := range spreadsheet.Sheets {
		if sheet.Properties.Title == tabName {
			return sheet.Properties, nil
		}
	}
	return nil, fmt.Errorf("could not find sheet: %s", tabName)
}

func (ss *SheetsService) copyRowsBatchUpdateRequest(numCopies int) sheets.BatchUpdateSpreadsheetRequest {
	var requests []*sheets.Request

	index := ss.RegisterSheet.SheetCoords.LastRow
	for i := 1; i <= numCopies; i++ {
		copyPasteRequest := sheets.CopyPasteRequest{
			Source:      ss.getCopySource(index),
			Destination: ss.getCopyDestination(index),
			PasteType:   "PASTE_NORMAL",
		}
		request := sheets.Request{
			CopyPaste: &copyPasteRequest,
		}
		requests = append(requests, &request)
		index += 2
	}

	updateReq := sheets.BatchUpdateSpreadsheetRequest{
		Requests: requests,
	}
	return updateReq
}

func (ss *SheetsService) getCopySource(index int64) *sheets.GridRange {
	return &sheets.GridRange{
		SheetId:          ss.RegisterSheet.ID,
		StartColumnIndex: 0,
		EndColumnIndex:   ss.RegisterSheet.SheetCoords.EndColumnIndex + 1,
		StartRowIndex:    index - 1,
		EndRowIndex:      index + 1,
	}
}

func (ss *SheetsService) getCopyDestination(index int64) *sheets.GridRange {
	return &sheets.GridRange{
		SheetId:          ss.RegisterSheet.ID,
		StartColumnIndex: 0,
		EndColumnIndex:   ss.RegisterSheet.SheetCoords.EndColumnIndex + 1,
		StartRowIndex:    index + 1,
		EndRowIndex:      index + 1,
	}
}

func (ss *SheetsService) readRangeFormulas(readRange string) ([]string, error) {
	resp, err := ss.Provider.GetFormula(readRange)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve data from sheet: %v", err)
	}
	rangeValues := resp.Values
	if len(rangeValues) == 0 {
		return nil, fmt.Errorf("no data found for read range: %s", readRange)
	}

	var retValues []string
	for _, val := range getRowValues(rangeValues, 0) {
		retValues = append(retValues, fmt.Sprintf("%v", val))
	}
	return retValues, nil
}
