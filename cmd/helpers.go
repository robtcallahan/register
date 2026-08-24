package cmd

import "time"

const PlaidDateFormat = "2006-01-02"

func weeksAgo(weeks int) string {
	//t := time.Now().AddDate(0, -3, 0)
	// changing to the first of the year
	// TODO need to fix for next year
	t := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	return t.Format(PlaidDateFormat)
}

func today() string {
	return time.Now().Format(PlaidDateFormat)
}
