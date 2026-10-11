package sheets_service

import (
	"fmt"
	"maps"
	"regexp"
	"slices"

	"register/pkg/models"

	"google.golang.org/api/sheets/v4"
)

func (ss *SheetsService) UpdateMonthlyCategories(tabName string, catAgg map[string]map[string]float64, columns []models.Column) error {
	rows := populateMonthlyCategories(catAgg, columns)
	props, err := ss.getSheetProperties(tabName)
	if err != nil {
		return fmt.Errorf("error: %w\n", err)
	}
	return ss.updateMonthly(tabName, props.SheetId, rows)
}

func (ss *SheetsService) UpdateMonthlyPayees(tabName string, catAgg map[string]map[string]float64) error {
	incomeNames := make([]string, len(ss.IncomeSources))
	for i, s := range ss.IncomeSources {
		incomeNames[i] = s.Name
	}
	rows := populateMonthlyPayees(catAgg, incomeNames)
	props, err := ss.getSheetProperties(tabName)
	if err != nil {
		return fmt.Errorf("error: %w\n", err)
	}
	return ss.updateMonthly(tabName, props.SheetId, rows)
}

func (ss *SheetsService) updateMonthly(tabName string, sheetID int64, rows []*sheets.RowData) error {
	// clear the tab first so stale rows/columns from a previous run don't linger
	if err := ss.Provider.ClearValues(tabName); err != nil {
		return fmt.Errorf("error: %w\n", err)
	}
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
	_, err := ss.Provider.BatchUpdate(&batchUpdateRequest)
	if err != nil {
		return fmt.Errorf("error: %w\n", err)
	}
	return nil
}

func (ss *SheetsService) Aggregate(cols []models.Column) (map[string]map[string]float64, map[string]map[string]float64, error) {
	// map of register entries by month and category
	catAgg := make(map[string]map[string]float64)

	// map of register entries by monty and payee
	payeeAgg := make(map[string]map[string]float64)

	rangeValues := ss.RegisterSheet.RangeValues
	for i, r := range ss.RegisterSheet.Register {
		// r.Date is MM/DD/YY (readDateValue emits a 2-digit year)
		re := regexp.MustCompile(`(\d\d)/\d\d/(\d\d)`)
		m := re.FindStringSubmatch(r.Date)
		if len(m) > 0 {
			k := m[1] + "/" + m[2]
			if _, ok := payeeAgg[k]; !ok {
				payeeAgg[k] = make(map[string]float64)
			}
			payeeAgg[k][r.Name] = payeeAgg[k][r.Name] + r.Deposit - r.Withdrawal - r.CreditCard

			if _, ok := catAgg[k]; !ok {
				catAgg[k] = make(map[string]float64)
			}

			if isIncomeName(r.Name, ss.IncomeSources) {
				catAgg[k][r.Name] += r.Deposit
				continue
			}

			row := getRowValues(rangeValues, i*2)
			for _, col := range cols {
				// categories only: the fixed A-J block isn't spending, and
				// the black payoff columns accumulate card balances rather
				// than monthly spend
				if !col.IsCategory() || col.Color == models.ColorGrey {
					continue
				}
				if r.Deposit != 0 {
					continue
				}
				// ColumnIndex is 1-based (the sheet); the row slice is 0-based
				f32, err := getDollarsCellByIndex(row, col.ColumnIndex-1)
				if err != nil {
					return nil, nil, err
				}
				catAgg[k][col.Name] = catAgg[k][col.Name] + f32
			}
		}
	}
	return catAgg, payeeAgg, nil
}

func populateMonthlyCategories(catAgg map[string]map[string]float64, cats []models.Column) []*sheets.RowData {
	var rows []*sheets.RowData

	months := sortAggregateMapKeys(catAgg)

	row := addSummaryTopRow(months)
	rows = append(rows, row)

	// category names only: the fixed A-J block has no place in the summary
	var names []string
	for _, col := range cats {
		if col.IsCategory() {
			names = append(names, col.Name)
		}
	}

	// now all the category rows
	rows = addSummaryRows(rows, catAgg, months, names, nil)

	return rows
}

func populateMonthlyPayees(payeeAgg map[string]map[string]float64, incomeNames []string) []*sheets.RowData {
	var rows []*sheets.RowData

	// sort the months
	months := sortAggregateMapKeys(payeeAgg)

	// add the top row of months
	row := addSummaryTopRow(months)
	rows = append(rows, row)

	// collect every payee seen in any month
	payeeSet := make(map[string]bool)
	for _, byPayee := range payeeAgg {
		for payee := range byPayee {
			payeeSet[payee] = true
		}
	}
	payees := slices.Sorted(maps.Keys(payeeSet))

	// now all the payee rows
	rows = addSummaryRows(rows, payeeAgg, months, payees, incomeNames)

	return rows
}

func addSummaryRows(rows []*sheets.RowData, aggData map[string]map[string]float64, months, names []string, incomeNames []string) []*sheets.RowData {
	r := 2

	for i, c := range names {
		var cells []*sheets.CellData

		// getOddOrEvenRowColor(rowIndex, even_color, odd_color)
		bgColor := getOddOrEvenRowColor(i, "white", "lightgrey")

		// 1st column: category name
		cells = append(cells, mkBoldFormat(c, "left", bgColor, false))

		// remaining columns: $value for each month
		for j := 0; j < len(months); j++ {
			m := months[j]
			cells = append(cells, mkCellDataDollars(aggData[m][c], "right", bgColor, false))
		}

		// add the totals and average in last 2 columns; the month block
		// runs B..endCol, derived from the actual month count
		endCol := models.ColumnLetter(len(months) + 1) // B is 1-based index 2
		f := mkCellDataFormula(fmt.Sprintf("=SUM(B%d:%s%d) * -1", r, endCol, r), "right", bgColor, false)
		f.UserEnteredFormat.TextFormat.Bold = true
		cells = append(cells, f)
		f = mkCellDataFormula(fmt.Sprintf("=AVERAGE(B%d:%s%d) * -1", r, endCol, r), "right", bgColor, false)
		f.UserEnteredFormat.TextFormat.Bold = true
		cells = append(cells, f)

		r++

		// add the cells to the row
		row := &sheets.RowData{Values: cells}
		rows = append(rows, row)
	}
	for _, name := range incomeNames {
		if aggDataHasName(aggData, name) {
			rows = append(rows, addSummarySalaryRow(name, r, months, aggData))
			r++
		}
	}
	return rows
}

// aggDataHasName reports whether any month has an entry for the name.
func aggDataHasName(aggData map[string]map[string]float64, name string) bool {
	for _, m := range aggData {
		if _, ok := m[name]; ok {
			return true
		}
	}
	return false
}

func addSummaryTopRow(months []string) *sheets.RowData {
	bgColor := "grey"

	// create first row of months
	var cells []*sheets.CellData

	// first cell is blank
	cells = append(cells, mkBoldFormat("Category", "left", bgColor, false))

	// the rest of the months on the top row
	for i := 0; i < len(months); i++ {
		m := months[i]
		cells = append(cells, mkBoldFormat(m, "center", bgColor, false))
	}
	// summary columns
	cells = append(cells, mkBoldFormat("Yearly Totals", "center", bgColor, false))
	cells = append(cells, mkBoldFormat("Monthly Average", "center", bgColor, false))

	// add the cells to the row
	return &sheets.RowData{Values: cells}
}

func addSummarySalaryRow(name string, rNum int, months []string, aggData map[string]map[string]float64) *sheets.RowData {
	bgColor := "grey"
	var cells []*sheets.CellData

	// 1st column: category name
	cells = append(cells, mkBoldFormat(name, "left", bgColor, false))

	// remaining columns: $value for each month
	for i := 0; i < len(months); i++ {
		m := months[i]
		cells = append(cells, mkCellDataDollars(aggData[m][name], "right", bgColor, false))
	}

	// add the totals and average in last 2 columns; the month block
	// runs B..endCol, derived from the actual month count
	endCol := models.ColumnLetter(len(months) + 1) // B is 1-based index 2
	f := mkCellDataFormula(fmt.Sprintf("=SUM(B%d:%s%d) * -1", rNum, endCol, rNum), "right", bgColor, false)
	f.UserEnteredFormat.TextFormat.Bold = true
	cells = append(cells, f)
	f = mkCellDataFormula(fmt.Sprintf("=AVERAGE(B%d:%s%d) * -1", rNum, endCol, rNum), "right", bgColor, false)
	f.UserEnteredFormat.TextFormat.Bold = true
	cells = append(cells, f)

	// add the cells to the row
	return &sheets.RowData{Values: cells}
}

func (ss *SheetsService) UpdateYearlySummary(tabName string, catAgg map[string]map[string]float64, columns []models.Column) error {
	incomeNames := make([]string, len(ss.IncomeSources))
	for i, s := range ss.IncomeSources {
		incomeNames[i] = s.Name
	}
	rows := populateYearlySummary(catAgg, columns, incomeNames)
	props, err := ss.getSheetProperties(tabName)
	if err != nil {
		return fmt.Errorf("error: %w\n", err)
	}
	// a fresh tab only has 26 columns; the yearly table is wider
	if err := ss.ensureColumnCount(props, int64(len(rows[0].Values))); err != nil {
		return fmt.Errorf("error: %w\n", err)
	}
	return ss.updateMonthly(tabName, props.SheetId, rows)
}

// ensureColumnCount widens the tab when the table needs more columns
// than the sheet currently has.
func (ss *SheetsService) ensureColumnCount(props *sheets.SheetProperties, need int64) error {
	have := int64(26)
	if props.GridProperties != nil {
		have = props.GridProperties.ColumnCount
	}
	if have >= need {
		return nil
	}
	req := &sheets.BatchUpdateSpreadsheetRequest{
		Requests: []*sheets.Request{
			{
				UpdateSheetProperties: &sheets.UpdateSheetPropertiesRequest{
					Properties: &sheets.SheetProperties{
						SheetId: props.SheetId,
						GridProperties: &sheets.GridProperties{
							ColumnCount: need,
						},
					},
					Fields: "gridProperties.columnCount",
				},
			},
		},
	}
	_, err := ss.Provider.BatchUpdate(req)
	return err
}

// populateYearlySummary renders the aggregate as months-as-rows,
// categories-as-columns: the transpose of the monthly tabs.
func populateYearlySummary(catAgg map[string]map[string]float64, cols []models.Column, incomeNames []string) []*sheets.RowData {
	var rows []*sheets.RowData

	months := sortAggregateMapKeys(catAgg)

	// data columns: categories in sheet order, then income
	var headers []string
	for _, col := range cols {
		if col.IsCategory() {
			headers = append(headers, col.Name)
		}
	}
	headers = append(headers, incomeNames...)

	rows = append(rows, addYearlyTopRow(headers))

	r := 2
	for i, m := range months {
		bgColor := getOddOrEvenRowColor(i, "white", "lightgrey")
		var cells []*sheets.CellData
		cells = append(cells, mkBoldFormat(m, "left", bgColor, false))
		for _, h := range headers {
			cells = append(cells, mkCellDataDollars(catAgg[m][h], "right", bgColor, false))
		}
		endCol := models.ColumnLetter(len(headers) + 1) // B is 1-based index 2
		f := mkCellDataFormula(fmt.Sprintf("=SUM(B%d:%s%d)", r, endCol, r), "right", bgColor, false)
		f.UserEnteredFormat.TextFormat.Bold = true
		cells = append(cells, f)
		rows = append(rows, &sheets.RowData{Values: cells})
		r++
	}

	// totals row
	bgColor := "grey"
	var cells []*sheets.CellData
	cells = append(cells, mkBoldFormat("Total", "left", bgColor, false))
	for j := range headers {
		colLetter := models.ColumnLetter(j + 2)
		f := mkCellDataFormula(fmt.Sprintf("=SUM(%s2:%s%d)", colLetter, colLetter, r-1), "right", bgColor, false)
		f.UserEnteredFormat.TextFormat.Bold = true
		cells = append(cells, f)
	}
	endCol := models.ColumnLetter(len(headers) + 1)
	f := mkCellDataFormula(fmt.Sprintf("=SUM(B%d:%s%d)", r, endCol, r), "right", bgColor, false)
	f.UserEnteredFormat.TextFormat.Bold = true
	cells = append(cells, f)
	rows = append(rows, &sheets.RowData{Values: cells})

	return rows
}

func addYearlyTopRow(headers []string) *sheets.RowData {
	bgColor := "grey"
	var cells []*sheets.CellData
	cells = append(cells, mkBoldFormat("Month", "left", bgColor, false))
	for _, h := range headers {
		cells = append(cells, mkBoldFormat(h, "center", bgColor, false))
	}
	cells = append(cells, mkBoldFormat("Total", "center", bgColor, false))
	return &sheets.RowData{Values: cells}
}
