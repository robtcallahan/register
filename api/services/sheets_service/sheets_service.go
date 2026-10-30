package sheets_service

import (
	"register/api/providers/sheets_provider"
	"register/pkg/config"
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
	CreditPurchase            = 6
	BankRegister              = 7
	Cleared                   = 8
	Delta                     = 9
	RegisterColumn            = "H"
	DeltaColumn               = "J"

	// Spreadsheet tab names.
	RegisterTabName = "Register"
	BudgetTabName   = "Budget"

	// Summary cells at the top of the Register tab.
	LastUpdateCell = "F1" // date stamp of the last update run

	CellDataString  = 1
	CellDataDollars = 2
	CellDataFormula = 3
	CellDataDate    = 4
)

type CellDataType int

type RegisterService struct {
	Provider sheets_provider.SheetsProviderInterface
	// IncomeSources are the configured income sources (config.json). Their
	// display names mark register rows as income deposits.
	IncomeSources []config.IncomeSource
}

func NewRegisterService(provider sheets_provider.SheetsProviderInterface) *RegisterService {
	return &RegisterService{
		Provider: provider,
	}
}

type BudgetService struct {
	Provider sheets_provider.SheetsProviderInterface
}

func NewBudgetService(provider sheets_provider.SheetsProviderInterface) *BudgetService {
	return &BudgetService{
		Provider: provider,
	}
}

type MonthlyService struct {
	Provider      sheets_provider.SheetsProviderInterface
	IncomeSources []config.IncomeSource
}

func NewMonthlyService(provider sheets_provider.SheetsProviderInterface, incomeSources []config.IncomeSource) *MonthlyService {
	return &MonthlyService{
		Provider:      provider,
		IncomeSources: incomeSources,
	}
}
