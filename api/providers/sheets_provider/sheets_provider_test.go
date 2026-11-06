package sheets_provider

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

func newTestProvider(t *testing.T, handler http.HandlerFunc) *SheetsProvider {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	svc, err := sheets.NewService(context.Background(),
		option.WithEndpoint(server.URL),
		option.WithoutAuthentication(),
	)
	if err != nil {
		t.Fatalf("could not build test sheets service: %v", err)
	}
	return &SheetsProvider{service: svc, spreadsheetID: "ssID"}
}

func TestSheetsProvider_GetValues(t *testing.T) {
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
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.respErr != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(&sheets.ValueRange{
					Range:  tt.response.Range,
					Values: tt.response.Values,
				})
			})
			p := newTestProvider(t, handler)
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
	fakeSpreadsheet := &sheets.Spreadsheet{
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
	}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(fakeSpreadsheet) // the struct you built in code
	})
	p := newTestProvider(t, handler)

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

func TestSheetsProvider_GetFormula(t *testing.T) {
	tests := []struct {
		name     string
		response *sheets.BatchGetValuesResponse
		respErr  error
		want     [][]any
		wantErr  string
	}{
		{
			name: "returns the formulas from the first value range",
			response: &sheets.BatchGetValuesResponse{
				ValueRanges: []*sheets.ValueRange{
					{Range: "Register!A1:A2", Values: [][]any{{"=SUM(A1:A3)"}, {"=B1*2"}}},
				},
			},
			want: [][]any{{"=SUM(A1:A3)"}, {"=B1*2"}},
		},
		{
			name:    "client error propagates",
			respErr: errors.New("boom"),
			wantErr: "boom",
		},
		{
			name:     "empty response is an error",
			response: &sheets.BatchGetValuesResponse{},
			wantErr:  "empty response",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.respErr != nil {
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte(tt.respErr.Error()))
					return
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(tt.response)
			})
			p := newTestProvider(t, handler)
			got, err := p.GetFormula("Register!A1:A2")
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("GetFormula() error = nil, want error containing %q", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("GetFormula() error = %v, want error containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("GetFormula() error = %v, want nil", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetFormula() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSheetsProvider_GetSpreadsheet_ClientError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	p := newTestProvider(t, handler)
	got, err := p.GetSpreadsheet("Register")
	if err == nil {
		t.Fatalf("GetSpreadsheet() error = nil, want error")
	}
	if got != nil {
		t.Errorf("GetSpreadsheet() = %+v, want nil", got)
	}
}

func TestSheetsProvider_BatchUpdate(t *testing.T) {
	tests := []struct {
		name    string
		respErr bool
		want    *sheets.BatchUpdateSpreadsheetResponse
		wantErr bool
	}{
		{
			name: "returns the batch update response",
			want: &sheets.BatchUpdateSpreadsheetResponse{
				SpreadsheetId: "ssID",
				UpdatedSpreadsheet: &sheets.Spreadsheet{
					SpreadsheetId: "ssID",
				},
			},
		},
		{
			name:    "client error propagates",
			respErr: true,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.respErr {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(tt.want)
			})
			p := newTestProvider(t, handler)
			got, err := p.BatchUpdate(&sheets.BatchUpdateSpreadsheetRequest{})
			if (err != nil) != tt.wantErr {
				t.Fatalf("BatchUpdate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if got.SpreadsheetId != tt.want.SpreadsheetId {
				t.Errorf("BatchUpdate() SpreadsheetId = %q, want %q", got.SpreadsheetId, tt.want.SpreadsheetId)
			}
			if got.UpdatedSpreadsheet == nil || got.UpdatedSpreadsheet.SpreadsheetId != tt.want.UpdatedSpreadsheet.SpreadsheetId {
				t.Errorf("BatchUpdate() UpdatedSpreadsheet = %+v, want %+v", got.UpdatedSpreadsheet, tt.want.UpdatedSpreadsheet)
			}
		})
	}
}

func TestSheetsProvider_Update(t *testing.T) {
	tests := []struct {
		name    string
		respErr bool
		want    *sheets.UpdateValuesResponse
		wantErr string
	}{
		{
			name: "returns the update response",
			want: &sheets.UpdateValuesResponse{
				SpreadsheetId: "ssID",
				UpdatedRange:  "Register!A1",
				UpdatedCells:  2,
			},
		},
		{
			name:    "client error is wrapped",
			respErr: true,
			wantErr: "unable to write cell data",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.respErr {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(tt.want)
			})
			p := newTestProvider(t, handler)
			got, err := p.Update("Register!A1", &sheets.ValueRange{
				Range:  "Register!A1",
				Values: [][]any{{"a"}, {"b"}},
			})
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("Update() error = nil, want error containing %q", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("Update() error = %v, want error containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Update() error = %v, want nil", err)
			}
			if got.SpreadsheetId != tt.want.SpreadsheetId || got.UpdatedRange != tt.want.UpdatedRange || got.UpdatedCells != tt.want.UpdatedCells {
				t.Errorf("Update() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestSheetsProvider_ClearValues(t *testing.T) {
	tests := []struct {
		name    string
		respErr bool
		wantErr bool
	}{
		{
			name: "clears without error",
		},
		{
			name:    "client error propagates",
			respErr: true,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.respErr {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(&sheets.ClearValuesResponse{ClearedRange: "Register!A1:A2"})
			})
			p := newTestProvider(t, handler)
			err := p.ClearValues("Register!A1:A2")
			if (err != nil) != tt.wantErr {
				t.Errorf("ClearValues() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
