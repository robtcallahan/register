package cmd

import "time"

const PlaidDateFormat = "2006-01-02"

func weeksAgo(weeks int) string {
	t := time.Now().AddDate(0, 0, -14)
	return t.Format(PlaidDateFormat)
}

func today() string {
	return time.Now().Format(PlaidDateFormat)
}
