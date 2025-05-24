package sheets_service

import (
	"encoding/json"
	"fmt"
	"google.golang.org/api/sheets/v4"
	"log"
	"math"
	"os"
	"regexp"
	"register/pkg/banking"
	"register/pkg/models"
	"sort"
	"strconv"
	"strings"
	"time"
)

var cellColors = map[string]*sheets.Color{
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

func hasNote(trans *models.Transaction) bool {
	if trans.Note != "" {
		return true
	}
	return false
}

func isPaycheck(name string) bool {
	if strings.Contains(name, banking.PayCheckName) {
		return true
	}
	return false
}

func getRegisterToDeltaReadRange(i int64) string {
	return fmt.Sprintf("%s!%s%d:%s%d", "Register", RegisterColumn, i+1, DeltaColumn, i+1)
}

func getBackgroundColor(trans *models.Transaction) string {
	if isPaycheck(trans.Name) {
		return "green"
	} else if trans.TaxDeductible {
		return "yellow"
	}
	return "white"
}

func isCheckingAccount(trans *models.Transaction) bool {
	if trans.Source == CheckingAccountSourceName || trans.IsCheck {
		return true
	}
	return false
}

func isBudgetColumn(colName string) bool {
	if colName != "" && colName != CreditCardColumnName {
		return true
	}
	return false
}

func getOddOrEvenRowColor(i int, even, odd string) string {
	if i%2 == 0 {
		return even
	}
	return odd
}

func mkColor(name string) *sheets.Color {
	return cellColors[name]
}

func mkBorders(on bool) *sheets.Borders {
	if !on {
		return &sheets.Borders{}
	}
	return &sheets.Borders{
		Left: &sheets.Border{
			Color: mkColor("black"),
			Style: "SOLID",
		},
		Right: &sheets.Border{
			Color: mkColor("black"),
			Style: "SOLID",
		},
		Bottom: &sheets.Border{
			Color: mkColor("black"),
			Style: "SOLID",
		},
	}
}

func mkTextFormat() *sheets.TextFormat {
	return &sheets.TextFormat{
		FontFamily: "Arial",
		FontSize:   10,
	}
}

func mkTextFormatReconcileColumn() *sheets.TextFormat {
	return &sheets.TextFormat{
		FontFamily: "Archivo Black",
		FontSize:   12,
	}
}

func mkNumberFormatDate() *sheets.NumberFormat {
	return &sheets.NumberFormat{
		Pattern: "mm/dd/yy",
		Type:    "DATE",
	}
}

func mkCellFormat(align, colorName string, bordersOn bool) *sheets.CellFormat {
	return &sheets.CellFormat{
		HorizontalAlignment: strings.ToUpper(align),
		TextFormat:          mkTextFormat(),
		BackgroundColor:     mkColor(colorName),
		Borders:             mkBorders(bordersOn),
	}
}

func mkCellDataString(value, align, color string, borders bool) *sheets.CellData {
	return &sheets.CellData{
		UserEnteredValue: &sheets.ExtendedValue{
			StringValue: &value,
		},
		UserEnteredFormat: mkCellFormat(align, color, borders),
	}
}

func mkBoldFormat(value, align, color string, borders bool) *sheets.CellData {
	c := mkCellDataString(value, align, color, borders)
	c.UserEnteredFormat.TextFormat.Bold = true
	return c
}

func mkCellDataNumber(value float64, align, color string, borders bool) *sheets.CellData {
	v := math.Round(value*100) / 100
	return &sheets.CellData{
		UserEnteredValue: &sheets.ExtendedValue{
			NumberValue: &v,
		},
		UserEnteredFormat: mkCellFormat(align, color, borders),
	}
}

func mkCellDataDollars(value float64, align, colorName string, borders bool) *sheets.CellData {
	v := math.Round(value*100) / 100
	return &sheets.CellData{
		UserEnteredValue: &sheets.ExtendedValue{
			NumberValue: &v,
		},
		UserEnteredFormat: &sheets.CellFormat{
			HorizontalAlignment: strings.ToUpper(align),
			TextFormat:          mkTextFormat(),
			NumberFormat:        getNumberFormatDollars(),
			BackgroundColor:     mkColor(colorName),
			Borders:             mkBorders(borders),
		},
	}
}

func mkCellDataEmpty(align, colorName string, borders bool) *sheets.CellData {
	s := ""
	return &sheets.CellData{
		UserEnteredValue: &sheets.ExtendedValue{
			StringValue: &s,
		},
		UserEnteredFormat: &sheets.CellFormat{
			HorizontalAlignment: strings.ToUpper(align),
			TextFormat:          mkTextFormat(),
			NumberFormat:        getNumberFormatDollars(),
			BackgroundColor:     mkColor(colorName),
			Borders:             mkBorders(borders),
		},
	}
}

func mkCellDataFormula(value string, align, colorName string, borders bool) *sheets.CellData {
	return &sheets.CellData{
		UserEnteredValue: &sheets.ExtendedValue{
			FormulaValue: &value,
		},
		UserEnteredFormat: &sheets.CellFormat{
			HorizontalAlignment: strings.ToUpper(align),
			TextFormat:          mkTextFormat(),
			NumberFormat:        getNumberFormatDollars(),
			BackgroundColor:     mkColor(colorName),
			Borders:             mkBorders(borders),
		},
	}
}

func getCellDataReconcileColumn(value, align, colorName string, borders bool) *sheets.CellData {
	return &sheets.CellData{
		UserEnteredValue: &sheets.ExtendedValue{
			StringValue: &value,
		},
		UserEnteredFormat: &sheets.CellFormat{
			HorizontalAlignment: strings.ToUpper(align),
			TextFormat:          mkTextFormatReconcileColumn(),
			BackgroundColor:     mkColor(colorName),
			Borders:             mkBorders(borders),
		},
	}
}

func getNumberFormatDollars() *sheets.NumberFormat {
	return &sheets.NumberFormat{
		Pattern: `_("$"* #,##0.00_);_("$"* \(#,##0.00\);_("$"* "-"??_);_(@_)`,
		Type:    "CURRENCY",
	}
}

func getCellDataDate(dateString, align, colorName string, borders bool) (*sheets.CellData, error) {
	dateString = formatYear(dateString)
	csvTime, err := time.Parse("01/02/06", dateString)
	if err != nil {
		return nil, fmt.Errorf("could not parse date string: %s: %s", dateString, err.Error())
	}

	serialTime, _ := time.Parse("01/02/2006", "12/30/1899")
	serialFormatFloat, err := strconv.ParseFloat(fmt.Sprintf("%.0f.0", csvTime.Sub(serialTime).Hours()/24.0), 64)
	if err != nil {
		return nil, fmt.Errorf("could not create serial format float with serialTime=%+v, csvTime=%+v: %s",
			serialTime, csvTime, err.Error())
	}

	cell := &sheets.CellData{
		UserEnteredValue: &sheets.ExtendedValue{
			NumberValue: &serialFormatFloat,
		},
		UserEnteredFormat: &sheets.CellFormat{
			HorizontalAlignment: strings.ToUpper(align),
			TextFormat:          mkTextFormat(),
			NumberFormat:        mkNumberFormatDate(),
			BackgroundColor:     mkColor(colorName),
			Borders:             mkBorders(borders),
		},
	}
	return cell, nil
}

func formatYear(date string) string {
	re := regexp.MustCompile(`^(\d+/\d+)/\d*(\d{2})$`)
	return re.ReplaceAllString(date, "${1}/${2}")
}

func readStringValue(text interface{}) string {
	return fmt.Sprintf("%v", text)
}

func readDollarsValue(value interface{}) float64 {
	dollars := readStringValue(value)
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
	// TODO: change to error return
	if err != nil {
		log.Fatalf("parseFloat error: %s", err.Error())
	}
	return f
}

func readDateValue(dateStr interface{}) string {
	date := readStringValue(dateStr)
	re := regexp.MustCompile(`(\d+)/(\d+)/(20)?(\d+)`)
	m := re.FindAllStringSubmatch(date, -1)
	mm, _ := strconv.Atoi(m[0][1])
	dd, _ := strconv.Atoi(m[0][2])
	yy, _ := strconv.Atoi(m[0][4])
	d := fmt.Sprintf("%02d/%02d/%02d", mm, dd, yy)
	return d
}

func addDeposit(amount float64, bgColor string, cells []*sheets.CellData) []*sheets.CellData {
	cells = append(cells, mkCellDataString("", "left", bgColor, false))
	cells = append(cells, mkCellDataDollars(amount, "right", bgColor, false))
	return cells
}

func addWithdrawal(amount float64, bgColor string, cells []*sheets.CellData) []*sheets.CellData {
	cells = append(cells, mkCellDataDollars(amount, "right", bgColor, false))
	cells = append(cells, mkCellDataString("", "left", bgColor, false))
	return cells
}

func addCheckingTransaction(trans *models.Transaction, bgColor string, cells []*sheets.CellData) []*sheets.CellData {
	if trans.Deposit > 0 {
		cells = addDeposit(trans.Deposit, bgColor, cells)
	} else {
		cells = addWithdrawal(trans.Withdrawal, bgColor, cells)
	}
	// add credit card cell
	cells = append(cells, mkCellDataString("", "left", bgColor, false))
	return cells
}

func addCCTransaction(trans *models.Transaction, bgColor string, cells []*sheets.CellData) []*sheets.CellData {
	cells = append(cells, mkCellDataString("", "left", bgColor, false))
	cells = append(cells, mkCellDataString("", "left", bgColor, false))
	cells = append(cells, mkCellDataDollars(trans.CreditPurchase, "right", bgColor, false))
	return cells
}

func isCorrectBudgetColumn(transName, colName string, transNameToColName map[string]string) bool {
	if _, ok := transNameToColName[transName]; ok {
		if colName == transNameToColName[transName] {
			return true
		}
	}
	return false
}

func getStringField(values []interface{}, i int) string {
	if i >= 0 && i < len(values) {
		return readStringValue(values[i])
	}
	return ""
}

func intInSlice(a int, list []int) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func sortAggregateMapKeys(aggMap *map[string]map[string]float64) *[]string {
	keys := make([]string, 0, len(*aggMap))
	for k := range *aggMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return &keys
}

func isCreditCardTransaction(source, colName string) bool {
	if source != CheckingAccountSourceName && colName == CreditCardColumnName {
		return true
	}
	return false
}

func isRegisterClearedOrDeltaColumn(i int) bool {
	if ok := intInSlice(i, []int{0, 1, 2}); ok {
		return true
	}
	return false
}

func getSourceField(values []interface{}) string {
	if fmt.Sprintf("%v", values[Source]) == "" {
		return strings.ToLower(CheckingAccountSourceName)
	}
	return strings.ToLower(getStringField(values, Source))
}

func getDateField(values []interface{}) string {
	dateString := fmt.Sprintf("%v", values[Date])
	if dateString == "" {
		return ""
	}
	return readDateValue(dateString)
}

func getAmountString(values []interface{}) string {
	amt := ""
	v := ""
	if v = fmt.Sprintf("%v", values[Withdrawals]); v != "" {
		amt = v
	} else if v = fmt.Sprintf("%v", values[Deposits]); v != "" {
		amt = v
	} else if v = fmt.Sprintf("%v", values[CreditCards]); v != "" {
		re := regexp.MustCompile(`[()]`)
		if re.Match([]byte(v)) {
			amt = re.ReplaceAllString(v, "")
			re := regexp.MustCompile(`[\s$,]`)
			amt = re.ReplaceAllString(amt, "")
			fl, _ := strconv.ParseFloat(amt, 64)
			amt = fmt.Sprintf("%.2f", -1*fl)
		} else {
			amt = v
		}
	}
	re := regexp.MustCompile(`[\s$,]`)
	amt = re.ReplaceAllString(amt, "")
	return amt
}

func getTransactionKey(values []interface{}) string {
	return fmt.Sprintf("%s:%s:%s", getSourceField(values), getDateField(values), getAmountString(values))
}

func addSourceDateNameCells(cells []*sheets.CellData, trans *models.Transaction, bgColor string) ([]*sheets.CellData, error) {
	cells = append(cells, getCellDataReconcileColumn("X", "center", bgColor, false))
	cells = append(cells, mkCellDataString(trans.Source, "center", bgColor, false))
	dateCell, err := getCellDataDate(trans.Date, "center", bgColor, false)
	if err != nil {
		return nil, err
	}
	cells = append(cells, dateCell)
	cells = append(cells, mkCellDataString(trans.Name, "left", bgColor, false))
	return cells, nil
}

func addAmountCell(cells []*sheets.CellData, trans *models.Transaction, bgColor string) []*sheets.CellData {
	if isCheckingAccount(trans) {
		cells = addCheckingTransaction(trans, bgColor, cells)
	} else {
		cells = addCCTransaction(trans, bgColor, cells)
	}
	return cells
}

func addCategoryCells(cells []*sheets.CellData, trans *models.Transaction, columns []models.Column, transNameToColName map[string]string, totalsFormulas []string) []*sheets.CellData {
	// colOffset is because we've already taken care of cols A-G (0-6)
	colOffset := BankRegister
	for i := 0; i < len(columns)-colOffset; i++ {
		col := columns[colOffset+i]
		if isRegisterClearedOrDeltaColumn(i) {
			// first 3 columns are Register, Cleared & Delta. We copied the cell formulas above and are pasting here
			cells = append(cells, mkCellDataFormula(totalsFormulas[i], "right", col.Color, false))
		} else if isCreditCardTransaction(trans.Source, col.Name) {
			// enter a positive value in the credit card column
			cells = append(cells, mkCellDataDollars(trans.CreditCard, "left", "yellow", true))
		} else if isCorrectBudgetColumn(trans.Name, col.Name, transNameToColName) {
			// enter the value in the budget category column
			cells = append(cells, mkCellDataDollars(trans.Budget, "left", col.Color, true))
		} else {
			// this cell doesn't apply. Just create an empty cell.
			cells = append(cells, mkCellDataEmpty("left", col.Color, true))
		}
	}
	return cells
}

func getDollarsCellByIndex(values []interface{}, i int) float64 {
	return readDollarsValue(values[i])
}

func getNameField(values []interface{}) string {
	return readStringValue(values[Description])
}

func WriteJSONFile(fileName string, data interface{}) error {
	j, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("error: %s\n", err.Error())
	}
	err = os.WriteFile(fileName, j, 0644)
	if err != nil {
		return fmt.Errorf("error: %s\n", err.Error())
	}
	return nil
}
