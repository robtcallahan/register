package sheets_provider

import (
	"context"
	"errors"
	"fmt"
	"log"

	"register/pkg/config"
	"register/pkg/sheets_auth"

	"github.com/googleapis/gax-go/v2/apierror"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

type SheetsProviderInterface interface {
	GetValues(range_ string) ([][]any, error)
	GetFormula(range_ string) ([][]any, error)
	GetSpreadsheet(tabName string) (*Spreadsheet, error)
	BatchUpdate(updateReq *sheets.BatchUpdateSpreadsheetRequest) (*sheets.BatchUpdateSpreadsheetResponse, error)
	Update(writeRange string, vRange *sheets.ValueRange) (*sheets.UpdateValuesResponse, error)
	ClearValues(range_ string) error
}

type SheetsProvider struct {
	service       *sheets.Service
	spreadsheetID string
}

type Spreadsheet struct {
	Title       string
	SheetID     int64
	RowCount    int64
	ColumnCount int64
}

func New(spreadsheetID string, cfg *config.Config) (*SheetsProvider, error) {
	ctx := context.Background()
	httpClient, err := sheets_auth.GetClient()
	if err != nil {
		return nil, fmt.Errorf("unable to get http client: %w", err)
	}
	service, err := sheets.NewService(ctx,
		option.WithHTTPClient(httpClient),
		option.WithCredentialsFile(cfg.GoogleCredentialsFile),
	)
	if err != nil {
		if ae, ok := errors.AsType[*apierror.APIError](err); ok {
			log.Println(ae.Reason())
			log.Println(ae.Details().Help.GetLinks())
			return nil, fmt.Errorf("unable to retrieve sheets client: %s", ae.Reason())
		}
		return nil, fmt.Errorf("unable to retrieve sheets client: %w", err)
	}
	return &SheetsProvider{
		service:       service,
		spreadsheetID: spreadsheetID,
	}, nil
}

func (p *SheetsProvider) GetValues(range_ string) ([][]any, error) {
	resp, err := p.service.Spreadsheets.Values.Get(p.spreadsheetID, range_).Do()
	if err != nil {
		log.Printf("error when trying to get cell data: %s", err.Error())
		return nil, err
	}
	return resp.Values, nil
}

func (p *SheetsProvider) GetFormula(range_ string) ([][]any, error) {
	call := p.service.Spreadsheets.Values.BatchGet(p.spreadsheetID)
	call.ValueRenderOption("FORMULA")
	call.Ranges(range_)
	resp, err := call.Do()
	if err != nil {
		return nil, err
	}
	if resp == nil || len(resp.ValueRanges) == 0 {
		return nil, errors.New("empty response")
	}
	return resp.ValueRanges[0].Values, err
}

func (p *SheetsProvider) GetSpreadsheet(tabName string) (*Spreadsheet, error) {
	resp, err := p.service.Spreadsheets.Get(p.spreadsheetID).Do()
	if err != nil {
		log.Printf("error when trying to get spreadsheet: %s", err)
		return nil, err
	}

	for _, sheet := range resp.Sheets {
		if sheet.Properties.Title == tabName {
			spreadsheet := &Spreadsheet{
				Title:   sheet.Properties.Title,
				SheetID: sheet.Properties.SheetId,
			}
			if sheet.Properties.GridProperties != nil {
				spreadsheet.RowCount = sheet.Properties.GridProperties.RowCount
				spreadsheet.ColumnCount = sheet.Properties.GridProperties.ColumnCount
			}
			return spreadsheet, nil
		}
	}
	return nil, fmt.Errorf("could not find sheet: %s", tabName)
}

func (p *SheetsProvider) BatchUpdate(updateReq *sheets.BatchUpdateSpreadsheetRequest) (*sheets.BatchUpdateSpreadsheetResponse, error) {
	resp, err := p.service.Spreadsheets.BatchUpdate(p.spreadsheetID, updateReq).Do()
	if err != nil {
		log.Printf("error when try to batch update spreadsheet: %s", err.Error())
		return nil, err
	}
	return resp, nil
}

func (p *SheetsProvider) Update(writeRange string, vRange *sheets.ValueRange) (*sheets.UpdateValuesResponse, error) {
	call := p.service.Spreadsheets.Values.Update(p.spreadsheetID, writeRange, vRange)
	call.ValueInputOption("USER_ENTERED")
	resp, err := call.Do()
	if err != nil {
		return resp, fmt.Errorf("unable to write cell data: %w", err)
	}
	return resp, err
}

func (p *SheetsProvider) ClearValues(range_ string) error {
	_, err := p.service.Spreadsheets.Values.Clear(p.spreadsheetID, range_, &sheets.ClearValuesRequest{}).Do()
	return err
}
