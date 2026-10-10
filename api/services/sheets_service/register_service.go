package sheets_service

import (
	"errors"
	"fmt"
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
	// highest-indexed column). Callers that don't read the DB (register
	// copy) pass no columns; they get the sheet's own grid width below.
	var endColumnName string
	var endColumnIndex int64
	if len(columns) > 0 {
		set := models.NewColumnSet(columns)
		if end, ok := set.End(); ok {
			endColumnName = set.LetterFor(end.ColumnIndex)
			endColumnIndex = int64(set.SliceOffset(end.ColumnIndex))
		}
	}
	ss.RegisterSheet = &RegisterSheet{
		TabName: RegisterTabName,
		SheetCoords: SheetCoords{
			StartRow:       cfg.RegisterStartRow,
			EndRow:         cfg.RegisterEndRow,
			EndColumnName:  endColumnName,
			EndColumnIndex: endColumnIndex,
		},
	}

	props, err := ss.getSheetProperties(RegisterTabName)
	if err != nil {
		return err
	}
	ss.RegisterSheet.ID = props.SheetId
	if props.GridProperties != nil {
		ss.RegisterSheet.SheetCoords.RowCount = props.GridProperties.RowCount
		ss.RegisterSheet.SheetCoords.ColumnCount = props.GridProperties.ColumnCount
		if len(columns) == 0 {
			// no DB columns: the register ends at the sheet's grid edge
			columnCount := props.GridProperties.ColumnCount
			ss.RegisterSheet.SheetCoords.EndColumnName = models.ColumnLetter(int(columnCount))
			ss.RegisterSheet.SheetCoords.EndColumnIndex = columnCount - 1
		}
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
