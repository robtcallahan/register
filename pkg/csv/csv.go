package csv

import (
	//"bytes"
	"fmt"
	"os"
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
}

// Client ...
type Client struct {
	FinanceDir string
	Banks      map[string]config.Bank
}

// FidelityVisa ...
type FidelityVisa struct {
	Date        string  `csv:"Date"`
	Transaction string  `csv:"Transaction"`
	Name        string  `csv:"Name"`
	Memo        string  `csv:"Memo"`
	Amount      float64 `csv:"Amount"`
}

// CostcoCitiVisa ...
type CostcoCitiVisa struct {
	Status      string  `csv:"Status"`
	Date        string  `csv:"Date"`
	Description string  `csv:"Description"`
	Debit       float64 `csv:"Debit"`
	Credit      float64 `csv:"Credit"`
	MemberName  string  `csv:"Member Name"`
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
	Date   string  `csv:"date"`
	Amount float64 `csv:"amount"`
	Dummy1 string  `csv:"dummy1"`
	Dummy2 string  `csv:"dummy2"`
	Name   string  `csv:"name"`
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

// New ...
func New(o ConfigOptions) *Client {
	return &Client{
		FinanceDir: o.FinanceDir,
		Banks:      o.Banks,
	}
}

func (c *Client) GetTransactions() ([]*models.Transaction, error) {
	var trans, t []*models.Transaction
	var err error

	//fmt.Printf("    Wells Fargo\n")
	//t, err = c.readWellsFargoCSV()
	//if err != nil {
	//	return nil, fmt.Errorf("could not read CSV file: %s", err.Error())
	//}

	fmt.Printf("    Fidelity\n")
	//trans = append(trans, t...)
	t, err = c.readFidelityCSV()
	if err != nil {
		return nil, fmt.Errorf("could not read CSV file: %s", err.Error())
	}
	trans = append(trans, t...)

	//fmt.Printf("    Chase\n")
	//trans = append(trans, t...)
	//t, err = c.readChaseCSV()
	//if err != nil {
	//	return nil, fmt.Errorf("could not read CSV file: %s", err.Error())
	//}

	//fmt.Printf("    Bank of America\n")
	//trans = append(trans, t...)
	//t, err = c.readBankOfAmericaCSV()
	//if err != nil {
	//	return nil, fmt.Errorf("could not read CSV file: %s", err.Error())
	//}
	//trans = append(trans, t...)

	return trans, nil
}

func (c *Client) readWellsFargoCSV() ([]*models.Transaction, error) {
	bank := c.Banks["wellsfargo"]
	csvFile := c.FinanceDir + "/" + bank.CSVFileName

	fileBytes, err := readFileContents(csvFile)

	// must add header row to the file if not present
	if !doWellsFargoHeadsExist(fileBytes) {
		err = addWellsFargoHeads(csvFile, fileBytes)
		if err != nil {
			return nil, fmt.Errorf("could not write csv file: %s", err.Error())
		}
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

func readWellsFargoCSVRows(csvFile string, bankId string) ([]*models.Transaction, error) {
	csvFilePtr, err := os.Open(csvFile)
	if err != nil {
		return nil, fmt.Errorf("could not open file %s: %s", csvFile, err.Error())
	}
	defer csvFilePtr.Close()

	var wellsFargo []*WellsFargo
	if err := gocsv.UnmarshalFile(csvFilePtr, &wellsFargo); err != nil {
		return nil, fmt.Errorf("could not unmarshal CSV file %s: %s", csvFile, err.Error())
	}
	return processWellsFargoData(wellsFargo, bankId), nil
}

func processWellsFargoData(wellsFargo []*WellsFargo, bankId string) []*models.Transaction {
	var trans []*models.Transaction
	for _, wf := range wellsFargo {
		t := &models.Transaction{
			Key:      fmt.Sprintf("%s:%s:%.2f", bankId, readDateValue(wf.Date), wf.Amount),
			Source:   "WellsFargo",
			Date:     readDateValue(wf.Date),
			Amount:   wf.Amount,
			BankName: wf.Name,
			Budget:   wf.Amount,
		}
		if wf.Amount < 0 {
			t.Withdrawal = -1 * wf.Amount
		} else {
			t.Deposit = wf.Amount
		}
		t = processCheck(wf.Name, t)
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
	dateHeader := ""
	for i := range []int8{0, 1, 2, 3, 4, 5} {
		dateHeader += string(fileBytes[i])
	}
	if dateHeader == "\"date\"" {
		return true
	}
	return false
}

func addWellsFargoHeads(fileName string, fileBytes []byte) error {
	headerBytes := []byte(`"date","amount","dummy1","dummy2","name"`)
	headerBytes = append(headerBytes, []byte("\n")...)
	fileBytes = append(headerBytes, fileBytes...)
	err := os.WriteFile(fileName, fileBytes, os.ModePerm)
	if err != nil {
		return fmt.Errorf("could not write csv file: %s", err.Error())
	}
	return nil
}

func readFidelityCSVRows(csvFile string, bankId string) ([]*models.Transaction, error) {
	csvFilePtr, err := os.Open(csvFile)
	if err != nil {
		return nil, fmt.Errorf("could not open file %s: %s", csvFile, err.Error())
	}
	defer csvFilePtr.Close()

	var fidelity []*FidelityVisa
	if err := gocsv.UnmarshalFile(csvFilePtr, &fidelity); err != nil {
		return nil, fmt.Errorf("could not unmarshal CSV file %s: %s", csvFile, err.Error())
	}

	return processFidelityData(fidelity, bankId), nil
}

func processFidelityData(fidelity []*FidelityVisa, bankId string) []*models.Transaction {
	var trans []*models.Transaction
	for _, f := range fidelity {
		re := regexp.MustCompile(`(payment\s+thank you)`)
		m := re.FindStringSubmatch(strings.ToLower(f.Name))
		if len(m) > 0 {
			continue
		}

		t := &models.Transaction{
			Key:            fmt.Sprintf("%s:%s:%.2f", bankId, readDateValue(f.Date), f.Amount),
			Source:         "Fidelity",
			Date:           readDateValue(f.Date),
			BankName:       f.Name,
			Amount:         f.Amount,      // amount stays as is
			CreditCard:     -1 * f.Amount, // convert to positive
			CreditPurchase: -1 * f.Amount, // convert to positive
			Budget:         f.Amount,      // already negative
		}
		trans = append(trans, t)
	}
	return trans
}

func readChaseCSVRows(csvFile string, bankId string) ([]*models.Transaction, error) {
	csvFilePtr, err := os.Open(csvFile)
	if err != nil {
		return nil, fmt.Errorf("could not open file %s: %s", csvFile, err.Error())
	}
	defer csvFilePtr.Close()

	var chase []*ChaseVisa
	if err := gocsv.UnmarshalFile(csvFilePtr, &chase); err != nil {
		return nil, fmt.Errorf("could not unmarshal CSV file %s: %s", csvFile, err.Error())
	}
	return processChaseData(chase, bankId), nil
}

func processChaseData(chase []*ChaseVisa, bankId string) []*models.Transaction {
	var trans []*models.Transaction
	for _, c := range chase {
		re := regexp.MustCompile(`(payment\s+thank you)`)
		m := re.FindStringSubmatch(strings.ToLower(c.Description))
		if len(m) > 0 {
			continue
		}

		t := &models.Transaction{
			Key:            fmt.Sprintf("%s:%s:%.2f", bankId, readDateValue(c.TransactionDate), c.Amount),
			Source:         "Chase",
			Date:           readDateValue(c.TransactionDate),
			Amount:         c.Amount,      // amount stays as is
			CreditCard:     -1 * c.Amount, // convert to positive
			CreditPurchase: -1 * c.Amount, // convert to positive
			Budget:         c.Amount,      // already negative
			BankName:       c.Description,
		}
		trans = append(trans, t)
	}
	return trans
}

func readBankOfAmericaCSVRows(csvFile string, bankId string) ([]*models.Transaction, error) {
	csvFilePtr, err := os.Open(csvFile)
	if err != nil {
		return nil, fmt.Errorf("could not open file %s: %s", csvFile, err.Error())
	}
	defer csvFilePtr.Close()

	var boa []*BankOfAmerica
	if err := gocsv.UnmarshalFile(csvFilePtr, &boa); err != nil {
		return nil, fmt.Errorf("could not unmarshal CSV file %s: %s", csvFile, err.Error())
	}

	return processBankOfAmericaData(boa, bankId), nil
}

func processBankOfAmericaData(boa []*BankOfAmerica, bankId string) []*models.Transaction {
	var trans []*models.Transaction
	for _, b := range boa {
		// skip CC payment transaction as these will show up as checking account payments
		re := regexp.MustCompile(`(ba electronic payment)`)
		m := re.FindStringSubmatch(strings.ToLower(b.Payee))
		if len(m) > 0 {
			continue
		}

		t := &models.Transaction{
			Key:        fmt.Sprintf("%s:%s:%.2f", bankId, readDateValue(b.PostedDate), b.Amount),
			Source:     bankId,
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
		mm, _ := strconv.Atoi(m[0][3])
		dd, _ := strconv.Atoi(m[0][4])
		yy, _ := strconv.Atoi(m[0][2])
		d = fmt.Sprintf("%02d/%02d/%02d", mm, dd, yy)
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
