package sheets_provider

import (
	"fmt"
	"log"

	"register/api/clients/sheetsclient"

	"google.golang.org/api/sheets/v4"
)

type sheetsProvider struct {
	SpreadsheetsService *sheets.Service
}

type sheetsServiceInterface interface {
	GetValues(spreadsheetID string, range_ string) (*sheets.ValueRange, error)
}

var SheetsProvider sheetsServiceInterface = &sheetsProvider{}

func New(service *sheets.Service) *sheetsProvider {
	return &sheetsProvider{
		SpreadsheetsService: service,
	}
}

func (p *sheetsProvider) GetValues(spreadsheetId string, range_ string) (*sheets.ValueRange, error) {
	resp, err := sheetsclient.ClientStruct.Get(p.SpreadsheetsService, spreadsheetId, range_)
	if err != nil {
		log.Println(fmt.Sprintf("error when trying to get sheet api: %s", err.Error()))
		return nil, err
	}
	return resp, nil
}
