package sheets_provider

import (
	"errors"
	"reflect"
	"testing"

	"register/api/clients/sheets_client"

	"google.golang.org/api/sheets/v4"
)

var getRequestFunc func(spreadsheetID string, range_ string) (*sheets.ValueRange, error)
var getSpreadsheetFunc func(spreadsheetID string) (*sheets.Spreadsheet, error)

type getClientMock struct{}

// We are mocking the client method "Get"
func (c *getClientMock) Get(ssService *sheets.Service, spreadsheetID string, range_ string) (*sheets.ValueRange, error) {
	return getRequestFunc(spreadsheetID, range_)
}

func (c *getClientMock) GetFormula(ssService *sheets.Service, spreadsheetID string, range_ string) (*sheets.ValueRange, error) {
	return getRequestFunc(spreadsheetID, range_)
}

func (c *getClientMock) GetSpreadsheet(ssService *sheets.Service, spreadsheetID string) (*sheets.Spreadsheet, error) {
	return getSpreadsheetFunc(spreadsheetID)
}

func (c *getClientMock) BatchUpdate(ssService *sheets.Service, spreadsheetID string, updateReq *sheets.BatchUpdateSpreadsheetRequest) (*sheets.BatchUpdateSpreadsheetResponse, error) {
	return nil, errors.New("getClientMock: BatchUpdate not implemented")
}

func (c *getClientMock) Update(ssService *sheets.Service, spreadsheetID string, writeRange string, vRange *sheets.ValueRange) (*sheets.UpdateValuesResponse, error) {
	return nil, errors.New("getClientMock: Update not implemented")
}

// useMockClient points the package-global client at the mock for the
// duration of a test and restores the real client afterwards.
func useMockClient(t *testing.T) {
	t.Helper()
	original := sheets_client.ClientStruct
	sheets_client.ClientStruct = &getClientMock{}
	t.Cleanup(func() { sheets_client.ClientStruct = original })
}

func TestSheetsProvider_GetValues(t *testing.T) {
	useMockClient(t)

	tests := []struct {
		name     string
		response *sheets.ValueRange
		respErr  error
		want     [][]any
		wantErr  bool
	}{
		{
			name: "returns only the values from the response",
			response: &sheets.ValueRange{
				Range:  "Register!A1:B2",
				Values: [][]any{{"a", "b"}, {"c", "d"}},
			},
			want: [][]any{{"a", "b"}, {"c", "d"}},
		},
		{
			name:    "client error propagates",
			respErr: errors.New("boom"),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			getRequestFunc = func(spreadsheetID string, range_ string) (*sheets.ValueRange, error) {
				return tt.response, tt.respErr
			}
			p := &SheetsProvider{service: &sheets.Service{}, spreadsheetID: "ssID"}
			got, err := p.GetValues("Register!A1:B2")
			if (err != nil) != tt.wantErr {
				t.Fatalf("GetValues() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetValues() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSheetsProvider_GetSpreadsheet(t *testing.T) {
	useMockClient(t)

	// The raw spreadsheet data is built in code: no JSON fixture files.
	getSpreadsheetFunc = func(spreadsheetID string) (*sheets.Spreadsheet, error) {
		return &sheets.Spreadsheet{
			Sheets: []*sheets.Sheet{
				{
					Properties: &sheets.SheetProperties{
						Title:          "Budget",
						SheetId:        987654321,
						GridProperties: &sheets.GridProperties{RowCount: 100, ColumnCount: 10},
					},
				},
				{
					Properties: &sheets.SheetProperties{
						Title:          "Register",
						SheetId:        617336355,
						GridProperties: &sheets.GridProperties{RowCount: 1000, ColumnCount: 47},
					},
				},
				{
					// No GridProperties: the counts must stay at zero.
					Properties: &sheets.SheetProperties{
						Title:   "NoGrid",
						SheetId: 555,
					},
				},
			},
		}, nil
	}

	p := &SheetsProvider{service: &sheets.Service{}, spreadsheetID: "ssID"}
	tests := []struct {
		name    string
		tabName string
		want    *Spreadsheet
		wantErr bool
	}{
		{
			name:    "finds the Register tab",
			tabName: "Register",
			want:    &Spreadsheet{Title: "Register", SheetID: 617336355, RowCount: 1000, ColumnCount: 47},
		},
		{
			name:    "finds the Budget tab",
			tabName: "Budget",
			want:    &Spreadsheet{Title: "Budget", SheetID: 987654321, RowCount: 100, ColumnCount: 10},
		},
		{
			name:    "missing GridProperties leaves zero counts",
			tabName: "NoGrid",
			want:    &Spreadsheet{Title: "NoGrid", SheetID: 555, RowCount: 0, ColumnCount: 0},
		},
		{
			name:    "unknown tab is an error",
			tabName: "Missing",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := p.GetSpreadsheet(tt.tabName)
			if (err != nil) != tt.wantErr {
				t.Fatalf("GetSpreadsheet() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetSpreadsheet() = %+v, want %+v", got, tt.want)
			}
		})
	}
}
