package sheets_service

import (
	"fmt"
	"regexp"
	"register/pkg/models"
	repo "register/pkg/repository"

	"google.golang.org/api/sheets/v4"
)

func (ss *SheetsService) UpdateMonthlyCategories(tabName string, catAgg map[string]map[string]float64, columns []models.Column) error {
	rows := populateMonthlyCategories(catAgg, columns)
	id, err := ss.getSheetID(tabName)
	if err != nil {
		return fmt.Errorf("error: %s\n", err.Error())
	}
	return ss.updateMonthly(id, rows)
}

func (ss *SheetsService) UpdateMonthlyPayees(tabName string, catAgg map[string]map[string]float64) error {
	rows := populateMonthlyPayees(catAgg)
	id, err := ss.getSheetID(tabName)
	if err != nil {
		return fmt.Errorf("error: %s\n", err.Error())
	}
	return ss.updateMonthly(id, rows)
}

func (ss *SheetsService) updateMonthly(sheetID int64, rows []*sheets.RowData) error {
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
		return fmt.Errorf("error: %s\n", err.Error())
	}
	return nil
}

func (ss *SheetsService) Aggregate(cols []models.Column) (map[string]map[string]float64, map[string]map[string]float64) {
	// map of register entries by month and category
	catAgg := make(map[string]map[string]float64)

	// map of register entries by monty and payee
	payeeAgg := make(map[string]map[string]float64)

	rangeValues := ss.RegisterSheet.RangeValues
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

			if r.Name == PayCheckName {
				catAgg[k][PayCheckName] += r.Deposit
				continue
			}

			for j := 10; j < len(rangeValues[i*2]); j++ {
				if cols[j].Name == "Credit Cards" || r.Deposit != 0 {
					continue
				}
				f32 := getDollarsCellByIndex(rangeValues[i*2], cols[j].ColumnIndex)
				catAgg[k][cols[j].Name] = catAgg[k][cols[j].Name] + f32
			}
		}
	}
	return catAgg, payeeAgg
}

func populateMonthlyCategories(catAgg map[string]map[string]float64, cats []models.Column) []*sheets.RowData {
	var rows []*sheets.RowData

	months := sortAggregateMapKeys(&catAgg)

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
	months := sortAggregateMapKeys(&payeeAgg)

	// add the top row of months
	row := addSummaryTopRow(months)
	rows = append(rows, row)

	// make a has of payees and sort
	payees := sortAggregateMapKeys(&payeeAgg)

	// now all the payee rows
	rows = addSummaryRows(rows, payeeAgg, months, payees)

	return rows
}

func addSummaryRows(rows []*sheets.RowData, aggData map[string]map[string]float64, months, cats *[]string) []*sheets.RowData {
	r := 2
	d := 10
	numCats := len(*cats) - d

	for i := 0; i < numCats; i++ {
		c := (*cats)[d]
		var cells []*sheets.CellData

		// getOddOrEvenRowColor(rowIndex, even_color, odd_color)
		bgColor := getOddOrEvenRowColor(i, "white", "lightgrey")

		// 1st column: category name
		cells = append(cells, mkBoldFormat(c, "left", bgColor, false))

		// remaining columns: $value for each month
		for j := 0; j < len(*months); j++ {
			m := (*months)[j]
			cells = append(cells, mkCellDataDollars(aggData[m][c], "right", bgColor, false))
		}

		// add the totals and average in last 2 columns
		f := mkCellDataFormula(fmt.Sprintf("=SUM(B%d:L%d) * -1", r, r), "right", bgColor, false)
		f.UserEnteredFormat.TextFormat.Bold = true
		cells = append(cells, f)
		f = mkCellDataFormula(fmt.Sprintf("=AVERAGE(B%d:L%d) * -1", r, r), "right", bgColor, false)
		f.UserEnteredFormat.TextFormat.Bold = true
		cells = append(cells, f)

		r++
		d++

		// add the cells to the row
		row := &sheets.RowData{Values: cells}
		rows = append(rows, row)
	}
	row := addSummarySalaryRow(r, months, aggData)
	rows = append(rows, row)
	return rows
}

func addSummaryTopRow(months *[]string) *sheets.RowData {
	bgColor := "grey"

	// create first row of months
	var cells []*sheets.CellData

	// first cell is blank
	cells = append(cells, mkBoldFormat("Category", "left", bgColor, false))

	// the rest of the months on the top row
	for i := 0; i < len(*months); i++ {
		m := (*months)[i]
		cells = append(cells, mkBoldFormat(m, "center", bgColor, false))
	}
	// summary columns
	cells = append(cells, mkBoldFormat("Yearly Totals", "center", bgColor, false))
	cells = append(cells, mkBoldFormat("Monthly Average", "center", bgColor, false))

	// add the cells to the row
	return &sheets.RowData{Values: cells}
}

func addSummarySalaryRow(rNum int, months *[]string, aggData map[string]map[string]float64) *sheets.RowData {
	bgColor := "grey"
	var cells []*sheets.CellData

	// 1st column: category name
	cells = append(cells, mkBoldFormat(PayCheckName, "left", bgColor, false))

	// remaining columns: $value for each month
	for i := 0; i < len(*months); i++ {
		m := (*months)[i]
		cells = append(cells, mkCellDataDollars(aggData[m][PayCheckName], "right", bgColor, false))
	}

	// add the totals and average in last 2 columns
	f := mkCellDataFormula(fmt.Sprintf("=SUM(B%d:L%d) * -1", rNum, rNum), "right", bgColor, false)
	f.UserEnteredFormat.TextFormat.Bold = true
	cells = append(cells, f)
	f = mkCellDataFormula(fmt.Sprintf("=AVERAGE(B%d:L%d) * -1", rNum, rNum), "right", bgColor, false)
	f.UserEnteredFormat.TextFormat.Bold = true
	cells = append(cells, f)

	// add the cells to the row
	return &sheets.RowData{Values: cells}
}
