package sheets_provider

import (
	"google.golang.org/api/sheets/v4"
)

var getRequestFunc func(spreadsheetID string, range_ string) (*sheets.ValueRange, error)

type getClientMock struct{}

// We are mocking the client method "Get"
func (c *getClientMock) Get(ssService *sheets.Service, spreadsheetID string, range_ string) (*sheets.ValueRange, error) {
	return getRequestFunc(spreadsheetID, range_)
}
