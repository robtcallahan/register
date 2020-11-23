package pkg

import (
	"flag"
	"fmt"
	"log"
	"regexp"

	"register/pkg/csv"
)

// Options ...
type Options struct {
	NumberOfCopies   *int
	RegisterStartRow *int64
	RegisterEndRow   *int64
	Banks            *string
	SpreadsheetID    *string
}

func formatYear(date string) string {
	re := regexp.MustCompile(`(\d+\/\d+)\/20(\d+)`)
	return re.ReplaceAllString(date, "${1}/${2}")
}

func parseOptions() Options {
	options := Options{
		NumberOfCopies:   flag.Int("n", 0, "The number of times to copy the last 2 rows"),
		RegisterStartRow: flag.Int64("s", config.RegisterStartRow, "The last used row in the spreadsheet"),
		RegisterEndRow:   flag.Int64("e", config.RegisterEndRow, "The last used row in the spreadsheet"),
		Banks:            flag.String("b", "wellsfargo,fidelity,costcocitivisa,chasevisa", "The desired bank CSV files to read"),
		SpreadsheetID:    flag.String("i", config.SpreadsheetID, "The Google spreadsheet id"),
	}
	flag.Parse()
	return options
}

func printRows(rows []*csv.FidelityVisa) {
	for _, row := range rows {
		fmt.Printf("%s %s %0.2f\n",
			row.Date,
			// row.Name.string,
			row.Amount,
		)
	}
}

func checkErrors(errs []error) {
	if len(errs) > 0 {
		for _, err := range errs {
			log.Printf("%s\n", err.Error())
		}
		log.Fatal("")
	}
}

func checkError(err error) {
	if err != nil {
		panic(err)
	}
}
