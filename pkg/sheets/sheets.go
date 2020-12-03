/*
Copyright © 2020 Rob Callahan <robtcallahan@aol.com>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package sheets

import (
	"fmt"
	"log"

	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"register/pkg/auth"
	"register/pkg/banking"
	"register/pkg/config"
	cfg "register/pkg/config"
	"register/pkg/models"

	"google.golang.org/api/sheets/v4"
)

// SheetService ...
type SheetService struct {
	Service       *sheets.Service
	SpreadsheetID string
	ColumnIndexes map[string]int64
}

// BudgetEntry ...
type BudgetEntry struct {
	Category     string
	Weekly       float64
	Monthly      float64
	Every2Weeks  float64
	TwiceMonthly float64
	Yearly       float64
	// RegisterColumnName string
}

// BudgetSheet ...
type BudgetSheet struct {
	Service        *sheets.Service
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

// RegisterEntry ...
type RegisterEntry struct {
	RowID        int64
	Reconciled   string
	Source       string
	Date         string
	Name         string
	Amount       float64
	Withdrawal   float32
	Deposit      float32
	CreditCard   float32
	BankRegister float32
	Cleared      float32
	Delta        float32
}

// RegisterSheet ...
type RegisterSheet struct {
	Service          *sheets.Service
	Config           cfg.Config
	ID               int64
	SpreadsheetID    string
	PaycheckName     string
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
	Debug            bool
}

// NewService ...
func NewService() *sheets.Service {
	client := auth.GetClient()
	service, err := sheets.New(client)
	if err != nil {
		log.Fatalf("unable to retrieve sheets client: %v", err)
	}
	return service
}

// NewRegisterSheet ...
func NewRegisterSheet(ss *SheetService, config cfg.Config, startRow, endRow int64, debug bool) *RegisterSheet {
	sheet := RegisterSheet{
		Service:        ss.Service,
		Config:         config,
		SpreadsheetID:  ss.SpreadsheetID,
		TabName:        config.TabNames["register"],
		StartRow:       startRow,
		EndRow:         endRow,
		EndColumnName:  "BB",
		EndColumnIndex: config.ColumnIndexes["BB"],
		Debug:          debug,
	}
	return &sheet
}

// NewBudgetSheet ...
func NewBudgetSheet(ss *SheetService, tabName string, startRow, endRow int64) *BudgetSheet {
	sheet := BudgetSheet{
		Service:        ss.Service,
		SpreadsheetID:  ss.SpreadsheetID,
		TabName:        tabName,
		StartRow:       startRow,
		EndRow:         endRow,
		EndColumnName:  "J",
		EndColumnIndex: ss.ColumnIndexes["J"],
	}
	return &sheet
}

// GetSheetID ...
func (ss *SheetService) GetSheetID(tabName string) (int64, error) {
	spreadsheet, err := ss.Service.Spreadsheets.Get(ss.SpreadsheetID).Do()
	if err != nil {
		log.Fatalf("unable to retrieve spreadsheet: %v", err)
	}
	for _, sheet := range spreadsheet.Sheets {
		p := sheet.Properties
		if p.Title == tabName {
			return p.SheetId, nil
		}
	}
	return 0, fmt.Errorf("could not get sheet id: %v", err)
}

// Read ...
func (bs *BudgetSheet) Read() {
	readRange := fmt.Sprintf("%s!B%d:%s%d", bs.TabName, bs.StartRow, bs.EndColumnName, bs.EndRow)
	resp, err := bs.Service.Spreadsheets.Values.Get(bs.SpreadsheetID, readRange).Do()
	if err != nil {
		log.Fatalf("unable to retrieve data from sheet: %v", err)
	}
	if len(resp.Values) == 0 {
		log.Fatalf("No data found")
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
			// Weekly:             getDollarsField(v[2]),
			// Monthly:            getDollarsField(v[3]),
			// Every2Weeks:        getDollarsField(v[4]),
			TwiceMonthly: getDollarsField(v[5]),
			// Yearly:             getDollarsField(v[6]),
			// RegisterColumnName: config.BudgetCategories[category],
		}
		entries = append(entries, entry)
		catMap[category] = entry
	}
	bs.BudgetEntries = entries
	bs.CategoriesMap = catMap
}

// Read ...
func (rs *RegisterSheet) Read() ([]*RegisterEntry, map[string]bool, [][]interface{}) {
	readRange := fmt.Sprintf("%s!A%d:%s%d", rs.TabName, rs.StartRow, rs.EndColumnName, rs.EndRow)
	resp, err := rs.Service.Spreadsheets.Values.Get(rs.SpreadsheetID, readRange).Do()
	if err != nil {
		log.Fatalf("unable to retrieve data from sheet: %v", err)
	}

	// rangeValues := resp.ValueRanges[0].Values
	rangeValues := resp.Values
	if len(rangeValues) == 0 {
		log.Fatalf("No data found")
	}

	// determine last used row in the spreadsheet
	rs.LastRow = int64(len(rangeValues)) + rs.StartRow - 2
	keysMap := make(map[string]bool)
	var register []*RegisterEntry

	for i := 0; int64(i) <= rs.LastRow; i += 2 {
		values := rangeValues[i]

		name := rs.getNameField(values)
		// if name == "VOID" || name == "Reallocation of funds" {
		// 	continue
		// }
		source := rs.getSourceField(values)
		date := rs.getDateField(values)
		amount := rs.getAmountFieldForKey(values)

		key := fmt.Sprintf("%s:%s:%s", source, date, amount)
		keysMap[key] = true

		c := &RegisterEntry{
			RowID:        rs.StartRow + int64(i),
			Reconciled:   getStringField(values, rs.Config.RegisterIndexes["Withdrawals"]),
			Source:       source,
			Date:         date,
			Name:         name,
			Withdrawal:   rs.GetRegisterField(values, rs.Config.RegisterIndexes["Withdrawals"]),
			Deposit:      rs.GetRegisterField(values, rs.Config.RegisterIndexes["Deposits"]),
			CreditCard:   rs.GetRegisterField(values, rs.Config.RegisterIndexes["CreditCards"]),
			BankRegister: rs.GetRegisterField(values, rs.Config.RegisterIndexes["BankRegister"]),
			Cleared:      rs.GetRegisterField(values, rs.Config.RegisterIndexes["Cleared"]),
			Delta:        rs.GetRegisterField(values, rs.Config.RegisterIndexes["Delta"]),
		}
		register = append(register, c)

		if values[rs.Config.RegisterIndexes["Date"]] == "" {
			rs.FirstRowToUpdate = int64(i) + rs.StartRow - 1
			break
		}
	}
	return register, keysMap, rangeValues
}

// CopyRows ...
func (rs *RegisterSheet) CopyRows(numCopies int) {
	// loop to copy NumberOfCopy times
	var requests []*sheets.Request
	index := rs.LastRow
	for i := 1; i <= numCopies; i++ {
		copyPasteRequest := sheets.CopyPasteRequest{
			Source: &sheets.GridRange{
				SheetId:          rs.ID,
				StartColumnIndex: 0,
				EndColumnIndex:   rs.EndColumnIndex + 1,
				StartRowIndex:    index - 1,
				EndRowIndex:      index + 1,
			},
			Destination: &sheets.GridRange{
				SheetId:          rs.ID,
				StartColumnIndex: 0,
				EndColumnIndex:   rs.EndColumnIndex + 1,
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
	// log.Println("Performing copy/paste")
	_, err := rs.Service.Spreadsheets.BatchUpdate(rs.SpreadsheetID, &batchUpdateRequest).Do()
	if err != nil {
		log.Fatalf("could not perform copy/paste action: %v", err)
	}
}

func (rs *RegisterSheet) readRange(readRange string) []string {
	call := rs.Service.Spreadsheets.Values.BatchGet(rs.SpreadsheetID)
	call.ValueRenderOption("FORMULA")
	call.Ranges(readRange)
	resp, err := call.Do()
	if err != nil {
		log.Fatalf("unable to retrieve data from sheet: %v", err)
	}
	rangeValues := resp.ValueRanges[0].Values

	if len(rangeValues) == 0 {
		log.Fatalf("No data found")
	}

	var retValues []string
	for _, val := range rangeValues[0] {
		retValues = append(retValues, fmt.Sprintf("%v", val))
	}
	return retValues
}

// UpdateRows ...
func (rs *RegisterSheet) UpdateRows(columns []models.Column, nameToCol map[string]string, transactions []*banking.Transaction) {
	var requests []*sheets.Request
	rows := rs.populateCells(columns, nameToCol, transactions)

	gc := &sheets.GridCoordinate{
		SheetId:     rs.ID,
		RowIndex:    rs.FirstRowToUpdate,
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
	_, err := rs.Service.Spreadsheets.BatchUpdate(rs.SpreadsheetID, &batchUpdateRequest).Do()
	if err != nil {
		log.Fatalf("could not perform update action: %v", err)
	}
}

func (rs *RegisterSheet) populateCells(columns []models.Column, nameToCol map[string]string, transactions []*banking.Transaction) []*sheets.RowData {
	// rows will be returned be added to the sheet
	var rows []*sheets.RowData
	// this is the first empty row to be updated
	rowIndex := rs.FirstRowToUpdate

	// loop over all the transactions to be added, duplicates have been previously filtered out
	for _, trans := range transactions {
		// TODO: should filter this out sooner
		if trans.Name == "Credit Card Payment" || trans.Name == "PAYMENT THANK YOU" {
			// we don't show these.
			continue
		}

		// cells will be added to row
		var cells []*sheets.CellData
		// each row appended to rows and then rows is returned
		row := &sheets.RowData{}

		// paycheck rows are marked green (like this font color)
		bgColor := "white"
		if trans.Name == rs.PaycheckName {
			bgColor = "green"
		}

		cells = append(cells, mkNumberCell(4, "center", bgColor, false))
		cells = append(cells, mkStringCell(trans.Source, "center", bgColor, false))
		cells = append(cells, mkDateCell(trans.Date, "center", bgColor, false))
		cells = append(cells, mkStringCell(trans.Name, "left", bgColor, false))

		if trans.Source == "-" {
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
			// credit card transaction
			// first 2 cells are blank
			cells = append(cells, mkStringCell("", "left", bgColor, false))
			cells = append(cells, mkStringCell("", "left", bgColor, false))
			cells = append(cells, mkDollarsCell(trans.CreditCard, "right", bgColor, false))
		}

		// create the read range to read the 3 adjacent cell formulas for Register, Cleared & Delta
		readRange := fmt.Sprintf("%s!H%d:J%d", rs.Config.TabNames["register"], rowIndex+1, rowIndex+1)
		totalsFormulas := rs.readRange(readRange)

		// colOffset is because we've already taken care of cols A-G (0-6)
		colOffset := 7

		// salary deposit
		if trans.Name == "CrowdStrike Salary" {
			// allocate out budgeted amounts and set background color appropriately
			for i := 0; i < len(columns)-colOffset; i++ {
				col := columns[colOffset+i]
				if ok := intInSlice(i, []int{0, 1, 2}); ok {
					// first 3 columns are Register, Cleared & Delta. We copied the cell formulas above and are pasting here
					cells = append(cells, mkDollarsCellFromFormulaString(totalsFormulas[i], "right", col.Color, false))
				} else if col.Name != "" {
					// enter the budgeted amount in this category column
					entry := rs.CategoriesMap[col.Name]
					cells = append(cells, mkDollarsCell(entry.TwiceMonthly, "left", col.Color, true))
				} else {
					// this cell doesn't apply. Just create an empty (opaque) cell.
					cells = append(cells, mkOpaqueCell(col.Color, true))
				}
			}
		} else {
			for i := 0; i < len(columns)-colOffset; i++ {
				col := columns[colOffset+i]
				if ok := intInSlice(i, []int{0, 1, 2}); ok {
					// first 3 columns are Register, Cleared & Delta. We copied the cell formulas above and are pasting here
					cells = append(cells, mkDollarsCellFromFormulaString(totalsFormulas[i], "right", col.Color, false))
				} else if trans.Source != "-" && col.Name == rs.Config.CreditCardColumnName {
					// enter a positive value in the credit card column
					cells = append(cells, mkDollarsCell(trans.CreditCard, "left", "yellow", true))
				} else if _, ok := nameToCol[trans.Name]; ok && col.Name == nameToCol[trans.Name] {
					// enter a negative value in the budget category column
					cells = append(cells, mkDollarsCell(-1*trans.CreditCard, "left", col.Color, true))
				} else {
					// this cell doesn't apply. Just create an empty (opaque) cell.
					cells = append(cells, mkOpaqueCell(col.Color, true))
				}
			}
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
	return rows
}

// UpdateMonthlyCategories ...
func (ss *SheetService) UpdateMonthlyCategories(tabName string, catAgg map[string]map[string]float64, columns []models.Column) {
	rows := populateMonthlyCategories(catAgg, columns)

	id, err := ss.GetSheetID(tabName)
	checkError(err)

	ss.updateMonthly(id, rows)
}

// UpdateMonthlyPayees ...
func (ss *SheetService) UpdateMonthlyPayees(tabName string, catAgg map[string]map[string]float64) {
	rows := populateMonthlyPayees(catAgg)
	id, err := ss.GetSheetID(tabName)
	checkError(err)
	ss.updateMonthly(id, rows)
}

func (ss *SheetService) updateMonthly(sheetID int64, rows []*sheets.RowData) {
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
	_, err := ss.Service.Spreadsheets.BatchUpdate(config.ReadConfig().SpreadsheetID, &batchUpdateRequest).Do()
	checkError(err)
}

func populateMonthlyCategories(catAgg map[string]map[string]float64, cats []models.Column) []*sheets.RowData {
	var rows []*sheets.RowData

	// sort the months
	months := sortKeys(&catAgg)

	// add the top row of months
	row := addSummaryTopRow(months)
	rows = append(rows, row)

	// make a hash of column names
	cNames := columnNames(cats)

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

func columnNames(cats []models.Column) *[]string {
	names := make([]string, 0, len(cats))
	for _, c := range cats {
		names = append(names, c.Name)
	}
	return &names
}

func mkStringCell(value, align, color string, bordersOn bool) *sheets.CellData {
	return &sheets.CellData{
		UserEnteredValue: &sheets.ExtendedValue{
			StringValue: &value,
		},
		UserEnteredFormat: formatCell(align, color, bordersOn),
	}
}

func mkBoldStringCell(value, align, color string, bordersOn bool) *sheets.CellData {
	c := sheets.CellData{
		UserEnteredValue: &sheets.ExtendedValue{
			StringValue: &value,
		},
		UserEnteredFormat: formatCell(align, color, bordersOn),
	}
	c.UserEnteredFormat.TextFormat.Bold = true
	return &c
}

func mkNumberCell(value float32, align, color string, bordersOn bool) *sheets.CellData {
	v := float64(value)
	return &sheets.CellData{
		UserEnteredValue: &sheets.ExtendedValue{
			NumberValue: &v,
		},
		UserEnteredFormat: formatCell(align, color, bordersOn),
	}
}

func mkDollarsCell(value float64, align, colorName string, bordersOn bool) *sheets.CellData {
	return &sheets.CellData{
		UserEnteredValue: &sheets.ExtendedValue{
			NumberValue: &value,
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

func mkDateCell(dateString, align, colorName string, bordersOn bool) *sheets.CellData {
	dateString = formatYear(dateString)
	csvTime, err := time.Parse("01/02/06", dateString)
	checkError(err)
	serialTime, err := time.Parse("01/02/2006", "12/30/1899")
	checkError(err)
	sinceTime := csvTime.Sub(serialTime)
	days := sinceTime.Hours() / 24.0
	serialFormatString := fmt.Sprintf("%.0f.0", days)
	serialFormatFloat, err := strconv.ParseFloat(serialFormatString, 64)
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

func mkOpaqueCell(colorName string, bordersOn bool) *sheets.CellData {
	return &sheets.CellData{
		UserEnteredFormat: &sheets.CellFormat{
			TextFormat:      font(),
			BackgroundColor: color(colorName),
			Borders:         borders(bordersOn),
		},
	}
}

func formatCell(align, colorName string, bordersOn bool) *sheets.CellFormat {
	return &sheets.CellFormat{
		HorizontalAlignment: strings.ToUpper(align),
		TextFormat:          font(),
		BackgroundColor:     color(colorName),
		Borders:             borders(bordersOn),
	}
}

func dollarFormat() *sheets.NumberFormat {
	return &sheets.NumberFormat{
		Pattern: `_("$"* #,##0.00_);_("$"* \(#,##0.00\);_("$"* "-"??_);_(@_)`,
		Type:    "CURRENCY",
	}
}

func dateFormat() *sheets.NumberFormat {
	return &sheets.NumberFormat{
		Pattern: "mm/dd/yy",
		Type:    "DATE",
	}
}

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

func font() *sheets.TextFormat {
	return &sheets.TextFormat{
		FontFamily: "Arial",
		FontSize:   10,
	}
}

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

// the following "get" functions read and interpret cell data from Google Sheets
func getDollarsField(value interface{}) float64 {
	dollars := fmt.Sprintf("%v", value)
	re := regexp.MustCompile(`[\s$,]`)
	dollars = re.ReplaceAllString(dollars, "")
	if dollars == "-" || dollars == "" {
		return 0
	}
	f, err := strconv.ParseFloat(dollars, 32)
	checkError(err)
	return f
}

func (rs *RegisterSheet) getAmountFieldForKey(values []interface{}) string {
	regI := rs.Config.RegisterIndexes

	amt := ""
	v := ""
	if v = fmt.Sprintf("%v", values[regI["Withdrawals"]]); v != "" {
		amt = "-" + v
	} else if v = fmt.Sprintf("%v", values[regI["Deposits"]]); v != "" {
		amt = v
	} else if v = fmt.Sprintf("%v", values[regI["CreditCards"]]); v != "" {
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

func getStringField(values []interface{}, i int) string {
	return fmt.Sprintf("%v", values[i])
}

// GetRegisterField ...
func (rs *RegisterSheet) GetRegisterField(values []interface{}, i int) float32 {
	amt := fmt.Sprintf("%s", values[i])
	re := regexp.MustCompile(`[\s$,-]`)
	amt = re.ReplaceAllString(amt, "")
	if amt == "" {
		return 0
	}

	re = regexp.MustCompile(`[()]`)
	if re.Match([]byte(amt)) {
		amt = "-" + re.ReplaceAllString(amt, "")
	}
	f, err := strconv.ParseFloat(amt, 32)
	if err != nil {
		fmt.Printf("error: amt: %s\n", amt)
		panic(err)
	}
	return float32(f)
}

func (rs *RegisterSheet) getDateField(values []interface{}) string {
	dateString := fmt.Sprintf("%v", values[rs.Config.RegisterIndexes["Date"]])
	if dateString == "" {
		return ""
	}
	return formatDate(dateString)
}

func (rs *RegisterSheet) getNameField(values []interface{}) string {
	return fmt.Sprintf("%v", values[rs.Config.RegisterIndexes["Description"]])
}

func (rs *RegisterSheet) getSourceField(values []interface{}) string {
	regI := rs.Config.RegisterIndexes
	source := fmt.Sprintf("%v", values[regI["Source"]])
	if fmt.Sprintf("%v", values[regI["Source"]]) == "" {
		source = "-"
	}
	return source
}

func intInSlice(a int, list []int) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func checkError(err error) {
	if err != nil {
		panic(err)
	}
}

func formatYear(date string) string {
	re := regexp.MustCompile(`(\d+/\d+)/20(\d+)`)
	return re.ReplaceAllString(date, "${1}/${2}")
}

func formatDate(date string) string {
	re := regexp.MustCompile(`(\d+)/(\d+)/(20)?(\d+)`)
	m := re.FindAllStringSubmatch(date, -1)
	mm, _ := strconv.Atoi(m[0][1])
	dd, _ := strconv.Atoi(m[0][2])
	yy, _ := strconv.Atoi(m[0][4])
	d := fmt.Sprintf("%02d/%02d/%02d", mm, dd, yy)
	return d
}
