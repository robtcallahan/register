package sheets_provider

import (
	"google.golang.org/api/sheets/v4"
)

var getRequestFunc func(spreadsheetId string, range_ string) (*sheets.ValueRange, error)

type getClientMock struct{}

//We are mocking the client method "Get"
func (c *getClientMock) Get(ssService *sheets.Service, spreadsheetId string, range_ string) (*sheets.ValueRange, error){
	return getRequestFunc(spreadsheetId, range_)
}

