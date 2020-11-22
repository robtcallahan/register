package main

import (
	"fmt"
	"log"

	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"google.golang.org/api/sheets/v4"
)

// SheetService ...
type SheetService struct {
	Service       *sheets.Service
	SpreadsheetID string
}

// BudgetEntry ...
type BudgetEntry struct {
	Category     string
	Weekly       float32
	Monthly      float32
	Every2Weeks  float32
	TwiceMonthly float32
	Yearly       float32
	// RegisterColumnName string
}

// BudgetSheet ...
type BudgetSheet struct {
	Service        *sheets.Service
	ID             int64
	SpreadsheetID  string
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
	Reconciled   string
	Source       string
	Date         string
	Description  string
	Amount       float32
	Withdrawl    string
	Deposit      string
	CreditCard   string
	BankRegister string
	Cleared      string
	Delta        string
}

// RegisterSheet ...
type RegisterSheet struct {
	Service          *sheets.Service
	ID               int64
	SpreadsheetID    string
	StartRow         int64
	EndRow           int64
	FirstRowToUpdate int64
	LastRow          int64
	EndColumnName    string
	EndColumnIndex   int64
	Spreadsheet      sheets.Spreadsheet
	RegisterEntries  []*RegisterEntry
	CSV              []*CSVRow
	CategoriesMap    map[string]*BudgetEntry
	ValuesMap        map[string][]interface{}
}

func newService() *sheets.Service {
	client := getClient()
	service, err := sheets.New(client)
	if err != nil {
		log.Fatalf("unable to retrieve sheets client: %v", err)
	}
	return service
}

func newRegisterSheet(ss *SheetService, startRow, endRow int64) *RegisterSheet {
	sheet := RegisterSheet{
		Service:        ss.Service,
		SpreadsheetID:  ss.SpreadsheetID,
		StartRow:       startRow,
		EndRow:         endRow,
		EndColumnName:  "BB",
		EndColumnIndex: config.ColumnIndexes["BB"],
	}
	return &sheet
}

func newBudgetSheet(ss *SheetService, startRow, endRow int64) *BudgetSheet {
	sheet := BudgetSheet{
		Service:        ss.Service,
		SpreadsheetID:  ss.SpreadsheetID,
		StartRow:       startRow,
		EndRow:         endRow,
		EndColumnName:  "J",
		EndColumnIndex: config.ColumnIndexes["J"],
	}
	return &sheet
}

func (ss *SheetService) getSheetID(tabName string) (int64, error) {
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

func (bs *BudgetSheet) read() {
	readRange := fmt.Sprintf("%s!B%d:%s%d", config.TabNames["budget"], bs.StartRow, bs.EndColumnName, bs.EndRow)
	resp, err := bs.Service.Spreadsheets.Values.Get(bs.SpreadsheetID, readRange).Do()
	if err != nil {
		log.Fatalf("unable to retrieve data from sheet: %v", err)
	}
	if len(resp.Values) == 0 {
		log.Fatalf("No data found")
	}

	entries := []*BudgetEntry{}
	var catMap map[string]*BudgetEntry = make(map[string]*BudgetEntry)
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

func (rs *RegisterSheet) read() {
	readRange := fmt.Sprintf("%s!A%d:%s%d", config.TabNames["register"], rs.StartRow, rs.EndColumnName, rs.EndRow)
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
	valuesMap := make(map[string][]interface{})
	register := []*RegisterEntry{}

	for i := 0; int64(i) <= rs.LastRow; i += 2 {
		values := rangeValues[i]

		descr := getNameField(values)
		if descr == "VOID" || descr == "Reallocation of funds" {
			continue
		}
		source := getSourceField(values)
		date := getDateField(values)
		amount := getAmountField(values)
		deposit := getRegisterField(values, config.RegisterIndexes["Deposits"])
		if deposit == "" {
			amount = "-" + amount
		}

		key := fmt.Sprintf("%s:%s:%s", source, date, amount)
		valuesMap[key] = values

		c := &RegisterEntry{
			Reconciled:   "4",
			Source:       source,
			Date:         date,
			Description:  descr,
			Withdrawl:    getRegisterField(values, config.RegisterIndexes["Withdrawals"]),
			Deposit:      deposit,
			CreditCard:   getRegisterField(values, config.RegisterIndexes["CreditCards"]),
			BankRegister: getRegisterField(values, config.RegisterIndexes["BankRegister"]),
			Cleared:      getRegisterField(values, config.RegisterIndexes["Cleared"]),
			Delta:        getRegisterField(values, config.RegisterIndexes["Delta"]),
		}
		register = append(register, c)

		if values[config.RegisterIndexes["Date"]] == "" {
			rs.FirstRowToUpdate = int64(i) + rs.StartRow - 1
			break
		}
	}
	rs.ValuesMap = valuesMap
}

func (rs *RegisterSheet) sortByCSVDate() {
	rows := rs.CSV
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Date == rows[j].Date {
			return rows[i].Name < rows[j].Name
		}
		return rows[i].Date < rows[j].Date
	})
	rs.CSV = rows
}

func (rs *RegisterSheet) filterCSVRows(rows []*CSVRow) []*CSVRow {
	filteredRows := []*CSVRow{}
	for _, row := range rows {
		key := fmt.Sprintf("%s:%s:%.2f", row.Source, row.Date, row.Amount)
		if _, ok := (rs.ValuesMap)[key]; !ok {
			filteredRows = append(filteredRows, row)
		}
	}
	return filteredRows
}

func (rs *RegisterSheet) copyRows(numCopies int) {
	// loop to copy NumberOfCopy times
	requests := []*sheets.Request{}
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
	// readRange := fmt.Sprintf("%s!A%d:%s%d", config.TabNames["register"], rs.StartRow, rs.EndColumnName, rs.EndRow)
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

	retValues := []string{}
	for _, val := range rangeValues[0] {
		retValues = append(retValues, fmt.Sprintf("%v", val))
	}
	return retValues
}

func (rs *RegisterSheet) updateRows() {
	requests := []*sheets.Request{}
	rows := rs.populateCells()

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

func (rs *RegisterSheet) populateCells() []*sheets.RowData {
	rows := []*sheets.RowData{}
	rowIndex := rs.FirstRowToUpdate
	for _, csvRow := range rs.CSV {
		if csvRow.Name == "Credit Card Payment" {
			continue
		}

		cells := []*sheets.CellData{}
		row := &sheets.RowData{}

		bgColor := "white"
		if csvRow.Name == config.PaycheckName {
			bgColor = "green"
		}
		borders := false

		cells = append(cells, mkNumberCell(4, "center", bgColor, borders))
		cells = append(cells, mkStringCell(csvRow.Source, "center", bgColor, borders))
		cells = append(cells, mkDateCell(csvRow.Date, "center", bgColor, borders))
		cells = append(cells, mkStringCell(csvRow.Name, "left", bgColor, borders))

		if csvRow.Source == "-" {
			// Wells Fargo Bank transaction
			if csvRow.Amount < 0 {
				cells = append(cells, mkDollarsCell(-1*csvRow.Amount, "right", bgColor, borders))
				cells = append(cells, mkStringCell("", "left", bgColor, borders))
			} else {
				cells = append(cells, mkStringCell("", "left", bgColor, borders))
				cells = append(cells, mkDollarsCell(csvRow.Amount, "right", bgColor, borders))
			}
			cells = append(cells, mkStringCell("", "left", bgColor, borders))
		} else {
			// credit card transaction
			cells = append(cells, mkStringCell("", "left", bgColor, borders))
			cells = append(cells, mkStringCell("", "left", bgColor, borders))
			cells = append(cells, mkDollarsCell(-1*csvRow.Amount, "right", bgColor, borders))
		}

		// salary deposit
		if csvRow.Name == "CrowdStrike Salary" {
			// allocate out budgeted amounts and set background color appropriately
			readRange := fmt.Sprintf("%s!H%d:J%d", config.TabNames["register"], rowIndex+1, rowIndex+1)
			totalsFormulas := rs.readRange(readRange)
			borders = false
			for i := 0; i < len(config.BudgetCategories); i++ {
				cat := config.BudgetCategories[i]
				if ok := intInSlice(i, []int{0, 1, 2}); ok {
					cells = append(cells, mkDollarsCellFromFormulaString(totalsFormulas[i], "right", cat.Color, borders))
				} else if cat.Name == "" {
					borders = true
					cells = append(cells, mkOpaqueCell(cat.Color, borders))
				} else {
					borders = true
					entry := rs.CategoriesMap[cat.Name]
					cells = append(cells, mkDollarsCell(entry.TwiceMonthly, "left", cat.Color, borders))
				}
			}
		}

		row.Values = cells
		rows = append(rows, row)

		emptyCells := []*sheets.CellData{}
		emptyRow := &sheets.RowData{
			Values: emptyCells,
		}
		rows = append(rows, emptyRow)
		rowIndex++
	}
	return rows
}

func mkStringCell(value, align, color string, bordersOn bool) *sheets.CellData {
	return &sheets.CellData{
		UserEnteredValue: &sheets.ExtendedValue{
			StringValue: &value,
		},
		UserEnteredFormat: formatCell(align, color, bordersOn),
	}
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

func mkDollarsCell(value float32, align, colorName string, bordersOn bool) *sheets.CellData {
	v := float64(value)
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
	var colors map[string]*sheets.Color = map[string]*sheets.Color{
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
	}
	return colors[name]
}

// the following "get" functions read and interpret cell data from Google Sheets
func getDollarsField(value interface{}) float32 {
	dollars := fmt.Sprintf("%v", value)
	re := regexp.MustCompile(`[\s\$,]`)
	dollars = re.ReplaceAllString(dollars, "")
	if dollars == "-" || dollars == "" {
		return 0
	}
	f, err := strconv.ParseFloat(dollars, 32)
	checkError(err)
	return float32(f)
}

func getAmountField(values []interface{}) string {
	regi := config.RegisterIndexes
	amount := fmt.Sprintf("%v", values[regi["Withdrawals"]]) +
		fmt.Sprintf("%v", values[regi["Deposits"]]) + fmt.Sprintf("%v", values[regi["CreditCards"]])
	re := regexp.MustCompile(`[\s\$,]`)
	amount = re.ReplaceAllString(amount, "")
	return amount
}

func getDateField(values []interface{}) string {
	dateString := fmt.Sprintf("%v", values[config.RegisterIndexes["Date"]])
	if dateString == "" {
		return ""
	}
	return formatDate(dateString)
}

func getNameField(values []interface{}) string {
	return fmt.Sprintf("%v", values[config.RegisterIndexes["Description"]])
}

func getRegisterField(values []interface{}, i int) string {
	return fmt.Sprintf("%v", values[i])
}

func getSourceField(values []interface{}) string {
	regi := config.RegisterIndexes
	source := fmt.Sprintf("%v", values[regi["Source"]])
	if fmt.Sprintf("%v", values[regi["Source"]]) == "" {
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
