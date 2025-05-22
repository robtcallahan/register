package sheets_service

import (
	"fmt"
	"log"
	"register/pkg/config"
	"register/pkg/models"

	"google.golang.org/api/sheets/v4"
)

type SheetCoords struct {
	StartRow         int64
	EndRow           int64
	LastRow          int64
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

func (ss *SheetsService) NewRegisterSheet(cfg *config.Config) error {
	ss.RegisterSheet = &RegisterSheet{
		TabName: "Register",
		SheetCoords: SheetCoords{
			StartRow:       cfg.RegisterStartRow,
			EndRow:         cfg.RegisterEndRow,
			EndColumnName:  cfg.RegisterCategoryEndColumn,
			EndColumnIndex: cfg.ColumnIndexes[cfg.RegisterCategoryEndColumn],
		},
	}

	id, err := ss.getSheetID("Register")
	if err != nil {
		return fmt.Errorf("unable to retrieve spreadsheet: %v", err)
	}
	ss.RegisterSheet.ID = id
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

	for i = 0; i <= ss.RegisterSheet.SheetCoords.LastRow && !ss.isEmptyRow(resp.Values[i]); i += 2 {
		transactionKey := getTransactionKey(resp.Values[i])
		keysMap[transactionKey] = true
		registerEntry := ss.populateRegisterEntry(resp.Values[i])
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
	return resp.Values[0][0], nil
}

func (ss *SheetsService) CopyRows(numCopies int) error {
	updateReq := ss.copyRowsBatchUpdateRequest(numCopies)
	_, err := ss.Provider.BatchUpdate(&updateReq)
	if err != nil {
		return fmt.Errorf("could not perform copy: %v", err)
	}
	return nil
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
	return readDollarsValue(v), nil
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

		bgColor := getBackgroundColor(trans)
		cells, err := addSourceDateNameCells(cells, trans, bgColor)
		if err != nil {
			return nil, err
		}
		cells = addAmountCell(cells, trans, bgColor)

		totalsFormulas := ss.readRangeFormulas(getRegisterToDeltaReadRange(rowIndex))
		if isPaycheck(trans.Name) {
			cells = ss.addSalaryCells(cells, columns, totalsFormulas)
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
	var cells = make([]*sheets.CellData, ss.RegisterSheet.SheetCoords.EndColumnIndex)
	for i := 1; i < 4; i++ {
		cells = append(cells, mkCellDataString("", "left", "lightgrey", false))
	}
	cells = append(cells, mkCellDataString(note, "left", "lightgrey", false))
	return &sheets.RowData{
		Values: cells,
	}
}

func (ss *SheetsService) addSalaryCells(cells []*sheets.CellData, columns []models.Column, totalsFormulas []string) []*sheets.CellData {
	// colOffset is because we've already taken care of cols A-G (0-6)
	colOffset := BankRegister

	// allocate out budgeted amounts and set background color appropriately
	for i := 0; i < len(columns)-colOffset; i++ {
		col := columns[colOffset+i]
		entry := ss.BudgetSheet.CategoriesMap[col.Name]

		if isRegisterClearedOrDeltaColumn(i) {
			// first 3 columns are Register, Cleared & Delta. We copied the cell formulas above and are pasting here
			cells = append(cells, mkCellDataFormula(totalsFormulas[i], "right", col.Color, false))
		} else if isBudgetColumn(col.Name) && entry != nil {
			// enter the budgeted amount in this category column
			cells = append(cells, mkCellDataDollars(entry.Monthly, "left", col.Color, true))
		} else {
			// this cell doesn't apply. Just create an empty (opaque) cell.
			cells = append(cells, mkCellDataDollars(0.00, "left", col.Color, true))
		}
	}
	return cells
}

func (ss *SheetsService) isEmptyRow(values []interface{}) bool {
	if values[Date] == "" {
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

func (ss *SheetsService) populateRegisterEntry(values []interface{}) *RegisterEntry {
	entry := &RegisterEntry{
		Key:          getTransactionKey(values),
		Reconciled:   getStringField(values, Reconciled),
		Source:       getSourceField(values),
		Date:         getDateField(values),
		Name:         getNameField(values),
		Withdrawal:   getDollarsCellByIndex(values, Withdrawals),
		Deposit:      getDollarsCellByIndex(values, Deposits),
		CreditCard:   getDollarsCellByIndex(values, CreditCards),
		BankRegister: getDollarsCellByIndex(values, BankRegister),
		Cleared:      getDollarsCellByIndex(values, Cleared),
		Delta:        getDollarsCellByIndex(values, Delta),
	}
	return entry
}

func (ss *SheetsService) getSheetID(tabName string) (int64, error) {
	spreadsheet, err := ss.Provider.GetSpreadsheet()
	if err != nil {
		return 0, fmt.Errorf("unable to retrieve spreadsheet: %v", err)
	}

	for _, sheet := range spreadsheet.Sheets {
		p := sheet.Properties
		if p.Title == tabName {
			return p.SheetId, nil
		}
	}
	return 0, fmt.Errorf("could not get sheet id: %v", err)
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

func (ss *SheetsService) readRangeFormulas(readRange string) []string {
	resp, err := ss.Provider.GetFormula(readRange)
	if err != nil {
		log.Fatalf("unable to retrieve data from sheet: %v", err)
	}
	rangeValues := resp.Values
	if len(rangeValues) == 0 {
		log.Fatalf("no data found for read range: %s", readRange)
	}

	var retValues []string
	for _, val := range rangeValues[0] {
		retValues = append(retValues, fmt.Sprintf("%v", val))
	}
	return retValues
}
