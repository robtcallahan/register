package models

import (
	"fmt"
	"regexp"
	"strconv"
)

var yearLastRE = regexp.MustCompile(`^(\d{1,2})/(\d{1,2})/(\d{4}|\d{2})$`)
var yearFirstRE = regexp.MustCompile(`^(\d{4})-(\d{1,2})-(\d{1,2})$`)

func ReadDateValue(dateStr string) string {
	if m := yearLastRE.FindStringSubmatch(dateStr); m != nil {
		mm, _ := strconv.Atoi(m[1])
		dd, _ := strconv.Atoi(m[2])
		yy, _ := strconv.Atoi(m[3])
		return fmt.Sprintf("%02d/%02d/%02d", mm, dd, yy%100)
	}
	if m := yearFirstRE.FindStringSubmatch(dateStr); m != nil {
		yy, _ := strconv.Atoi(m[1])
		mm, _ := strconv.Atoi(m[2])
		dd, _ := strconv.Atoi(m[3])
		return fmt.Sprintf("%02d/%02d/%02d", mm, dd, yy%100)
	}
	return ""
}
