package sheets_service

import (
	"fmt"
	"log"
	"register/api/providers/sheets_provider"

	"google.golang.org/api/sheets/v4"
)

type sheetsService struct {
	SpreadsheetsService *sheets.Service
}

type sheetsServiceInterface interface {
	GetValues(spreadsheetID string, range_ string) (*sheets.ValueRange, error)
}

var SheetsService sheetsServiceInterface = &sheetsService{}

// RegisterEntry ...
type RegisterEntry struct {
	RowID        int64
	Key          string
	Reconciled   string
	Source       string
	Date         string
	Name         string
	Amount       float64
	Withdrawal   float64
	Deposit      float64
	CreditCard   float64
	BankRegister float64
	Cleared      float64
	Delta        float64
}

func New(service *sheets.Service) *sheetsService {
	return &sheetsService{
		SpreadsheetsService: service,
	}
}

func (s *sheetsService) GetValues(spreadsheetID string, range_ string) ([]*RegisterEntry, map[string]bool, [][]interface{}) {
	sheetsProvider := sheets_provider.New(s.SpreadsheetsService)
	resp, err := sheetsProvider.GetValues(spreadsheetID, range_)
	if err != nil {
		log.Fatalf("could not get sheet values: %s\n", err.Error())
	}

	rangeValues := resp.Values
	if len(rangeValues) == 0 {
		log.Fatalf("No data found")
	}

	// determine last used row in the spreadsheet
	lastRow := int64(len(rangeValues)) + rs.StartRow - 2
	keysMap := make(map[string]bool)
	var register []*RegisterEntry

	for i := 0; int64(i) <= rs.LastRow; i += 2 {
		values := rangeValues[i]

		name := rs.getNameField(values)
		source := rs.getSourceField(values)
		date := rs.getDateField(values)
		amount := rs.getAmount(values)

		key := fmt.Sprintf("%s:%s:%s", source, date, amount)
		keysMap[key] = true

		c := &RegisterEntry{
			Key:          key,
			RowID:        rs.StartRow + int64(i),
			Reconciled:   getStringField(values, rs.Config.RegisterIndexes["Withdrawals"]),
			Source:       source,
			Date:         date,
			Name:         name,
			Withdrawal:   rs.GetRegisterField(values, rs.Config.RegisterIndexes["Withdrawals"]),
			Deposit:      rs.GetRegisterField(values, rs.Config.RegisterIndexes["Deposits"]),
			CreditCard:   rs.GetRegisterField(values, rs.Config.RegisterIndexes["CreditCards"]),
			BankRegister: rs.GetRegisterField(values, rs.Config.RegisterIndexes["BankRegister"]),
			Cleared:      rs.GetRegisterField(values, rs.Config.RegisterIndexes["Cleared"]),
			Delta:        rs.GetRegisterField(values, rs.Config.RegisterIndexes["Delta"]),
		}
		register = append(register, c)

		if values[rs.Config.RegisterIndexes["Date"]] == "" {
			rs.FirstRowToUpdate = int64(i) + rs.StartRow - 1
			break
		}
	}
	return register, keysMap, rangeValues
}

