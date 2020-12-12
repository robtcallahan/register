package sheets_provider

import (
	"fmt"
	"log"
	"register/pkg/auth"

	"register/api/clients/sheets_client"

	"google.golang.org/api/sheets/v4"
)

type SheetsProviderInterface interface {
	GetValues(range_ string) (*sheets.ValueRange, error)
	GetFormula(range_ string) (*sheets.ValueRange, error)
	GetSpreadsheet() (*sheets.Spreadsheet, error)
	BatchUpdate(updateReq *sheets.BatchUpdateSpreadsheetRequest) (*sheets.BatchUpdateSpreadsheetResponse, error)
	Update(writeRange string, vRange *sheets.ValueRange) (*sheets.UpdateValuesResponse, error)
}

type SheetsProvider struct {
	service       *sheets.Service
	spreadsheetID string
}

func New(spreadsheetID string) *SheetsProvider {
	client := auth.GetClient()
	service, err := sheets.New(client)
	if err != nil {
		panic(fmt.Errorf("unable to retrieve sheets client: %v", err))
	}
	return &SheetsProvider{
		service:       service,
		spreadsheetID: spreadsheetID,
	}
}

func (p *SheetsProvider) GetValues(range_ string) (*sheets.ValueRange, error) {
	resp, err := sheets_client.ClientStruct.Get(p.service, p.spreadsheetID, range_)
	if err != nil {
		log.Printf("error when trying to get cell data: %s", err.Error())
		return nil, err
	}
	return resp, nil
}

func (p *SheetsProvider) Update(writeRange string, vRange *sheets.ValueRange) (*sheets.UpdateValuesResponse, error) {
	resp, err := sheets_client.ClientStruct.Update(p.service, p.spreadsheetID, writeRange, vRange)
	if err != nil {
		log.Printf("error when try to update spreadsheet: %s", err.Error())
		return nil, err
	}
	return resp, nil
}

func (p *SheetsProvider) GetSpreadsheet() (*sheets.Spreadsheet, error) {
	resp, err := sheets_client.ClientStruct.GetSpreadsheet(p.service, p.spreadsheetID)
	if err != nil {
		log.Printf("error when try to get spreadsheet: %s", err.Error())
		return nil, err
	}
	return resp, nil
}

func (p *SheetsProvider) GetFormula(range_ string) (*sheets.ValueRange, error) {
	resp, err := sheets_client.ClientStruct.GetFormula(p.service, p.spreadsheetID, range_)
	if err != nil {
		log.Printf("error when trying to get formula cells: %s", err.Error())
		return nil, err
	}
	return resp, nil
}

func (p *SheetsProvider) BatchUpdate(updateReq *sheets.BatchUpdateSpreadsheetRequest) (*sheets.BatchUpdateSpreadsheetResponse, error) {
	resp, err := sheets_client.ClientStruct.BatchUpdate(p.service, p.spreadsheetID, updateReq)
	if err != nil {
		log.Printf("error when try to batch update spreadsheet: %s", err.Error())
		return nil, err
	}
	return resp, nil
}

