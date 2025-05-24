package sheets_service

import (
	"register/api/providers/sheets_provider"

	"google.golang.org/api/sheets/v4"
)

const (
	CreditCardColumnName      = "Credit Cards"
	CheckingAccountSourceName = "WellsFargo"
	Reconciled                = 0
	Source                    = 1
	Date                      = 2
	Description               = 3
	Withdrawals               = 4
	Deposits                  = 5
	CreditCards               = 6
	BankRegister              = 7
	Cleared                   = 8
	Delta                     = 9
	RegisterColumn            = "H"
	DeltaColumn               = "J"

	CellDataString  = 1
	CellDataDollars = 2
	CellDataFormula = 3
	CellDataDate    = 4
)

type CellDataType int

type SheetsService struct {
	service       *sheets.Service
	Provider      sheets_provider.SheetsProviderInterface
	SpreadsheetID string
	BudgetSheet   *BudgetSheet
	RegisterSheet *RegisterSheet
	Debug         bool
	Verbose       bool
}

func New(provider sheets_provider.SheetsProviderInterface) *SheetsService {
	return &SheetsService{
		Provider: provider,
	}
}
