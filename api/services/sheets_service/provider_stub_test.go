package sheets_service

import (
	"testing"

	"register/api/providers/sheets_provider"

	"google.golang.org/api/sheets/v4"
)

// stubSheetsProvider is a configurable fake for error-path tests: each
// provider method can fail on demand, and BatchUpdate/Update/ClearValues
// calls are recorded for inspection. Pointer receivers throughout, so
// recordings survive the call.
type stubSheetsProvider struct {
	getValuesFunc      func(string) ([][]any, error)
	getValuesErr       error
	getValuesErrOnCall int
	getValuesCalls     int
	getFormula         [][]any
	getFormulaErr      error
	spreadsheet        *sheets_provider.Spreadsheet
	spreadsheetErr     error
	batchUpdateErr     error
	batchUpdates       []*sheets.BatchUpdateSpreadsheetRequest
	updateErr          error
	updateRanges       []string
	updateValues       []*sheets.ValueRange
	clearValuesErr     error
	clearValuesCalls   []string
}

func (s *stubSheetsProvider) GetValues(range_ string) ([][]any, error) {
	s.getValuesCalls++
	if s.getValuesFunc != nil {
		return s.getValuesFunc(range_)
	}
	if s.getValuesErrOnCall > 0 && s.getValuesCalls == s.getValuesErrOnCall {
		return nil, errTestProvider
	}
	if s.getValuesErr != nil {
		return nil, s.getValuesErr
	}
	return nil, nil
}

func (s *stubSheetsProvider) GetFormula(range_ string) ([][]any, error) {
	return s.getFormula, s.getFormulaErr
}

func (s *stubSheetsProvider) GetSpreadsheet(tabName string) (*sheets_provider.Spreadsheet, error) {
	if s.spreadsheetErr != nil {
		return nil, s.spreadsheetErr
	}
	return s.spreadsheet, nil
}

func (s *stubSheetsProvider) BatchUpdate(req *sheets.BatchUpdateSpreadsheetRequest) (*sheets.BatchUpdateSpreadsheetResponse, error) {
	s.batchUpdates = append(s.batchUpdates, req)
	if s.batchUpdateErr != nil {
		return nil, s.batchUpdateErr
	}
	return &sheets.BatchUpdateSpreadsheetResponse{}, nil
}

func (s *stubSheetsProvider) Update(writeRange string, vRange *sheets.ValueRange) (*sheets.UpdateValuesResponse, error) {
	s.updateRanges = append(s.updateRanges, writeRange)
	s.updateValues = append(s.updateValues, vRange)
	if s.updateErr != nil {
		return nil, s.updateErr
	}
	return &sheets.UpdateValuesResponse{}, nil
}

func (s *stubSheetsProvider) ClearValues(range_ string) error {
	s.clearValuesCalls = append(s.clearValuesCalls, range_)
	return s.clearValuesErr
}

// stashProviderFuncs restores the package-level provider seams when the
// calling test ends, so a test that stubs GetValues/GetFormula cannot leak
// its stub into the next test.
func stashProviderFuncs(t *testing.T) {
	t.Helper()
	oldValues, oldFormula := getValuesProviderFunc, getFormulaProviderFunc
	t.Cleanup(func() {
		getValuesProviderFunc, getFormulaProviderFunc = oldValues, oldFormula
	})
}
