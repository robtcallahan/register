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
	Date        string `csv:"DATE"`
	Description string `csv:"DESCRIPTION"`
	Amount      string `csv:"AMOUNT"`
	CheckNum    string `csv:"CHECK #"`
	Status      string `csv:"STATUS"`
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

func (c *Client) GetTransactions(bankIDs []string) ([]*models.Transaction, error) {
	var trans, t []*models.Transaction
	var err error

	for _, bankID := range bankIDs {
		if bankID == "wellsfargo" {
			fmt.Printf("    Wells Fargo\n")
			t, err = c.readWellsFargoCSV()
			if err != nil {
				return nil, fmt.Errorf("could not read CSV file: %s", err.Error())
			}
			trans = append(trans, t...)
		} else if bankID == "fidelity" {
			fmt.Printf("    Fidelity\n")
			t, err = c.readFidelityCSV()
			if err != nil {
				return nil, fmt.Errorf("could not read CSV file: %s", err.Error())
			}
			trans = append(trans, t...)
		} else if bankID == "chase" {
			fmt.Printf("    Chase\n")
			trans = append(trans, t...)
			t, err = c.readChaseCSV()
			if err != nil {
				return nil, fmt.Errorf("could not read CSV file: %s", err.Error())
			}
			trans = append(trans, t...)
		} else if bankID == "boa" {
			fmt.Printf("    Bank of America\n")
			trans = append(trans, t...)
			t, err = c.readBankOfAmericaCSV()
			if err != nil {
				return nil, fmt.Errorf("could not read CSV file: %s", err.Error())
			}
			trans = append(trans, t...)
		} else {
			return nil, fmt.Errorf("unknown bankID: %s", bankID)
		}
	}
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
		amount := 0.0
		if s, err := strconv.ParseFloat(wf.Amount, 64); err == nil {
			amount = s
		}

		t := &models.Transaction{
			Key:      fmt.Sprintf("%s:%s:%.2f", bankId, readDateValue(wf.Date), -amount),
			Source:   "WellsFargo",
			Date:     readDateValue(wf.Date),
			Amount:   amount,
			BankName: wf.Description,
			Budget:   amount,
		}
		if amount < 0 {
			t.Withdrawal = 0
			t.Deposit = -1 * amount
			t.Budget = -1 * amount
		} else {
			t.Withdrawal = amount
			t.Deposit = 0
			t.Budget = amount
		}
		t = processCheck(wf.CheckNum, t)
		trans = append(trans, t)
	}
	return trans
}

func processCheck(name string, t *models.Transaction) *models.Transaction {
	re := regexp.MustCompile(`CHECK # (\d+)`)
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
	if dateHeader == "\"DATE\"" {
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
		t := &models.Transaction{
			//Key:            fmt.Sprintf("%s:%s:%.2f", bankId, readDateValue(f.Date), f.Amount),
			Source:         "Fidelity",
			Date:           readDateValue(f.Date),
			BankName:       f.Name,
			Amount:         f.Amount,      // amount stays as is
			CreditPurchase: -1 * f.Amount, // convert to positive
			CreditCard:     -1 * f.Amount, // convert to positive
			Budget:         f.Amount,      // already negative
		}
		t.Key = fmt.Sprintf("%s:%s:%.2f", bankId, t.Date, t.CreditCard)
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
			Key:            fmt.Sprintf("%s:%s:%.2f", bankId, readDateValue(c.TransactionDate), -c.Amount),
			Source:         "Chase",
			Date:           readDateValue(c.TransactionDate),
			Amount:         c.Amount,
			CreditPurchase: c.Amount,
			CreditCard:     -1 * c.Amount,
			Budget:         c.Amount,
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
	re := regexp.MustCompile(`(\d\d)/(\d\d)/(20)?(\d\d)`)
	m := re.FindAllStringSubmatch(date, -1)
	if m != nil {
		mm, _ := strconv.Atoi(m[0][1])
		dd, _ := strconv.Atoi(m[0][2])
		yy, _ := strconv.Atoi(m[0][4])
		d = fmt.Sprintf("%02d/%02d/%02d", mm, dd, yy)
	} else {
		re := regexp.MustCompile(`(20)?(\d\d)-(\d\d)-(\d\d)`)
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
