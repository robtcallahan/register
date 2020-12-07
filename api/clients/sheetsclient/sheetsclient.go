package sheetsclient

import "google.golang.org/api/sheets/v4"

type clientStruct struct{}

type ClientInterface interface{
	Get(ssService *sheets.Service, spreadsheetID string, range_ string) (*sheets.ValueRange, error )
}

var ClientStruct ClientInterface = &clientStruct{}

//func New(service *sheets.Service) *clientStruct {
//	return &clientStruct{
//		SpreadsheetsService: service,
//	}
//}

func (c *clientStruct) Get(ssService *sheets.Service, spreadsheetId string, range_ string) (*sheets.ValueRange, error){
	//resp, err := c.SpreadsheetsService.Spreadsheets.Values.Get(spreadsheetId, range_).Do()
	resp, err := ssService.Spreadsheets.Values.Get(spreadsheetId, range_).Do()
	return resp, err
}
