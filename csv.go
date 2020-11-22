package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/gocarina/gocsv"
)

// Name ...
type Name struct {
	string
}

// FidelityVisa ...
type FidelityVisa struct {
	Date        string  `csv:"Date"`
	Transaction string  `csv:"Transaction"`
	Name        Name    `csv:"Name"`
	Memo        string  `csv:"Memo"`
	Amount      float32 `csv:"Amount"`
}

// CostcoCitiVisa ...
type CostcoCitiVisa struct {
	Status      string  `csv:"Status"`
	Date        string  `csv:"Date"`
	Description Name    `csv:"Description"`
	Debit       float32 `csv:"Debit"`
	Credit      float32 `csv:"Credit"`
	MemberName  string  `csv:"Member Name"`
}

// ChaseVisa ...
type ChaseVisa struct {
	TransactionDate string  `csv:"Transaction Date"`
	PostDate        string  `csv:"Post Date"`
	Description     Name    `csv:"Description"`
	Category        string  `csv:"Category"`
	Type            string  `csv:"Type"`
	Amount          float32 `csv:"Amount"`
}

// WellsFargo ...
type WellsFargo struct {
	Date   string  `csv:"Date"`
	Amount float32 `csv:"Amount"`
	Dummy1 string
	Dummy2 string
	Name   Name `csv:"Name"`
}

// CSVRow ...
type CSVRow struct {
	Key    string
	Source string
	Date   string
	Amount float32
	Name   string
}

func readWellsFargoCSV(bankFile string) []*CSVRow {
	// must a header row to the file
	fileBytes, err := ioutil.ReadFile(bankFile)
	if err != nil {
		log.Fatalf("could not read csv file: %s", err.Error())
	}
	headerBytes := []byte(`"Date","Amount","Dummy1","Dummy2","Name"`)
	headerBytes = append(headerBytes, []byte("\n")...)
	fileBytes = append(headerBytes, fileBytes...)
	ioutil.WriteFile(bankFile, fileBytes, os.ModePerm)

	csvFile, err := os.Open(bankFile)
	if err != nil {
		panic(err)
	}
	defer csvFile.Close()

	// read the csv file into an array of WellsFargo structs
	rows := []*WellsFargo{}
	if err := gocsv.UnmarshalFile(csvFile, &rows); err != nil {
		panic(err)
	}

	csvRows := []*CSVRow{}
	for _, row := range rows {
		name := row.Name.string
		if row.Name.string == "Venmo Payment" && row.Amount == -150 {
			name = "Margie Knight (Venmo)"
		} else if row.Name.string == "Venmo Payment" && row.Amount == -5 {
			name = "AA Meeting (Venmo)"
		}
		csv := &CSVRow{
			Key:    fmt.Sprintf("-:%s:%.2f", formatDate(row.Date), row.Amount),
			Source: "-",
			Date:   formatDate(row.Date),
			Amount: row.Amount,
			Name:   name,
		}
		csvRows = append(csvRows, csv)
	}
	return csvRows
}

func readFidelityCSV(bankFile string) []*CSVRow {
	csvFile, err := os.Open(bankFile)
	if err != nil {
		panic(err)
	}
	defer csvFile.Close()

	rows := []*FidelityVisa{}
	if err := gocsv.UnmarshalFile(csvFile, &rows); err != nil {
		panic(err)
	}
	csvRows := []*CSVRow{}
	for _, row := range rows {
		csv := &CSVRow{
			Key:    fmt.Sprintf("VISA:%s:%.2f", formatDate(row.Date), row.Amount),
			Source: "VISA",
			Date:   formatDate(row.Date),
			Amount: row.Amount,
			Name:   row.Name.string,
		}
		csvRows = append(csvRows, csv)
	}
	return csvRows
}

func readCitiCSV(bankFile string) []*CSVRow {
	csvFile, err := os.Open(bankFile)
	if err != nil {
		panic(err)
	}
	defer csvFile.Close()

	rows := []*CostcoCitiVisa{}
	if err := gocsv.UnmarshalFile(csvFile, &rows); err != nil {
		panic(err)
	}
	csvRows := []*CSVRow{}
	for _, row := range rows {
		csv := &CSVRow{
			Key:    fmt.Sprintf("CITI:%s:%.2f", formatDate(row.Date), -row.Debit),
			Source: "CITI",
			Date:   formatDate(row.Date),
			Amount: -row.Debit,
			Name:   row.Description.string,
		}
		csvRows = append(csvRows, csv)
	}
	return csvRows
}

func readChaseCSV(bankFile string) []*CSVRow {
	csvFile, err := os.Open(bankFile)
	if err != nil {
		panic(err)
	}
	defer csvFile.Close()

	rows := []*ChaseVisa{}
	if err := gocsv.UnmarshalFile(csvFile, &rows); err != nil {
		panic(err)
	}
	csvRows := []*CSVRow{}
	for _, row := range rows {
		csv := &CSVRow{
			Key:    fmt.Sprintf("CHASE:%s:%.2f", formatDate(row.TransactionDate), row.Amount),
			Source: "CHASE",
			Date:   formatDate(row.TransactionDate),
			Amount: row.Amount,
			Name:   row.Description.string,
		}
		csvRows = append(csvRows, csv)
	}
	return csvRows
}

// UnmarshalCSV Convert the CSV merchant name string to human readable
func (name *Name) UnmarshalCSV(csv string) (err error) {
	for substr, replace := range config.Merchants {
		if strings.Contains(csv, substr) {
			name.string = replace
			return nil
		}
	}
	name.string = csv
	return nil
}

func formatDate(date string) string {
	re := regexp.MustCompile(`(\d+)\/(\d+)\/(20)?(\d+)`)
	m := re.FindAllStringSubmatch(date, -1)
	mm, _ := strconv.Atoi(m[0][1])
	dd, _ := strconv.Atoi(m[0][2])
	yy, _ := strconv.Atoi(m[0][4])
	d := fmt.Sprintf("%02d/%02d/%02d", mm, dd, yy)
	return d
}
