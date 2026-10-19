package csv

import (
	"bytes"
	stdcsv "encoding/csv"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"register/pkg/config"
	"register/pkg/models"
	"strconv"
	"strings"

	"github.com/gocarina/gocsv"
)

// ConfigOptions ...
type ConfigOptions struct {
	FinanceDir string
	Banks      map[string]config.Bank
	BankIDs    []string
}

// Client ...
type Client struct {
	FinanceDir string
	Banks      map[string]config.Bank
	BankIDs    []string
}

// FidelityVisa ...
type FidelityVisa struct {
	Date        string  `csv:"Date"`
	Transaction string  `csv:"Transaction"`
	Name        string  `csv:"Name"`
	Memo        string  `csv:"Memo"`
	Amount      float64 `csv:"Amount"`
}

// ChaseVisa ...
type ChaseVisa struct {
	TransactionDate string  `csv:"Transaction Date"`
	PostDate        string  `csv:"Post Date"`
	Description     string  `csv:"Description"`
	Category        string  `csv:"Category"`
	Type            string  `csv:"Type"`
	Amount          float64 `csv:"Amount"`
}

// BankOfAmerica ...
type BankOfAmerica struct {
	PostedDate      string  `csv:"Posted Date"`
	ReferenceNumber string  `csv:"Reference Number"`
	Payee           string  `csv:"Payee"`
	Address         string  `csv:"Address"`
	Amount          float64 `csv:"Amount"`
}

// WellsFargo ...
type WellsFargo struct {
	Date        string  `csv:"DATE"`
	Description string  `csv:"DESCRIPTION"`
	Amount      float64 `csv:"AMOUNT"`
	CheckNumber string  `csv:"CHECK #"`
	Status      string  `csv:"STATUS"`
}

// Row ...
type Row struct {
	Key    string
	Source string
	Date   string
	Amount float64
	Name   string
}

// myReadFile is a function variable that can be reassigned to handle mocking for testing
var myReadFile = os.ReadFile

// normalizeHeader canonicalizes a CSV header label for matching: surrounding
// whitespace is ignored and matching is case-insensitive.
func normalizeHeader(h string) string {
	return strings.ToLower(strings.TrimSpace(h))
}

// csvHeaders returns the `csv:` struct tags of a gocsv-mapped struct: the
// headers the bank's file must contain.
func csvHeaders(sample any) []string {
	t := reflect.TypeOf(sample)
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	var headers []string
	for field := range t.Fields() {
		if tag, ok := field.Tag.Lookup("csv"); ok && tag != "" && tag != "-" {
			headers = append(headers, tag)
		}
	}
	return headers
}

// checkCSVHeaders reads a CSV file's header row and verifies it contains every
// header the given gocsv struct expects. gocsv maps columns by header name
// but stays silent when one is missing (the field just keeps its zero value),
// so a renamed bank column would silently zero a field; this makes it loud.
func checkCSVHeaders(csvFile string, sample any) error {
	f, err := os.Open(csvFile)
	if err != nil {
		return fmt.Errorf("could not open file %s: %w", csvFile, err)
	}
	defer f.Close()
	r := stdcsv.NewReader(f)
	header, err := r.Read()
	if err != nil {
		return fmt.Errorf("could not read header row of %s: %w", csvFile, err)
	}
	have := make(map[string]bool)
	for _, h := range header {
		have[normalizeHeader(h)] = true
	}
	var missing []string
	for _, want := range csvHeaders(sample) {
		if !have[normalizeHeader(want)] {
			missing = append(missing, want)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("CSV file %s is missing expected headers: %s", csvFile, strings.Join(missing, ", "))
	}
	return nil
}

// New ...
func New(o ConfigOptions) *Client {
	return &Client{
		FinanceDir: o.FinanceDir,
		Banks:      o.Banks,
		BankIDs:    o.BankIDs,
	}
}

func (c *Client) GetTransactions() ([]*models.Transaction, error) {
	var csvTrans []*models.Transaction

	for _, bankID := range c.BankIDs {
		switch bankID {
		case "fidelity":
			{
				fmt.Printf("    Fidelity\n")
				trans, err := c.readFidelityCSV()
				if err != nil {
					return nil, fmt.Errorf("could not read CSV file: %w", err)
				}
				csvTrans = append(csvTrans, trans...)
			}
		case "wellsfargo":
			{
				fmt.Printf("    Wells Fargo\n")
				trans, err := c.readWellsFargoCSV()
				if err != nil {
					return nil, fmt.Errorf("could not read CSV file: %w", err)
				}
				csvTrans = append(csvTrans, trans...)
			}
		case "chase":
			{
				fmt.Printf("    Chase\n")
				trans, err := c.readChaseCSV()
				if err != nil {
					return nil, fmt.Errorf("could not read CSV file: %w", err)
				}
				csvTrans = append(csvTrans, trans...)
			}
		case "boa":
			{
				fmt.Printf("    Bank of America\n")
				trans, err := c.readBankOfAmericaCSV()
				if err != nil {
					return nil, fmt.Errorf("could not read CSV file: %w", err)
				}
				csvTrans = append(csvTrans, trans...)
			}
		}
	}

	return csvTrans, nil
}

func (c *Client) readWellsFargoCSV() ([]*models.Transaction, error) {
	bank := c.Banks["wellsfargo"]
	csvFile := c.FinanceDir + "/" + bank.CSVFileName

	fileBytes, err := readFileContents(csvFile)

	// must add header row to the file if not present
	if !doWellsFargoHeadsExist(fileBytes) {
		err = addWellsFargoHeads(csvFile, fileBytes)
		if err != nil {
			return nil, fmt.Errorf("could not write csv file: %w", err)
		}
	} else if err := checkCSVHeaders(csvFile, WellsFargo{}); err != nil {
		return nil, err
	}

	return readWellsFargoCSVRows(csvFile, bank.ID)
}

func (c *Client) readFidelityCSV() ([]*models.Transaction, error) {
	bank := c.Banks["fidelity"]
	trans, err := readFidelityCSVRows(c.FinanceDir+"/"+bank.CSVFileName, bank.ID)
	return trans, err
}

func (c *Client) readChaseCSV() ([]*models.Transaction, error) {
	bank := c.Banks["chase"]
	trans, err := readChaseCSVRows(c.FinanceDir+"/"+bank.CSVFileName, bank.ID)
	return trans, err
}

func (c *Client) readBankOfAmericaCSV() ([]*models.Transaction, error) {
	bank := c.Banks["boa"]
	trans, err := readBankOfAmericaCSVRows(c.FinanceDir+"/"+bank.CSVFileName, bank.ID)
	return trans, err
}

func readWellsFargoCSVRows(csvFile string, bankID string) ([]*models.Transaction, error) {
	csvFilePtr, err := os.Open(csvFile)
	if err != nil {
		return nil, fmt.Errorf("could not open file %s: %w", csvFile, err)
	}
	defer csvFilePtr.Close()

	var wellsFargo []*WellsFargo
	if err := gocsv.UnmarshalFile(csvFilePtr, &wellsFargo); err != nil {
		return nil, fmt.Errorf("could not unmarshal CSV file %s: %w", csvFile, err)
	}
	return processWellsFargoData(wellsFargo, bankID), nil
}

func processWellsFargoData(wellsFargo []*WellsFargo, bankID string) []*models.Transaction {
	var trans []*models.Transaction
	for _, wf := range wellsFargo {
		t := &models.Transaction{
			Key:      models.TransactionKey(bankID, readDateValue(wf.Date), wf.Amount),
			Source:   "WellsFargo",
			Date:     readDateValue(wf.Date),
			Amount:   wf.Amount,
			BankName: wf.Description,
			Budget:   wf.Amount,
		}
		if wf.Amount < 0 {
			t.Withdrawal = -1 * wf.Amount
		} else {
			t.Deposit = wf.Amount
		}
		t = processCheck(wf.CheckNumber, t)
		trans = append(trans, t)
	}
	return trans
}

func processCheck(name string, t *models.Transaction) *models.Transaction {
	re := regexp.MustCompile(`CHECK # (\d\d\d\d?)`)
	m := re.FindStringSubmatch(name)
	if len(m) > 0 {
		t.Source = m[1]
		t.IsCheck = true
		t.Name = "CHECK"
		t.BankName = "CHECK"
	}
	return t
}

func doWellsFargoHeadsExist(fileBytes []byte) bool {
	return bytes.HasPrefix(fileBytes, []byte("\"DATE\""))
}

func addWellsFargoHeads(fileName string, fileBytes []byte) error {
	headerBytes := []byte(`"date","amount","dummy1","dummy2","name"`)
	headerBytes = append(headerBytes, []byte("\n")...)
	fileBytes = append(headerBytes, fileBytes...)
	err := os.WriteFile(fileName, fileBytes, os.ModePerm)
	if err != nil {
		return fmt.Errorf("could not write csv file: %w", err)
	}
	return nil
}

func readFidelityCSVRows(csvFile string, bankID string) ([]*models.Transaction, error) {
	if err := checkCSVHeaders(csvFile, FidelityVisa{}); err != nil {
		return nil, err
	}
	csvFilePtr, err := os.Open(csvFile)
	if err != nil {
		return nil, fmt.Errorf("could not open file %s: %w", csvFile, err)
	}
	defer csvFilePtr.Close()

	var fidelity []*FidelityVisa
	if err := gocsv.UnmarshalFile(csvFilePtr, &fidelity); err != nil {
		return nil, fmt.Errorf("could not unmarshal CSV file %s: %w", csvFile, err)
	}

	return processFidelityData(fidelity, bankID), nil
}

func processFidelityData(fidelity []*FidelityVisa, bankID string) []*models.Transaction {
	var trans []*models.Transaction
	for _, f := range fidelity {
		t := &models.Transaction{
			Source:         "Fidelity",
			Date:           readDateValue(f.Date),
			BankName:       f.Name,
			Amount:         f.Amount,      // amount stays as is
			CreditPurchase: -1 * f.Amount, // convert to positive
			CreditCard:     -1 * f.Amount, // convert to positive
			Budget:         f.Amount,      // already negative
		}
		t.Key = models.TransactionKey(bankID, t.Date, t.CreditCard)
		trans = append(trans, t)
	}
	return trans
}

func readChaseCSVRows(csvFile string, bankID string) ([]*models.Transaction, error) {
	if err := checkCSVHeaders(csvFile, ChaseVisa{}); err != nil {
		return nil, err
	}
	csvFilePtr, err := os.Open(csvFile)
	if err != nil {
		return nil, fmt.Errorf("could not open file %s: %w", csvFile, err)
	}
	defer csvFilePtr.Close()

	var chase []*ChaseVisa
	if err := gocsv.UnmarshalFile(csvFilePtr, &chase); err != nil {
		return nil, fmt.Errorf("could not unmarshal CSV file %s: %w", csvFile, err)
	}
	return processChaseData(chase, bankID), nil
}

func processChaseData(chase []*ChaseVisa, bankID string) []*models.Transaction {
	var trans []*models.Transaction
	re := regexp.MustCompile(`(payment\s+thank you)`)
	for _, c := range chase {
		m := re.FindStringSubmatch(strings.ToLower(c.Description))
		if len(m) > 0 {
			continue
		}

		t := &models.Transaction{
			Key:            models.TransactionKey(bankID, readDateValue(c.TransactionDate), -1*c.Amount),
			Source:         "Chase",
			Date:           readDateValue(c.TransactionDate),
			Amount:         c.Amount, // amount stays as is
			CreditPurchase: c.Amount, // convert to positive to be added to CreditCard column (like Deposit/Withdrawal)
			CreditCard:     c.Amount, // convert to positive
			Budget:         c.Amount, // already negative to be subtracted from budget column
			BankName:       c.Description,
		}
		trans = append(trans, t)
	}
	return trans
}

func readBankOfAmericaCSVRows(csvFile string, bankID string) ([]*models.Transaction, error) {
	if err := checkCSVHeaders(csvFile, BankOfAmerica{}); err != nil {
		return nil, err
	}
	csvFilePtr, err := os.Open(csvFile)
	if err != nil {
		return nil, fmt.Errorf("could not open file %s: %w", csvFile, err)
	}
	defer csvFilePtr.Close()

	var boa []*BankOfAmerica
	if err := gocsv.UnmarshalFile(csvFilePtr, &boa); err != nil {
		return nil, fmt.Errorf("could not unmarshal CSV file %s: %w", csvFile, err)
	}

	return processBankOfAmericaData(boa, bankID), nil
}

func processBankOfAmericaData(boa []*BankOfAmerica, bankID string) []*models.Transaction {
	var trans []*models.Transaction
	re := regexp.MustCompile(`(ba electronic payment)`)
	for _, b := range boa {
		// skip CC payment transaction as these will show up as checking account payments
		m := re.FindStringSubmatch(strings.ToLower(b.Payee))
		if len(m) > 0 {
			continue
		}

		t := &models.Transaction{
			Key:        models.TransactionKey(bankID, readDateValue(b.PostedDate), b.Amount),
			Source:     bankID,
			Date:       readDateValue(b.PostedDate),
			Amount:     -b.Amount,
			CreditCard: -b.Amount,
			BankName:   b.Payee,
		}
		trans = append(trans, t)
	}
	return trans
}

func readDateValue(date string) string {
	var d string
	re := regexp.MustCompile(`(\d+)/(\d+)/(20)?(\d+)`)
	m := re.FindAllStringSubmatch(date, -1)
	if m != nil {
		mm, _ := strconv.Atoi(m[0][1])
		dd, _ := strconv.Atoi(m[0][2])
		yy, _ := strconv.Atoi(m[0][4])
		d = fmt.Sprintf("%02d/%02d/%02d", mm, dd, yy)
	} else {
		re := regexp.MustCompile(`(20)?(\d+)-(\d+)-(\d+)`)
		m := re.FindAllStringSubmatch(date, -1)
		if m != nil {
			mm, _ := strconv.Atoi(m[0][3])
			dd, _ := strconv.Atoi(m[0][4])
			yy, _ := strconv.Atoi(m[0][2])
			d = fmt.Sprintf("%02d/%02d/%02d", mm, dd, yy)
		}
	}
	return d
}

func readFileContents(path string) ([]byte, error) {
	var err error
	if contents, err := myReadFile(path); err == nil {
		return contents, nil
	}
	return nil, err
}
