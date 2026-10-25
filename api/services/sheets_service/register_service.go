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
	SheetCoords SheetCoords
	Entries     []*RegisterEntry
	RangeValues [][]any
}

// Public methods

func (ss *RegisterService) NewRegisterSheet(cfg *config.Config, columns []models.Column) (*RegisterSheet, error) {
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
	registerSheet := &RegisterSheet{
		TabName: RegisterTabName,
		SheetCoords: SheetCoords{
			StartRow:       cfg.RegisterStartRow,
			EndRow:         cfg.RegisterEndRow,
			EndColumnName:  endColumnName,
			EndColumnIndex: endColumnIndex,
		},
	}

	spreadsheet, err := ss.Provider.GetSpreadsheet(RegisterTabName)
	if err != nil {
		return nil, err
	}
	registerSheet.ID = spreadsheet.SheetID
	registerSheet.SheetCoords.RowCount = spreadsheet.RowCount
	registerSheet.SheetCoords.ColumnCount = spreadsheet.ColumnCount
	if len(columns) == 0 {
		// no DB columns: the register ends at the sheet's grid edge
		registerSheet.SheetCoords.EndColumnName = models.ColumnLetter(int(spreadsheet.ColumnCount))
		registerSheet.SheetCoords.EndColumnIndex = spreadsheet.ColumnCount - 1
	}
	return registerSheet, nil
}

func (ss *RegisterService) ReadRegisterSheet(registerSheet *RegisterSheet) (*RegisterSheet, error) {
	var entries []*RegisterEntry
	var i int64

	var readRange = ss.getReadRange(*registerSheet)
	values, err := ss.Provider.GetValues(readRange)
	if err != nil {
		return nil, fmt.Errorf("could not get sheet values: %w\n", err)
	}
	if len(values) == 0 {
		return nil, fmt.Errorf("no data found for read range: %s", ss.getReadRange(*registerSheet))
	}

	// determine last used row in the spreadsheet
	registerSheet.SheetCoords.LastRow = ss.getLastRow(*registerSheet, values)

	// The Reconciled..Delta index constants assume fixed positions; verify
	// the sheet agrees before trusting them.
	if err := ss.checkRegisterHeaders(*registerSheet); err != nil {
		return nil, err
	}

	for i = 0; i <= registerSheet.SheetCoords.LastRow; i += 2 {
		row := getRowValues(values, int(i))
		if ss.isEmptyRow(row) {
			break
		}
		registerEntry, err := ss.populateRegisterEntry(row)
		if err != nil {
			return nil, err
		}
		registerEntry.RowID = ss.getRowID(*registerSheet, i)
		entries = append(entries, registerEntry)
	}

	registerSheet.SheetCoords.FirstRowToUpdate = ss.getFirstRowToUpdate(*registerSheet, i)
	registerSheet.Entries = entries
	registerSheet.RangeValues = values

	return registerSheet, nil
}

func (ss *RegisterService) ReadCell(registerSheet RegisterSheet, cell string, cellDataType CellDataType) (any, error) {
	var values [][]any
	var err error

	readRange := fmt.Sprintf("%s!%s:%s", registerSheet.TabName, cell, cell)

	if cellDataType == CellDataFormula {
		values, err = ss.Provider.GetFormula(readRange)
	} else {
		values, err = ss.Provider.GetValues(readRange)
	}
	if err != nil {
		return nil, err
	}
	row := getRowValues(values, 0)
	if len(row) == 0 {
		return nil, fmt.Errorf("no data found for cell: %s", cell)
	}
	return row[0], nil
}

// errSheetFull means the copy would run past the end of the sheet's grid.
var errSheetFull = errors.New("register sheet is full")

func (ss *RegisterService) CopyRows(registerSheet RegisterSheet, numCopies int) error {
	if err := ss.checkRowsFit(registerSheet, numCopies); err != nil {
		return err
	}
	updateReq := ss.copyRowsBatchUpdateRequest(registerSheet, numCopies)
	_, err := ss.Provider.BatchUpdate(&updateReq)
	if err != nil {
		if strings.Contains(err.Error(), "grid limits") {
			return fmt.Errorf("%w — add rows and re-run", errSheetFull)
		}
		return fmt.Errorf("could not perform copy: %w", err)
	}
	return nil
}

func (ss *RegisterService) checkRowsFit(registerSheet RegisterSheet, numCopies int) error {
	coords := registerSheet.SheetCoords
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

func (ss *RegisterService) ReadStringCell(registerSheet RegisterSheet, cell string) (string, error) {
	v, err := ss.ReadCell(registerSheet, cell, CellDataString)
	if err != nil {
		return "", fmt.Errorf("could not read string cell %s: %w", cell, err)
	}
	return readStringValue(v), nil
}

func (ss *RegisterService) ReadDollarsCell(registerSheet RegisterSheet, cell string) (float64, error) {
	v, err := ss.ReadCell(registerSheet, cell, CellDataDollars)
	if err != nil {
		return 0, err
	}
	return readDollarsValue(v)
}

func (ss *RegisterService) ReadFormulaCell(registerSheet RegisterSheet, cell string) (string, error) {
	v, err := ss.ReadCell(registerSheet, cell, CellDataFormula)
	if err != nil {
		return "", err
	}
	return readStringValue(v), nil
}

func (ss *RegisterService) ReadDateCell(registerSheet RegisterSheet, cell string) (string, error) {
	v, err := ss.ReadCell(registerSheet, cell, CellDataDate)
	if err != nil {
		return "", err
	}
	return models.ReadDateValue(readStringValue(v)), nil
}

func (ss *RegisterService) WriteCell(registerSheet RegisterSheet, cell string, value any) (*sheets.UpdateValuesResponse, error) {
	var v [][]any
	v = append(v, []any{value})

	writeRange := fmt.Sprintf("%s!%s:%s", registerSheet.TabName, cell, cell)
	vRange := &sheets.ValueRange{
		Values: v,
	}
	resp, err := ss.Provider.Update(writeRange, vRange)
	if err != nil {
		return nil, fmt.Errorf("unable to write cell data: %w", err)
	}
	return resp, nil
}

func (ss *RegisterService) isEmptyRow(values []any) bool {
	if getStringField(values, Date) == "" {
		return true
	}
	return false
}

func (ss *RegisterService) getFirstRowToUpdate(registerSheet RegisterSheet, i int64) int64 {
	return registerSheet.SheetCoords.StartRow + i - 1
}

func (ss *RegisterService) getLastRow(registerSheet RegisterSheet, values [][]any) int64 {
	return int64(len(values)) + registerSheet.SheetCoords.StartRow - 2
}

func (ss *RegisterService) getReadRange(registerSheet RegisterSheet) string {
	return fmt.Sprintf("%s!A%d:%s%d", registerSheet.TabName, registerSheet.SheetCoords.StartRow,
		registerSheet.SheetCoords.EndColumnName, registerSheet.SheetCoords.EndRow)
}

func (ss *RegisterService) getRowID(registerSheet RegisterSheet, i int64) int64 {
	return registerSheet.SheetCoords.StartRow + i
}

func (ss *RegisterService) populateRegisterEntry(values []any) (*RegisterEntry, error) {
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

func (ss *RegisterService) copyRowsBatchUpdateRequest(registerSheet RegisterSheet, numCopies int) sheets.BatchUpdateSpreadsheetRequest {
	var requests []*sheets.Request

	index := registerSheet.SheetCoords.LastRow
	for i := 1; i <= numCopies; i++ {
		copyPasteRequest := sheets.CopyPasteRequest{
			Source:      ss.getCopySource(registerSheet, index),
			Destination: ss.getCopyDestination(registerSheet, index),
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

func (ss *RegisterService) getCopySource(registerSheet RegisterSheet, index int64) *sheets.GridRange {
	return &sheets.GridRange{
		SheetId:          registerSheet.ID,
		StartColumnIndex: 0,
		EndColumnIndex:   registerSheet.SheetCoords.EndColumnIndex + 1,
		StartRowIndex:    index - 1,
		EndRowIndex:      index + 1,
	}
}

func (ss *RegisterService) getCopyDestination(registerSheet RegisterSheet, index int64) *sheets.GridRange {
	return &sheets.GridRange{
		SheetId:          registerSheet.ID,
		StartColumnIndex: 0,
		EndColumnIndex:   registerSheet.SheetCoords.EndColumnIndex + 1,
		StartRowIndex:    index + 1,
		EndRowIndex:      index + 1,
	}
}

func (ss *RegisterService) readRangeFormulas(readRange string) ([]string, error) {
	values, err := ss.Provider.GetFormula(readRange)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve data from sheet: %w", err)
	}
	rangeValues := values
	if len(rangeValues) == 0 {
		return nil, fmt.Errorf("no data found for read range: %s", readRange)
	}

	var retValues []string
	for _, val := range getRowValues(rangeValues, 0) {
		retValues = append(retValues, fmt.Sprintf("%v", val))
	}
	return retValues, nil
}

// registerFixedHeaders are the expected labels of the Register tab's fixed
// A-J columns (row 4), in position order. They anchor the Reconciled..Delta
// index constants: a moved, renamed, or missing header is a loud error,
// not a silently wrong column.
var registerFixedHeaders = []string{
	"Recon", "Check", "Date", "Description", "Withdrawals",
	"Deposits", "Credit Purchases", "Register", "Cleared", "Delta",
}

// checkRegisterHeaders verifies the Register tab's fixed column headers
// against the positions the index constants assume.
func (ss *RegisterService) checkRegisterHeaders(registerSheet RegisterSheet) error {
	headerRange := fmt.Sprintf("%s!A4:J4", registerSheet.TabName)
	values, err := ss.Provider.GetValues(headerRange)
	if err != nil {
		return fmt.Errorf("could not read register header row: %w", err)
	}
	if len(values) == 0 {
		return errors.New("no header row found on Register tab")
	}
	decoder, err := newRowDecoder(values[0])
	if err != nil {
		return err
	}
	for i, want := range registerFixedHeaders {
		got, err := decoder.require(want)
		if err != nil {
			return fmt.Errorf("register header %q not found: %w", want, err)
		}
		if got != i {
			return fmt.Errorf("register header %q moved: expected column %d, found at column %d", want, i, got)
		}
	}
	return nil
}
