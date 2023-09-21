package sheets_client

import (
	"errors"
	"fmt"

	"google.golang.org/api/sheets/v4"
)

type ClientInterface interface {
	Get(ssService *sheets.Service, spreadsheetID string, range_ string) (*sheets.ValueRange, error)
	GetFormula(ssService *sheets.Service, spreadsheetId string, range_ string) (*sheets.ValueRange, error)
	GetSpreadsheet(ssService *sheets.Service, spreadsheetId string) (*sheets.Spreadsheet, error)
	BatchUpdate(ssService *sheets.Service, spreadsheetID string, updateReq *sheets.BatchUpdateSpreadsheetRequest) (*sheets.BatchUpdateSpreadsheetResponse, error)
	Update(ssService *sheets.Service, spreadsheetId string, writeRange string, vRange *sheets.ValueRange) (*sheets.UpdateValuesResponse, error)
}

type clientStruct struct{}

var ClientStruct ClientInterface = &clientStruct{}

func (c *clientStruct) Get(ssService *sheets.Service, spreadsheetId string, range_ string) (*sheets.ValueRange, error) {
	resp, err := ssService.Spreadsheets.Values.Get(spreadsheetId, range_).Do()
	return resp, err
}

func (c *clientStruct) GetFormula(ssService *sheets.Service, spreadsheetId string, range_ string) (*sheets.ValueRange, error) {
	call := ssService.Spreadsheets.Values.BatchGet(spreadsheetId)
	call.ValueRenderOption("FORMULA")
	call.Ranges(range_)
	resp, err := call.Do()
	if resp == nil {
		return nil, errors.New("empty response")
	}
	return resp.ValueRanges[0], err
}

func (c *clientStruct) GetSpreadsheet(ssService *sheets.Service, spreadsheetId string) (*sheets.Spreadsheet, error) {
	resp, err := ssService.Spreadsheets.Get(spreadsheetId).Do()
	return resp, err
}

func (c *clientStruct) BatchUpdate(ssService *sheets.Service, spreadsheetID string, updateReq *sheets.BatchUpdateSpreadsheetRequest) (*sheets.BatchUpdateSpreadsheetResponse, error) {
	resp, err := ssService.Spreadsheets.BatchUpdate(spreadsheetID, updateReq).Do()
	return resp, err
}

func (c *clientStruct) Update(ssService *sheets.Service, spreadsheetId string, writeRange string, vRange *sheets.ValueRange) (*sheets.UpdateValuesResponse, error) {
	call := ssService.Spreadsheets.Values.Update(spreadsheetId, writeRange, vRange)
	call.ValueInputOption("USER_ENTERED")
	resp, err := call.Do()
	if err != nil {
		return resp, fmt.Errorf("unable to write cell data: %s", err.Error())
	}
	return resp, err
}
