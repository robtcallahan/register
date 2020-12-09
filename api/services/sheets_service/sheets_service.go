package sheets_service

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"regexp"
	repo "register/pkg/repository"
	"sort"
	"strconv"
	"strings"
	"time"

	"register/api/providers/sheets_provider"
	"register/pkg/auth"
	"register/pkg/models"

	"google.golang.org/api/sheets/v4"
)

const (
	PayCheckName         = "CrowdStrike Salary"
	CreditCardColumnName = "Credit Cards"
	JSONDir = "/Users/rcallahan/workspace/go/src/register/api/services/sheets_service/json/"
	//Reconciled = 0
	Source       = 1
	Date         = 2
	Description  = 3
	Withdrawals  = 4
	Deposits     = 5
	CreditCards  = 6
	BankRegister = 7
	Cleared      = 8
	Delta        = 9
)

type BudgetEntry struct {
	Category     string
	Weekly       float64
	Monthly      float64
	Every2Weeks  float64
	TwiceMonthly float64
	Yearly       float64
	// RegisterColumnName string
}

type BudgetSheet struct {
	ID             int64
	SpreadsheetID  string
	TabName        string
	StartRow       int64
	EndRow         int64
	LastRow        int64
	EndColumnName  string
	EndColumnIndex int64
	Spreadsheet    sheets.Spreadsheet
	BudgetEntries  []*BudgetEntry
	CategoriesMap  map[string]*BudgetEntry
}

type RegisterEntry struct {
	RowID        int64
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
	ID               int64
	TabName          string
	StartRow         int64
	EndRow           int64
	FirstRowToUpdate int64
	LastRow          int64
	EndColumnName    string
	EndColumnIndex   int64
	Spreadsheet      sheets.Spreadsheet
	Register         []*RegisterEntry
	CategoriesMap    map[string]*BudgetEntry
	KeysMap          map[string]bool
	RangeValues      [][]interface{}
}

type sheetsService struct {
	service       *sheets.Service
	SpreadsheetID string
	BudgetSheet   *BudgetSheet
	RegisterSheet *RegisterSheet
	Debug         bool
	Verbose       bool
}

type sheetsServiceInterface interface {
	NewRegisterSheet(startRow, endRow int64) error
	NewBudgetSheet(startRow, endRow int64) error
	ReadRegisterSheet() error
	ReadBudgetSheet() error
}

var SheetsService sheetsServiceInterface = &sheetsService{}


func New(spreadsheetID string, verbose bool) (*sheetsService, error) {
	client := auth.GetClient()
	service, err := sheets.New(client)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve sheets client: %v", err)
	}
	return &sheetsService{
		service:       service,
		SpreadsheetID: spreadsheetID,
		Verbose:       verbose,
	}, nil
}

func (ss *sheetsService) NewRegisterSheet(startRow, endRow int64) error {
	ss.RegisterSheet = &RegisterSheet{
		TabName:        "Register",
		StartRow:       startRow,
		EndRow:         endRow,
		EndColumnName:  "BB",
		EndColumnIndex: 53,
	}

	id, err := ss.getSheetID("Register")
	if err != nil {
		return fmt.Errorf("unable to retrieve spreadsheet: %v", err)
	}
	ss.RegisterSheet.ID = id
	return nil
}

func (ss *sheetsService) NewBudgetSheet(startRow, endRow int64) error {
	ss.BudgetSheet = &BudgetSheet{
		TabName:        "Budget",
		StartRow:       startRow,
		EndRow:         endRow,
		EndColumnName:  "J",
		EndColumnIndex: 9,
	}
	id, err := ss.getSheetID("Budget")
	if err != nil {
		return fmt.Errorf("unable to retrieve spreadsheet: %v", err)
	}
	ss.BudgetSheet.ID = id
	return nil
}

func (ss *sheetsService) ReadRegisterSheet() error {
	range_ := fmt.Sprintf("%s!A%d:%s%d", ss.RegisterSheet.TabName, ss.RegisterSheet.StartRow, ss.RegisterSheet.EndColumnName, ss.RegisterSheet.EndRow)
	resp, err := sheets_provider.SheetsProvider.GetValues(range_)
	if err != nil {
		log.Fatalf("could not get sheet values: %s\n", err.Error())
	}

	rangeValues := resp.Values
	if len(rangeValues) == 0 {
		return fmt.Errorf("no data found: %s", err.Error())
	}

	// determine last used row in the spreadsheet
	ss.RegisterSheet.LastRow = int64(len(rangeValues)) + ss.RegisterSheet.StartRow - 2
	keysMap := make(map[string]bool)
	var register []*RegisterEntry

	for i := 0; int64(i) <= ss.RegisterSheet.LastRow; i += 2 {
		values := rangeValues[i]

		name := ss.getNameField(values)
		source := ss.getSourceField(values)
		date := ss.getDateField(values)
		amount := ss.getAmount(values)

		key := fmt.Sprintf("%s:%s:%s", source, date, amount)
		keysMap[key] = true

		c := &RegisterEntry{
			Key:          key,
			RowID:        ss.RegisterSheet.StartRow + int64(i),
			Reconciled:   getStringField(values, Withdrawals),
			Source:       source,
			Date:         date,
			Name:         name,
			Withdrawal:   ss.GetRegisterField(values, Withdrawals),
			Deposit:      ss.GetRegisterField(values, Deposits),
			CreditCard:   ss.GetRegisterField(values, CreditCards),
			BankRegister: ss.GetRegisterField(values, BankRegister),
			Cleared:      ss.GetRegisterField(values, Cleared),
			Delta:        ss.GetRegisterField(values, Delta),
		}
		register = append(register, c)

		if values[Date] == "" {
			ss.RegisterSheet.FirstRowToUpdate = int64(i) + ss.RegisterSheet.StartRow - 1
			break
		}
	}
	ss.RegisterSheet.Register = register
	ss.RegisterSheet.KeysMap = keysMap
	ss.RegisterSheet.RangeValues = rangeValues
	return nil
}

func (ss *sheetsService) ReadBudgetSheet() error {
	range_ := fmt.Sprintf("%s!B%d:%s%d", ss.BudgetSheet.TabName, ss.BudgetSheet.StartRow, ss.BudgetSheet.EndColumnName, ss.BudgetSheet.EndRow)
	resp, err := sheets_provider.SheetsProvider.GetValues(range_)
	if err != nil {
		return fmt.Errorf("could not get sheet values: %s\n", err.Error())
	}
	if len(resp.Values) == 0 {
		return errors.New("no values found")
	}

	var entries []*BudgetEntry
	var catMap = make(map[string]*BudgetEntry)
	for _, v := range resp.Values {
		if len(v) < 6 {
			continue
		}
		category := fmt.Sprintf("%s", v[0])
		dueDate := fmt.Sprintf("%s", v[1])
		if category == "" || dueDate == "X" {
			continue
		}
		entry := &BudgetEntry{
			Category: category,
			// Weekly:             readDollarsValue(v[2]),
			// Monthly:            readDollarsValue(v[3]),
			// Every2Weeks:        readDollarsValue(v[4]),
			TwiceMonthly: readDollarsValue(v[5]),
			// Yearly:             readDollarsValue(v[6]),
			// RegisterColumnName: config.BudgetCategories[category],
		}
		entries = append(entries, entry)
		catMap[category] = entry
	}
	ss.BudgetSheet.BudgetEntries = entries
	ss.BudgetSheet.CategoriesMap = catMap
	return nil
}

func (ss *sheetsService) getSheetID(tabName string) (int64, error) {
	provider := sheets_provider.New(ss.service, ss.SpreadsheetID)
	spreadsheet, err := provider.GetSpreadsheet()
	if err != nil {
		log.Fatalf("unable to retrieve spreadsheet: %v", err)
	}
	WriteJSONFile(JSONDir + "spreadsheet.json", spreadsheet)

	for _, sheet := range spreadsheet.Sheets {
		p := sheet.Properties
		if p.Title == tabName {
			return p.SheetId, nil
		}
	}
	return 0, fmt.Errorf("could not get sheet id: %v", err)
}

func (ss *sheetsService) UpdateMonthlyCategories(tabName string, catAgg map[string]map[string]float64, columns []models.Column) {
	rows := populateMonthlyCategories(catAgg, columns)
	id, err := ss.getSheetID(tabName)
	checkError(err)
	ss.updateMonthly(id, rows)
}

func (ss *sheetsService) UpdateMonthlyPayees(tabName string, catAgg map[string]map[string]float64) {
	rows := populateMonthlyPayees(catAgg)
	id, err := ss.getSheetID(tabName)
	checkError(err)
	ss.updateMonthly(id, rows)
}

func populateMonthlyCategories(catAgg map[string]map[string]float64, cats []models.Column) []*sheets.RowData {
	var rows []*sheets.RowData

	// sort the months
	months := sortKeys(&catAgg)

	// add the top row of months
	row := addSummaryTopRow(months)
	rows = append(rows, row)

	// make a hash of column names
	cNames := repo.ColumnNames(cats)

	// now all the category rows
	rows = addSummaryRows(rows, catAgg, months, cNames)

	return rows
}

func populateMonthlyPayees(payeeAgg map[string]map[string]float64) []*sheets.RowData {
	var rows []*sheets.RowData

	// sort the months
	months := sortKeys(&payeeAgg)

	// add the top row of months
	row := addSummaryTopRow(months)
	rows = append(rows, row)

	// make a has of payees and sort
	payees := sortKeys(&payeeAgg)

	// now all the payee rows
	rows = addSummaryRows(rows, payeeAgg, months, payees)

	return rows
}

func (ss *sheetsService) updateMonthly(sheetID int64, rows []*sheets.RowData) {
	gc := &sheets.GridCoordinate{
		SheetId:     sheetID,
		RowIndex:    0,
		ColumnIndex: 0,
	}
	updateCellsRequest := sheets.UpdateCellsRequest{
		Fields: "*",
		Rows:   rows,
		Start:  gc,
	}

	var requests []*sheets.Request
	request := sheets.Request{
		UpdateCells: &updateCellsRequest,
	}
	requests = append(requests, &request)

	// create the batch request
	batchUpdateRequest := sheets.BatchUpdateSpreadsheetRequest{
		Requests: requests,
	}

	// execute the request
	resp, err := sheets_provider.SheetsProvider.BatchUpdate(&batchUpdateRequest)
	checkError(err)
	WriteJSONFile(JSONDir + "updateMonthly.json", resp)
}

func (ss *sheetsService) CopyRows(numCopies int) {
	// loop to copy NumberOfCopy times
	var requests []*sheets.Request
	index := ss.RegisterSheet.LastRow
	for i := 1; i <= numCopies; i++ {
		copyPasteRequest := sheets.CopyPasteRequest{
			Source: &sheets.GridRange{
				SheetId:          ss.RegisterSheet.ID,
				StartColumnIndex: 0,
				EndColumnIndex:   ss.RegisterSheet.EndColumnIndex + 1,
				StartRowIndex:    index - 1,
				EndRowIndex:      index + 1,
			},
			Destination: &sheets.GridRange{
				SheetId:          ss.RegisterSheet.ID,
				StartColumnIndex: 0,
				EndColumnIndex:   ss.RegisterSheet.EndColumnIndex + 1,
				StartRowIndex:    index + 1,
				EndRowIndex:      index + 1,
			},
			PasteType: "PASTE_NORMAL",
		}
		request := sheets.Request{
			CopyPaste: &copyPasteRequest,
		}
		requests = append(requests, &request)
		index += 2
	}

	// create the batch request
	batchUpdateRequest := sheets.BatchUpdateSpreadsheetRequest{
		Requests: requests,
	}

	// execute the request
	//provider := sheets_provider.New(ss.service, ss.SpreadsheetID)
	_, err := sheets_provider.SheetsProvider.BatchUpdate(&batchUpdateRequest)
	if err != nil {
		log.Fatalf("could not perform copy/paste action: %v", err)
	}
}

func (ss *sheetsService) Aggregate(cols []models.Column) (map[string]map[string]float64, map[string]map[string]float64) {
	// map of register entries by month and category
	catAgg := make(map[string]map[string]float64)

	// map of register entries by monty and payee
	payeeAgg := make(map[string]map[string]float64)

	rangeVals := ss.RegisterSheet.RangeValues
	for i, r := range ss.RegisterSheet.Register {
		re := regexp.MustCompile(`(\d\d)/\d\d/20`)
		m := re.FindStringSubmatch(r.Date)
		if len(m) > 0 {
			k := m[1] + "/20"
			if _, ok := payeeAgg[k]; !ok {
				payeeAgg[k] = make(map[string]float64)
			}
			payeeAgg[k][r.Name] = payeeAgg[k][r.Name] + r.Deposit - r.Withdrawal - r.CreditCard

			if _, ok := catAgg[k]; !ok {
				catAgg[k] = make(map[string]float64)
			}

			if r.Name == "CrowdStrike Salary" {
				catAgg[k]["CrowdStrike Salary"] += r.Deposit
				continue
			}

			for j := 10; j < len(rangeVals[i*2]); j++ {
				if cols[j].Name == "Credit Cards" || r.Deposit != 0 {
					continue
				}
				f32 := ss.GetRegisterField(rangeVals[i*2], cols[j].ColumnIndex)
				catAgg[k][cols[j].Name] = catAgg[k][cols[j].Name] + f32
			}
		}
	}
	return catAgg, payeeAgg
}

func (ss *sheetsService) readRangeFormulas(readRange string) []string {
	resp, err := sheets_provider.SheetsProvider.GetFormula(readRange)
	if err != nil {
		log.Fatalf("unable to retrieve data from sheet: %v", err)
	}
	rangeValues := resp.Values
	if len(rangeValues) == 0 {
		log.Fatalf("No data found")
	}

	var retValues []string
	for _, val := range rangeValues[0] {
		retValues = append(retValues, fmt.Sprintf("%v", val))
	}
	return retValues
}

func (ss *sheetsService) ReadStringCell(cell string) string {
	return readStringValue(ss.readCell(cell, "string"))
}

func (ss *sheetsService) ReadDollarsCell(cell string) float64 {
	return readDollarsValue(ss.readCell(cell, "dollars"))
}

func (ss *sheetsService) ReadFormulaCell(cell string) string {
	return readStringValue(ss.readCell(cell, "formula"))
}

func (ss *sheetsService) ReadDateCell(cell string) string {
	return readDateValue(ss.readCell(cell, "date"))
}

func (ss *sheetsService) readCell(cell string, dType string) interface{} {
	var val interface{}
	readRange := fmt.Sprintf("%s!%s:%s", ss.RegisterSheet.TabName, cell, cell)

	if dType == "formula" {
		resp, err := sheets_provider.SheetsProvider.GetFormula(readRange)
		if err != nil {
			log.Fatalf("unable to retrieve data from sheet: %v", err)
		}
		val = resp.Values[0][0]
	} else {
		resp, err := sheets_provider.SheetsProvider.GetValues(readRange)
		if err != nil {
			log.Fatalf("unable to retrieve data from sheet: %v", err)
		}
		val = resp.Values[0][0]
	}
	return val
}

func (ss *sheetsService) WriteCell(cell string, value interface{}) *sheets.UpdateValuesResponse {
	var v [][]interface{}
	v = append(v, []interface{}{value})

	writeRange := fmt.Sprintf("%s!%s:%s", ss.RegisterSheet.TabName, cell, cell)
	vRange := &sheets.ValueRange{
		Values: v,
	}
	resp, err := sheets_provider.SheetsProvider.Update(writeRange, vRange)
	if err != nil {
		log.Fatalf("unable to write cell data: %s", err.Error())
	}
	return resp
}

func (ss *sheetsService) UpdateRows(columns []models.Column, nameToCol map[string]string, transactions []*models.Transaction) {
	var requests []*sheets.Request
	rows := ss.populateCells(columns, nameToCol, transactions)

	gc := &sheets.GridCoordinate{
		SheetId:     ss.RegisterSheet.ID,
		RowIndex:    ss.RegisterSheet.FirstRowToUpdate,
		ColumnIndex: 0,
	}
	updateCellsRequest := sheets.UpdateCellsRequest{
		Fields: "*",
		Rows:   rows,
		Start:  gc,
	}

	request := sheets.Request{
		UpdateCells: &updateCellsRequest,
	}
	requests = append(requests, &request)

	// create the batch request
	batchUpdateRequest := sheets.BatchUpdateSpreadsheetRequest{
		Requests: requests,
	}

	// execute the request
	resp, err := sheets_provider.SheetsProvider.BatchUpdate(&batchUpdateRequest)
	if err != nil {
		log.Fatalf("could not perform update action: %v", err)
	}
	WriteJSONFile(JSONDir + "UpdateRows.json", resp)
}

func (ss *sheetsService) populateCells(columns []models.Column, nameToCol map[string]string, transactions []*models.Transaction) []*sheets.RowData {
	// rows will be returned be added to the sheet
	var rows []*sheets.RowData
	// this is the first empty row to be updated
	rowIndex := ss.RegisterSheet.FirstRowToUpdate

	// loop over all the transactions to be added, duplicates have been previously filtered out
	for _, trans := range transactions {
		// cells will be added to row
		var cells []*sheets.CellData
		// each row appended to rows and then rows is returned
		row := &sheets.RowData{}

		// paycheck rows are marked green (like this font color)
		bgColor := "white"
		if trans.Name == PayCheckName {
			bgColor = "green"
		} else if trans.TaxDeductible {
			bgColor = "yellow"
		}

		// first 4 columns
		cells = ss.addSourceDateNameCells(cells, trans, bgColor)
		// amount: deposit, withdrawal, credit
		cells = ss.addAmountCells(cells, trans, bgColor)

		// create the read range to read the 3 adjacent cell formulas for Register, Cleared & Delta
		readRange := fmt.Sprintf("%s!H%d:J%d", "Register", rowIndex+1, rowIndex+1)
		totalsFormulas := ss.readRangeFormulas(readRange)

		if trans.Name == PayCheckName {
			// salary deposit
			cells = ss.addSalaryCells(cells, columns, totalsFormulas)
		} else {
			// all other transaction entries
			cells = ss.addCategoryCells(cells, trans, columns, nameToCol, totalsFormulas)
		}

		row.Values = cells
		rows = append(rows, row)

		var emptyCells []*sheets.CellData
		emptyRow := &sheets.RowData{
			Values: emptyCells,
		}
		rows = append(rows, emptyRow)
		rowIndex += 2
	}
	WriteJSONFile(JSONDir + "populateCells.json", rows)
	return rows
}

func (ss *sheetsService) addSourceDateNameCells(cells []*sheets.CellData, trans *models.Transaction, bgColor string) []*sheets.CellData {
	cells = append(cells, mkNumberCell(4, "center", bgColor, false))
	cells = append(cells, mkStringCell(trans.Source, "center", bgColor, false))
	cells = append(cells, mkDateCell(trans.Date, "center", bgColor, false))
	cells = append(cells, mkStringCell(trans.Name, "left", bgColor, false))
	return cells
}

func (ss *sheetsService) addAmountCells(cells []*sheets.CellData, trans *models.Transaction, bgColor string) []*sheets.CellData {
	if trans.Name == PayCheckName {
		// Wells Fargo Bank transaction
		if trans.Amount < 0 {
			// value is < 0 if it is a deposit
			cells = append(cells, mkStringCell("", "left", bgColor, false))
			cells = append(cells, mkDollarsCell(-1*trans.Amount, "right", bgColor, false))
		} else {
			// show the withdrawal
			cells = append(cells, mkDollarsCell(trans.Amount, "right", bgColor, false))
			cells = append(cells, mkStringCell("", "left", bgColor, false))
		}
		// credit card cell is blank
		cells = append(cells, mkStringCell("", "left", bgColor, false))
	} else {
		// credit card transaction - first 2 cells are blank
		cells = append(cells, mkStringCell("", "left", bgColor, false))
		cells = append(cells, mkStringCell("", "left", bgColor, false))
		cells = append(cells, mkDollarsCell(trans.CreditCard, "right", bgColor, false))
	}
	return cells
}

func (ss *sheetsService) addCategoryCells(cells []*sheets.CellData, trans *models.Transaction, columns []models.Column, nameToCol map[string]string, totalsFormulas []string) []*sheets.CellData {
	// colOffset is because we've already taken care of cols A-G (0-6)
	colOffset := 7
	for i := 0; i < len(columns)-colOffset; i++ {
		col := columns[colOffset+i]
		if ok := intInSlice(i, []int{0, 1, 2}); ok {
			// first 3 columns are Register, Cleared & Delta. We copied the cell formulas above and are pasting here
			cells = append(cells, mkDollarsCellFromFormulaString(totalsFormulas[i], "right", col.Color, false))
		} else if trans.Source != "WellsFargo" && col.Name == CreditCardColumnName {
			// enter a positive value in the credit card column
			cells = append(cells, mkDollarsCell(trans.CreditCard, "left", "yellow", true))
		} else if _, ok := nameToCol[trans.Name]; ok && col.Name == nameToCol[trans.Name] {
			// enter a negative value in the budget category column
			cells = append(cells, mkDollarsCell(-1*trans.Amount, "left", col.Color, true))
		} else {
			// this cell doesn't apply. Just create an empty (opaque) cell.
			cells = append(cells, mkOpaqueCell(col.Color, true))
		}
	}
	return cells
}

func (ss *sheetsService) addSalaryCells(cells []*sheets.CellData, columns []models.Column, totalsFormulas []string) []*sheets.CellData {
	// colOffset is because we've already taken care of cols A-G (0-6)
	colOffset := 7

	// allocate out budgeted amounts and set background color appropriately
	for i := 0; i < len(columns)-colOffset; i++ {
		col := columns[colOffset+i]
		if ok := intInSlice(i, []int{0, 1, 2}); ok {
			// first 3 columns are Register, Cleared & Delta. We copied the cell formulas above and are pasting here
			cells = append(cells, mkDollarsCellFromFormulaString(totalsFormulas[i], "right", col.Color, false))
		} else if col.Name != "" {
			// enter the budgeted amount in this category column
			entry := ss.RegisterSheet.CategoriesMap[col.Name]
			cells = append(cells, mkDollarsCell(entry.TwiceMonthly, "left", col.Color, true))
		} else {
			// this cell doesn't apply. Just create an empty (opaque) cell.
			cells = append(cells, mkOpaqueCell(col.Color, true))
		}
	}
	return cells
}

func (ss *sheetsService) getAmount(values []interface{}) string {
	amt := ""
	v := ""
	if v = fmt.Sprintf("%v", values[Withdrawals]); v != "" {
		amt = "-" + v
	} else if v = fmt.Sprintf("%v", values[Deposits]); v != "" {
		amt = v
	} else if v = fmt.Sprintf("%v", values[CreditCards]); v != "" {
		re := regexp.MustCompile(`[()]`)
		if re.Match([]byte(v)) {
			amt = re.ReplaceAllString(v, "")
		} else {
			amt = "-" + v
		}
	}
	re := regexp.MustCompile(`[\s$,]`)
	amt = re.ReplaceAllString(amt, "")
	return amt
}

func (ss *sheetsService) GetRegisterField(values []interface{}, i int) float64 {
	return readDollarsValue(values[i])
}

func (ss *sheetsService) getDateField(values []interface{}) string {
	dateString := fmt.Sprintf("%v", values[Date])
	if dateString == "" {
		return ""
	}
	return readDateValue(dateString)
}

func (ss *sheetsService) getNameField(values []interface{}) string {
	return readStringValue(values[Description])
}

func (ss *sheetsService) getSourceField(values []interface{}) string {
	source := fmt.Sprintf("%v", values[Source])
	if fmt.Sprintf("%v", values[Source]) == "" {
		source = "WellsFargo"
	}
	return source
}

func addSummaryTopRow(months *[]string) *sheets.RowData {
	bgColor := "grey"

	// create first row of months
	var cells []*sheets.CellData

	// first cell is blank
	cells = append(cells, mkBoldStringCell("Category", "left", bgColor, false))

	// the rest of the months on the top row
	for i := 0; i < len(*months); i++ {
		m := (*months)[i]
		cells = append(cells, mkBoldStringCell(m, "center", bgColor, false))
	}
	// summary columns
	cells = append(cells, mkBoldStringCell("Yearly Totals", "center", bgColor, false))
	cells = append(cells, mkBoldStringCell("Monthly Average", "center", bgColor, false))

	// add the cells to the row
	return &sheets.RowData{Values: cells}
}

func addSummaryRows(rows []*sheets.RowData, aggData map[string]map[string]float64, months, cats *[]string) []*sheets.RowData {
	r := 2
	d := 10
	numCats := len(*cats) - d

	for i := 0; i < numCats; i++ {
		c := (*cats)[d]
		var cells []*sheets.CellData

		// rowColor(rowIndex, even_color, odd_color)
		bgColor := rowColor(i, "white", "lightgrey")

		// 1st column: category name
		cells = append(cells, mkBoldStringCell(c, "left", bgColor, false))

		// remaining columns: $value for each month
		for j := 0; j < len(*months); j++ {
			m := (*months)[j]
			cells = append(cells, mkDollarsCell(aggData[m][c], "right", bgColor, false))
		}

		// add the totals and average in last 2 columns
		f := mkDollarsCellFromFormulaString(fmt.Sprintf("=SUM(B%d:L%d) * -1", r, r), "right", bgColor, false)
		f.UserEnteredFormat.TextFormat.Bold = true
		cells = append(cells, f)
		f = mkDollarsCellFromFormulaString(fmt.Sprintf("=AVERAGE(B%d:L%d) * -1", r, r), "right", bgColor, false)
		f.UserEnteredFormat.TextFormat.Bold = true
		cells = append(cells, f)

		r++
		d++

		// add the cells to the row
		row := &sheets.RowData{Values: cells}
		rows = append(rows, row)
	}
	row := addSalaryRow(r, months, aggData)
	rows = append(rows, row)
	return rows
}

func addSalaryRow(rNum int, months *[]string, aggData map[string]map[string]float64) *sheets.RowData {
	bgColor := "grey"
	var cells []*sheets.CellData

	// 1st column: category name
	cells = append(cells, mkBoldStringCell("CrowdStrike Salary", "left", bgColor, false))

	// remaining columns: $value for each month
	for i := 0; i < len(*months); i++ {
		m := (*months)[i]
		cells = append(cells, mkDollarsCell(aggData[m]["CrowdStrike Salary"], "right", bgColor, false))
	}

	// add the totals and average in last 2 columns
	f := mkDollarsCellFromFormulaString(fmt.Sprintf("=SUM(B%d:L%d) * -1", rNum, rNum), "right", bgColor, false)
	f.UserEnteredFormat.TextFormat.Bold = true
	cells = append(cells, f)
	f = mkDollarsCellFromFormulaString(fmt.Sprintf("=AVERAGE(B%d:L%d) * -1", rNum, rNum), "right", bgColor, false)
	f.UserEnteredFormat.TextFormat.Bold = true
	cells = append(cells, f)

	// add the cells to the row
	return &sheets.RowData{Values: cells}
}

func rowColor(i int, even, odd string) string {
	if i%2 == 0 {
		return even
	}
	return odd
}

func sortKeys(aggMap *map[string]map[string]float64) *[]string {
	keys := make([]string, 0, len(*aggMap))
	for k := range *aggMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return &keys
}

func mkStringCell(value, align, color string, bordersOn bool) *sheets.CellData {
	return &sheets.CellData{
		UserEnteredValue: &sheets.ExtendedValue{
			StringValue: &value,
		},
		UserEnteredFormat: formatCell(align, color, bordersOn),
	}
}

// mkBoldStringCell has test
func mkBoldStringCell(value, align, color string, bordersOn bool) *sheets.CellData {
	c := mkStringCell(value, align, color, bordersOn)
	c.UserEnteredFormat.TextFormat.Bold = true
	return c
}

// mkNumberCell has test
func mkNumberCell(value float64, align, color string, bordersOn bool) *sheets.CellData {
	v := math.Round(value*100) / 100
	return &sheets.CellData{
		UserEnteredValue: &sheets.ExtendedValue{
			NumberValue: &v,
		},
		UserEnteredFormat: formatCell(align, color, bordersOn),
	}
}

// mkDollarsCell has test
func mkDollarsCell(value float64, align, colorName string, bordersOn bool) *sheets.CellData {
	v := math.Round(value*100) / 100
	return &sheets.CellData{
		UserEnteredValue: &sheets.ExtendedValue{
			NumberValue: &v,
		},
		UserEnteredFormat: &sheets.CellFormat{
			HorizontalAlignment: strings.ToUpper(align),
			TextFormat:          font(),
			NumberFormat:        dollarFormat(),
			BackgroundColor:     color(colorName),
			Borders:             borders(bordersOn),
		},
	}
}

// mkDollarsCellFromFormulaString has test
func mkDollarsCellFromFormulaString(value string, align, colorName string, bordersOn bool) *sheets.CellData {
	return &sheets.CellData{
		UserEnteredValue: &sheets.ExtendedValue{
			FormulaValue: &value,
		},
		UserEnteredFormat: &sheets.CellFormat{
			HorizontalAlignment: strings.ToUpper(align),
			TextFormat:          font(),
			NumberFormat:        dollarFormat(),
			BackgroundColor:     color(colorName),
			Borders:             borders(bordersOn),
		},
	}
}

// mkDateCell has test
func mkDateCell(dateString, align, colorName string, bordersOn bool) *sheets.CellData {
	dateString = formatYear(dateString)
	csvTime, err := time.Parse("01/02/06", dateString)
	checkError(err)
	serialTime, err := time.Parse("01/02/2006", "12/30/1899")
	checkError(err)
	sinceTime := csvTime.Sub(serialTime)
	days := sinceTime.Hours() / 24.0
	serialReadStringValue := fmt.Sprintf("%.0f.0", days)
	serialFormatFloat, err := strconv.ParseFloat(serialReadStringValue, 64)
	checkError(err)

	uev := &sheets.ExtendedValue{
		NumberValue: &serialFormatFloat,
	}
	cell := &sheets.CellData{
		UserEnteredValue: uev,
		UserEnteredFormat: &sheets.CellFormat{
			HorizontalAlignment: strings.ToUpper(align),
			TextFormat:          font(),
			NumberFormat:        dateFormat(),
			BackgroundColor:     color(colorName),
			Borders:             borders(bordersOn),
		},
		// dateCell(align, color),
	}
	return cell
}

// mkOpaqueCell has test
func mkOpaqueCell(colorName string, bordersOn bool) *sheets.CellData {
	return &sheets.CellData{
		UserEnteredFormat: &sheets.CellFormat{
			TextFormat:      font(),
			BackgroundColor: color(colorName),
			Borders:         borders(bordersOn),
		},
	}
}

// formatCell has test
func formatCell(align, colorName string, bordersOn bool) *sheets.CellFormat {
	return &sheets.CellFormat{
		HorizontalAlignment: strings.ToUpper(align),
		TextFormat:          font(),
		BackgroundColor:     color(colorName),
		Borders:             borders(bordersOn),
	}
}

// dollarFormat has test
func dollarFormat() *sheets.NumberFormat {
	return &sheets.NumberFormat{
		Pattern: `_("$"* #,##0.00_);_("$"* \(#,##0.00\);_("$"* "-"??_);_(@_)`,
		Type:    "CURRENCY",
	}
}

// dateFormat has test
func dateFormat() *sheets.NumberFormat {
	return &sheets.NumberFormat{
		Pattern: "mm/dd/yy",
		Type:    "DATE",
	}
}

// getStringField has test
func getStringField(values []interface{}, i int) string {
	if i >= 0 && i < len(values) {
		return fmt.Sprintf("%v", values[i])
	}
	return ""
}

// readStringValue has test
func readStringValue(text interface{}) string {
	return fmt.Sprintf("%v", text)
}

// readDollarsValue has test
func readDollarsValue(value interface{}) float64 {
	dollars := fmt.Sprintf("%v", value)
	re := regexp.MustCompile(`[\s$,]`)
	dollars = re.ReplaceAllString(dollars, "")
	if dollars == "-" || dollars == "" {
		return 0
	}

	re = regexp.MustCompile(`[()]`)
	if re.Match([]byte(dollars)) {
		dollars = "-" + re.ReplaceAllString(dollars, "")
	}

	f, err := strconv.ParseFloat(dollars, 64)
	checkError(err)
	return f
}

// readDateValue has test
func readDateValue(dateStr interface{}) string {
	date := fmt.Sprintf("%v", dateStr)
	re := regexp.MustCompile(`(\d+)/(\d+)/(20)?(\d+)`)
	m := re.FindAllStringSubmatch(date, -1)
	mm, _ := strconv.Atoi(m[0][1])
	dd, _ := strconv.Atoi(m[0][2])
	yy, _ := strconv.Atoi(m[0][4])
	d := fmt.Sprintf("%02d/%02d/%02d", mm, dd, yy)
	return d
}

// borders has test
func borders(on bool) *sheets.Borders {
	if !on {
		return &sheets.Borders{}
	}
	return &sheets.Borders{
		Left: &sheets.Border{
			Color: color("black"),
			Style: "SOLID",
		},
		Right: &sheets.Border{
			Color: color("black"),
			Style: "SOLID",
		},
		Bottom: &sheets.Border{
			Color: color("black"),
			Style: "SOLID",
		},
	}
}

// font has test
func font() *sheets.TextFormat {
	return &sheets.TextFormat{
		FontFamily: "Arial",
		FontSize:   10,
	}
}

// color test not necessary
func color(name string) *sheets.Color {
	var colors = map[string]*sheets.Color{
		"black": {
			Alpha: 1,
			Blue:  0,
			Red:   0,
			Green: 0,
		},
		"white": {
			Alpha: 1,
			Blue:  1,
			Red:   1,
			Green: 1,
		},
		"green": {
			Alpha: 1,
			Blue:  0,
			Red:   0.5,
			Green: 1,
		},
		"yellow": {
			Alpha: 1,
			Blue:  0.6,
			Red:   1,
			Green: 1,
		},
		"blue": {
			Alpha: 1,
			Blue:  1,
			Red:   0,
			Green: 0.8,
		},
		"lightgrey": {
			Alpha: 1,
			Blue:  0.937,
			Red:   0.937,
			Green: 0.937,
		},
		"grey": {
			Alpha: 1,
			Blue:  0.8,
			Red:   0.8,
			Green: 0.8,
		},
	}
	return colors[name]
}

// intInSlice has test
func intInSlice(a int, list []int) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

// formatYear has test
func formatYear(date string) string {
	re := regexp.MustCompile(`(\d+/\d+)/20(\d+)`)
	return re.ReplaceAllString(date, "${1}/${2}")
}

func WriteJSONFile(fileName string, data interface{}) {
	j, err := json.Marshal(data)
	checkError(err)
	err = ioutil.WriteFile(fileName, j, 0644)
	checkError(err)
}

// checkError test not necessary
func checkError(err error) {
	if err != nil {
		log.Fatalf("error: %s\n", err.Error())
	}
}
