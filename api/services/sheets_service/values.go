package sheets_service

// Reading values back out of sheet rows: parsing and field access.

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"register/pkg/models"
)

func formatYear(date string) string {
	re := regexp.MustCompile(`^(\d+/\d+)/\d*(\d{2})$`)
	return re.ReplaceAllString(date, "${1}/${2}")
}

func readStringValue(text interface{}) string {
	return fmt.Sprintf("%v", text)
}

func readDollarsValue(value interface{}) (float64, error) {
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
		return 0, fmt.Errorf("parseFloat error: %s", err.Error())
	}
	return f, nil
}

func readDateValue(dateStr interface{}) string {
	date := readStringValue(dateStr)
	re := regexp.MustCompile(`(\d+)/(\d+)/(20)?(\d+)`)
	m := re.FindAllStringSubmatch(date, -1)
	if len(m) == 0 {
		return ""
	}
	mm, _ := strconv.Atoi(m[0][1])
	dd, _ := strconv.Atoi(m[0][2])
	yy, _ := strconv.Atoi(m[0][4])
	d := fmt.Sprintf("%02d/%02d/%02d", mm, dd, yy)
	return d
}

func getStringField(values []interface{}, i int) string {
	if i >= 0 && i < len(values) {
		return readStringValue(values[i])
	}
	return ""
}

// getRowValues safely returns the i-th row of a value range, or nil if i is out of range.
func getRowValues(values [][]interface{}, i int) []interface{} {
	if i >= 0 && i < len(values) {
		return values[i]
	}
	return nil
}

func getSourceField(values []interface{}) string {
	if getStringField(values, Source) == "" {
		return strings.ToLower(CheckingAccountSourceName)
	}
	return strings.ToLower(getStringField(values, Source))
}

func getDateField(values []interface{}) string {
	dateString := getStringField(values, Date)
	if dateString == "" {
		return ""
	}
	return readDateValue(dateString)
}

func getAmountString(values []interface{}) string {
	amt := ""
	v := ""
	if v = getStringField(values, Withdrawals); v != "" {
		amt = v
	} else if v = getStringField(values, Deposits); v != "" {
		amt = v
	} else if v = getStringField(values, CreditCards); v != "" {
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
	amtStr := getAmountString(values)
	if amt, err := strconv.ParseFloat(amtStr, 64); err == nil {
		return models.TransactionKey(getSourceField(values), getDateField(values), amt)
	}
	return fmt.Sprintf("%s:%s:%s", getSourceField(values), getDateField(values), amtStr)
}

func getDollarsCellByIndex(values []interface{}, i int) (float64, error) {
	if i >= 0 && i < len(values) {
		return readDollarsValue(values[i])
	}
	return 0, nil
}

func getNameField(values []interface{}) string {
	return getStringField(values, Description)
}
