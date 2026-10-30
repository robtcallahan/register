package sheets_service

// Reading values back out of sheet rows: parsing and field access.

import (
	"fmt"
	"regexp"
	"register/pkg/models"
	"strconv"
	"strings"
)

func formatYear(date string) string {
	re := regexp.MustCompile(`^(\d+/\d+)/\d*(\d{2})$`)
	return re.ReplaceAllString(date, "${1}/${2}")
}

func readStringValue(text any) string {
	return fmt.Sprintf("%v", text)
}

func readDollarsValue(value any) (float64, error) {
	dollars := readStringValue(value)
	re := regexp.MustCompile(`[\s$,]`)
	dollars = re.ReplaceAllString(dollars, "")
	if dollars == "-" || dollars == "" {
		return 0, nil
	}

	re = regexp.MustCompile(`[()]`)
	if re.Match([]byte(dollars)) {
		dollars = "-" + re.ReplaceAllString(dollars, "")
	}

	f, err := strconv.ParseFloat(dollars, 64)
	if err != nil {
		return 0, fmt.Errorf("parseFloat error: %w", err)
	}
	return f, nil
}

func getStringField(values []any, i int) string {
	if i >= 0 && i < len(values) {
		return readStringValue(values[i])
	}
	return ""
}

// getRowValues safely returns the i-th row of a value range, or nil if i is out of range.
func getRowValues(values [][]any, i int) []any {
	if i >= 0 && i < len(values) {
		return values[i]
	}
	return nil
}

func getSourceField(values []any) string {
	if getStringField(values, Source) == "" {
		return strings.ToLower(CheckingAccountSourceName)
	}
	return strings.ToLower(getStringField(values, Source))
}

func getDateField(values []any) string {
	dateString := getStringField(values, Date)
	if dateString == "" {
		return ""
	}
	return models.ReadDateValue(dateString)
}

func getAmountString(values []any) string {
	v := getStringField(values, Withdrawals)
	if v == "" {
		v = getStringField(values, Deposits)
	}
	if v == "" {
		v = getStringField(values, CreditPurchase)
	}

	// Accounting format: (123.45) means negative
	negative := strings.Contains(v, "(")

	// Strip parentheses, spaces, $, commas
	re := regexp.MustCompile(`[()\s$,]`)
	cleaned := re.ReplaceAllString(v, "")

	// Bare dash, empty, or unparseable → zero
	if cleaned == "" || cleaned == "-" {
		return "0.00"
	}

	fl, err := strconv.ParseFloat(cleaned, 64)
	if err != nil {
		return "0.00"
	}

	if negative {
		fl = -fl
	}
	if fl == 0 {
		fl = 0 // normalize -0.0
	}

	return fmt.Sprintf("%.2f", fl)
}

func getTransactionKey(values []any) string {
	amtStr := getAmountString(values)
	if amt, err := strconv.ParseFloat(amtStr, 64); err == nil {
		return models.TransactionKey(getSourceField(values), getDateField(values), amt)
	}
	return fmt.Sprintf("%s:%s:%s", getSourceField(values), getDateField(values), amtStr)
}

func getDollarsCellByIndex(values []any, i int) (float64, error) {
	if i >= 0 && i < len(values) {
		return readDollarsValue(values[i])
	}
	return 0, nil
}

func getNameField(values []any) string {
	return getStringField(values, Description)
}
