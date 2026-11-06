package sheets_service

import (
	"strings"
	"testing"

	"register/api/providers/sheets_provider"
	"register/pkg/config"
	"register/pkg/models"

	"google.golang.org/api/sheets/v4"
)

func Test_monthlyService_UpdateMonthly_Errors(t *testing.T) {
	rows := []*sheets.RowData{
		{Values: []*sheets.CellData{mkCellDataString("x", "left", "", false)}},
	}
	t.Run("clear values error", func(t *testing.T) {
		ms := NewMonthlyService(&stubSheetsProvider{clearValuesErr: errTestProvider}, nil)
		if err := ms.updateMonthly("Budget", 987654321, rows); err == nil || !strings.Contains(err.Error(), "error:") {
			t.Errorf("updateMonthly() error = %v, want clear error", err)
		}
	})
	t.Run("batch update error", func(t *testing.T) {
		ms := NewMonthlyService(&stubSheetsProvider{batchUpdateErr: errTestProvider}, nil)
		if err := ms.updateMonthly("Budget", 987654321, rows); err == nil || !strings.Contains(err.Error(), "error:") {
			t.Errorf("updateMonthly() error = %v, want batch error", err)
		}
	})
}

func Test_monthlyService_UpdateYearlySummary_Errors(t *testing.T) {
	columns := []models.Column{
		{Name: "Grocery", ColumnIndex: 11, Color: models.ColorBlue},
		{Name: "Rent", ColumnIndex: 12, Color: models.ColorBlue},
	}
	catAgg := map[string]map[string]float64{
		"01/26": {"Grocery": 10},
	}

	t.Run("income names are listed", func(t *testing.T) {
		ms := NewMonthlyService(&providerMock{}, []config.IncomeSource{
			{Name: "CrowdStrike Salary", Frequency: "monthly"},
		})
		if err := ms.UpdateYearlySummary("Budget", catAgg, columns); err != nil {
			t.Errorf("UpdateYearlySummary() error = %v, want nil", err)
		}
	})
	t.Run("ensure column count error", func(t *testing.T) {
		// The yearly table is 4 columns wide; a 2-column tab must widen,
		// and the widening batch update fails here.
		stub := &stubSheetsProvider{
			spreadsheet:    &sheets_provider.Spreadsheet{Title: "Budget", SheetID: 1, ColumnCount: 2},
			batchUpdateErr: errTestProvider,
		}
		ms := NewMonthlyService(stub, nil)
		if err := ms.UpdateYearlySummary("Budget", catAgg, columns); err == nil || !strings.Contains(err.Error(), "error:") {
			t.Errorf("UpdateYearlySummary() error = %v, want widen error", err)
		}
	})
}
