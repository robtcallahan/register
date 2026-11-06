package sheets_service

import (
	"errors"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"register/pkg/config"
)

func registerBulkSheet() RegisterSheet {
	return RegisterSheet{
		TabName: RegisterTabName,
		SheetCoords: SheetCoords{
			StartRow:       5,
			EndRow:         100,
			EndColumnName:  "AU",
			EndColumnIndex: 46,
		},
	}
}

func registerBulkHeader() []any {
	return []any{"Transaction ID", "Source", "Date", "Description", " Withdrawals ", " Deposits ", " Credit Purchases ", " Register ", " Cleared ", " Delta "}
}

func registerBulkDataRows() [][]any {
	// Two rows per entry (data row + note row), as the sheet lays them
	// out: TARGET at values[0], AMAZON at values[2].
	return [][]any{
		{"", "WellsFargo", "12/07/20", "TARGET", "46.80", "", "", "8761.12", "", ""},
		{"", "", "", "", "", "", "", "", "", ""},
		{"", "WellsFargo", "12/08/20", "AMAZON", "10.00", "", "", "", "", ""},
		{"", "", "", "", "", "", "", "", "", ""},
	}
}

func Test_registerService_NewRegisterSheet_SpreadsheetError(t *testing.T) {
	cfg := &config.Config{RegisterStartRow: 5, RegisterEndRow: 100}
	svc := NewRegisterService(&stubSheetsProvider{spreadsheetErr: errTestProvider})
	if _, err := svc.NewRegisterSheet(cfg, nil); !errors.Is(err, errTestProvider) {
		t.Errorf("NewRegisterSheet() error = %v, wantErr %v", err, errTestProvider)
	}
}

func Test_registerService_ReadRegisterSheet_Errors(t *testing.T) {
	tests := []struct {
		name    string
		stub    func(string) ([][]any, error)
		wantSub string
		wantErr error
	}{
		{
			name:    "values error",
			stub:    func(string) ([][]any, error) { return nil, errTestProvider },
			wantSub: "could not get sheet values",
		},
		{
			name:    "no values",
			stub:    func(string) ([][]any, error) { return [][]any{}, nil },
			wantSub: "no data found for read range",
		},
		{
			name: "header error",
			stub: func(range_ string) ([][]any, error) {
				if range_ == "Register!A4:J4" {
					header := registerBulkHeader()
					header[9] = "Total" // Delta renamed: the require fails
					return [][]any{header}, nil
				}
				return registerBulkDataRows(), nil
			},
			wantSub: `register header "Delta" not found`,
		},
		{
			name: "bad row amount",
			stub: func(range_ string) ([][]any, error) {
				if range_ == "Register!A4:J4" {
					return [][]any{registerBulkHeader()}, nil
				}
				rows := registerBulkDataRows()
				rows[0][Withdrawals] = "xx"
				return rows, nil
			},
			wantErr: strconv.ErrSyntax,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stashProviderFuncs(t)
			getValuesProviderFunc = tt.stub
			s := registerBulkSheet()
			_, err := ss.ReadRegisterSheet(&s)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("ReadRegisterSheet() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantSub) {
				t.Errorf("ReadRegisterSheet() error = %v, want %q", err, tt.wantSub)
			}
		})
	}
}

func Test_registerService_ReadRegisterSheet_Loop(t *testing.T) {
	stashProviderFuncs(t)
	getValuesProviderFunc = func(range_ string) ([][]any, error) {
		if range_ == "Register!A4:J4" {
			return [][]any{registerBulkHeader()}, nil
		}
		return registerBulkDataRows(), nil
	}
	sheet := registerBulkSheet()
	got, err := ss.ReadRegisterSheet(&sheet)
	if err != nil {
		t.Fatalf("ReadRegisterSheet() error = %v", err)
	}
	// 4 value rows: LastRow = 4 + 5 - 2 = 7; the loop reads i = 0, 2, 4
	// and breaks on the out-of-range row at i = 4.
	if got.SheetCoords.LastRow != 7 {
		t.Errorf("LastRow = %d, want 7", got.SheetCoords.LastRow)
	}
	if got.SheetCoords.FirstRowToUpdate != 8 {
		t.Errorf("FirstRowToUpdate = %d, want 8", got.SheetCoords.FirstRowToUpdate)
	}
	if len(got.Entries) != 2 {
		t.Fatalf("len(Entries) = %d, want 2", len(got.Entries))
	}
	first := got.Entries[0]
	if first.RowID != 5 || first.Name != "TARGET" || first.Withdrawal != 46.80 || first.BankRegister != 8761.12 {
		t.Errorf("Entries[0] = %+v, want RowID 5 TARGET 46.80/8761.12", first)
	}
	if first.Key != "wellsfargo:12/07/20:46.80" {
		t.Errorf("Entries[0].Key = %q, want wellsfargo:12/07/20:46.80", first.Key)
	}
	second := got.Entries[1]
	if second.RowID != 7 || second.Name != "AMAZON" || second.Withdrawal != 10 {
		t.Errorf("Entries[1] = %+v, want RowID 7 AMAZON 10", second)
	}
	if !reflect.DeepEqual(got.RangeValues, registerBulkDataRows()) {
		t.Errorf("RangeValues = %v, want the data rows", got.RangeValues)
	}
}

func Test_registerService_ReadCell_Errors(t *testing.T) {
	sheet := registerBulkSheet()
	t.Run("values error", func(t *testing.T) {
		stashProviderFuncs(t)
		getValuesProviderFunc = func(string) ([][]any, error) { return nil, errTestProvider }
		if _, err := ss.ReadCell(sheet, "A1", CellDataString); !errors.Is(err, errTestProvider) {
			t.Errorf("ReadCell() error = %v, wantErr %v", err, errTestProvider)
		}
	})
	t.Run("formula error", func(t *testing.T) {
		stashProviderFuncs(t)
		getFormulaProviderFunc = func(string) ([][]any, error) { return nil, errTestProvider }
		if _, err := ss.ReadCell(sheet, "A1", CellDataFormula); !errors.Is(err, errTestProvider) {
			t.Errorf("ReadCell() error = %v, wantErr %v", err, errTestProvider)
		}
	})
	t.Run("no data", func(t *testing.T) {
		stashProviderFuncs(t)
		getValuesProviderFunc = func(string) ([][]any, error) { return [][]any{}, nil }
		if _, err := ss.ReadCell(sheet, "A1", CellDataString); err == nil || !strings.Contains(err.Error(), "no data found for cell: A1") {
			t.Errorf("ReadCell() error = %v, want no-data error", err)
		}
	})
}

func Test_registerService_ReadCells_Errors(t *testing.T) {
	sheet := registerBulkSheet()
	t.Run("string cell error", func(t *testing.T) {
		stashProviderFuncs(t)
		getValuesProviderFunc = func(string) ([][]any, error) { return nil, errTestProvider }
		if _, err := ss.ReadStringCell(sheet, "A1"); err == nil || !strings.Contains(err.Error(), "could not read string cell A1") {
			t.Errorf("ReadStringCell() error = %v, want string-cell error", err)
		}
	})
	t.Run("dollars cell error", func(t *testing.T) {
		stashProviderFuncs(t)
		getValuesProviderFunc = func(string) ([][]any, error) { return nil, errTestProvider }
		if _, err := ss.ReadDollarsCell(sheet, "A1"); !errors.Is(err, errTestProvider) {
			t.Errorf("ReadDollarsCell() error = %v, wantErr %v", err, errTestProvider)
		}
	})
	t.Run("formula cell error", func(t *testing.T) {
		stashProviderFuncs(t)
		getFormulaProviderFunc = func(string) ([][]any, error) { return nil, errTestProvider }
		if _, err := ss.ReadFormulaCell(sheet, "A1"); !errors.Is(err, errTestProvider) {
			t.Errorf("ReadFormulaCell() error = %v, wantErr %v", err, errTestProvider)
		}
	})
	t.Run("date cell error", func(t *testing.T) {
		stashProviderFuncs(t)
		getValuesProviderFunc = func(string) ([][]any, error) { return nil, errTestProvider }
		if _, err := ss.ReadDateCell(sheet, "A1"); !errors.Is(err, errTestProvider) {
			t.Errorf("ReadDateCell() error = %v, wantErr %v", err, errTestProvider)
		}
	})
}

func Test_registerService_WriteCell_UpdateError(t *testing.T) {
	svc := NewRegisterService(&stubSheetsProvider{updateErr: errTestProvider})
	if _, err := svc.WriteCell(RegisterTabName, "G1", 10.00); err == nil || !strings.Contains(err.Error(), "unable to write cell data") {
		t.Errorf("WriteCell() error = %v, want write error", err)
	}
}

func Test_registerService_CopyRows_BatchUpdateErrors(t *testing.T) {
	sheet := RegisterSheet{
		TabName: RegisterTabName,
		SheetCoords: SheetCoords{
			LastRow:        100,
			RowCount:       1000,
			EndColumnIndex: 45,
		},
	}
	t.Run("grid limits error maps to errSheetFull", func(t *testing.T) {
		svc := NewRegisterService(&stubSheetsProvider{batchUpdateErr: errors.New("request exceeds grid limits")})
		if err := svc.CopyRows(sheet, 1); !errors.Is(err, errSheetFull) {
			t.Errorf("CopyRows() error = %v, want errSheetFull", err)
		}
	})
	t.Run("other batch error", func(t *testing.T) {
		svc := NewRegisterService(&stubSheetsProvider{batchUpdateErr: errTestProvider})
		if err := svc.CopyRows(sheet, 1); err == nil || !strings.Contains(err.Error(), "could not perform copy") {
			t.Errorf("CopyRows() error = %v, want copy error", err)
		}
	})
	t.Run("unknown grid size lets the API decide", func(t *testing.T) {
		if err := ss.checkRowsFit(RegisterSheet{SheetCoords: SheetCoords{RowCount: 0, LastRow: 100}}, 1); err != nil {
			t.Errorf("checkRowsFit(RowCount 0) error = %v, want nil", err)
		}
	})
}

func Test_registerService_PopulateRegisterEntry_BadAmounts(t *testing.T) {
	base := []any{"", "WellsFargo", "12/07/20", "TARGET", "46.80", "", "", "8761.12", "", ""}
	fields := []struct {
		name string
		idx  int
	}{
		{"deposits", Deposits},
		{"credit purchases", CreditPurchase},
		{"bank register", BankRegister},
		{"cleared", Cleared},
		{"delta", Delta},
	}
	for _, f := range fields {
		t.Run(f.name, func(t *testing.T) {
			row := make([]any, len(base))
			copy(row, base)
			row[f.idx] = "xx"
			if _, err := ss.populateRegisterEntry(row); !errors.Is(err, strconv.ErrSyntax) {
				t.Errorf("populateRegisterEntry() error = %v, wantErr %v", err, strconv.ErrSyntax)
			}
		})
	}
}

func Test_registerService_ReadRangeFormulas_Errors(t *testing.T) {
	t.Run("provider error", func(t *testing.T) {
		stashProviderFuncs(t)
		getFormulaProviderFunc = func(string) ([][]any, error) { return nil, errTestProvider }
		if _, err := ss.readRangeFormulas("A1:A2"); err == nil || !strings.Contains(err.Error(), "unable to retrieve data from sheet") {
			t.Errorf("readRangeFormulas() error = %v, want retrieve error", err)
		}
	})
	t.Run("no data", func(t *testing.T) {
		stashProviderFuncs(t)
		getFormulaProviderFunc = func(string) ([][]any, error) { return [][]any{}, nil }
		if _, err := ss.readRangeFormulas("A1:A2"); err == nil || !strings.Contains(err.Error(), "no data found for read range: A1:A2") {
			t.Errorf("readRangeFormulas() error = %v, want no-data error", err)
		}
	})
}

func Test_registerService_CheckRegisterHeaders_Errors(t *testing.T) {
	sheet := RegisterSheet{TabName: RegisterTabName}
	tests := []struct {
		name    string
		stub    func(string) ([][]any, error)
		wantSub string
	}{
		{
			name:    "values error",
			stub:    func(string) ([][]any, error) { return nil, errTestProvider },
			wantSub: "could not read register header row",
		},
		{
			name:    "no header row",
			stub:    func(string) ([][]any, error) { return [][]any{}, nil },
			wantSub: "no header row found on Register tab",
		},
		{
			name: "duplicate header",
			stub: func(string) ([][]any, error) {
				header := registerBulkHeader()
				header[3] = "Date" // Description renamed to a duplicate Date
				return [][]any{header}, nil
			},
			wantSub: "duplicate header",
		},
		{
			name: "missing header",
			stub: func(string) ([][]any, error) {
				return [][]any{registerBulkHeader()[:9]}, nil
			},
			wantSub: `register header "Delta" not found`,
		},
		{
			name: "moved header",
			stub: func(string) ([][]any, error) {
				header := registerBulkHeader()
				header[8], header[9] = header[9], header[8] // Cleared and Delta swapped
				return [][]any{header}, nil
			},
			wantSub: `register header "Cleared" moved`,
		},
		{
			name:    "headers ok",
			stub:    func(string) ([][]any, error) { return [][]any{registerBulkHeader()}, nil },
			wantSub: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stashProviderFuncs(t)
			getValuesProviderFunc = tt.stub
			err := ss.checkRegisterHeaders(sheet)
			if tt.wantSub == "" {
				if err != nil {
					t.Errorf("checkRegisterHeaders() error = %v, want nil", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantSub) {
				t.Errorf("checkRegisterHeaders() error = %v, want %q", err, tt.wantSub)
			}
		})
	}
}
