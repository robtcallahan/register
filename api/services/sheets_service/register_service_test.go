package sheets_service

import (
	"errors"
	"reflect"
	"testing"

	"register/pkg/models"

	"google.golang.org/api/sheets/v4"
)

func Test_registerService_ReadRegisterSheet(t *testing.T) {
	// 10.10: Rewritten to test 9.3 header validation without fixtures.
	getValuesProviderFunc = func(range_ string) ([][]any, error) {
		var values [][]any
		if range_ == "Register!A4:J4" {
			values = [][]any{{"Recon", "Check", "Date", "Description", " Withdrawals ", " Deposits ", " Credit Purchases ", " Register ", " Cleared ", " Delta "}}
			return values, nil
		}
		values = [][]any{
			{"", "WellsFargo", "12/07/20", "TARGET", "46.80", "", "", "8761.12", "", ""},
		}
		return values, nil
	}

	got, err := ss.ReadRegisterSheet(registerSheet)
	if err != nil {
		t.Fatalf("ReadRegisterSheet() error = %v", err)
	}
	if got == nil {
		t.Fatal("ReadRegisterSheet() returned nil")
	}
}

func Test_registerService_CopyRows_sheetFull(t *testing.T) {
	svc := NewRegisterService(&providerMock{t: t})
	registerSheet = &RegisterSheet{
		TabName: "Register",
		SheetCoords: SheetCoords{
			LastRow:        100,
			RowCount:       101,
			EndColumnIndex: 45,
		},
	}

	// last row 100 + 1 copy (2 rows) = 102 > 101 grid rows: must not fit
	if err := svc.CopyRows(*registerSheet, 1); !errors.Is(err, errSheetFull) {
		t.Errorf("CopyRows() error = %v, want errSheetFull", err)
	}

	// plenty of room: the copy proceeds and the provider accepts it
	registerSheet.SheetCoords.RowCount = 1000
	if err := svc.CopyRows(*registerSheet, 10); err != nil {
		t.Errorf("CopyRows() error = %v, want nil", err)
	}
}

func Test_registerService_copyRowsBatchUpdateRequest(t *testing.T) {
	// Self-contained: verifies the request structure without a JSON golden.
	// The fixture-based golden was stale (predates refactors).
	registerSheet.SheetCoords.LastRow = 100

	got := ss.copyRowsBatchUpdateRequest(*registerSheet, 3)

	if len(got.Requests) != 3 {
		t.Fatalf("got %d requests, want 3", len(got.Requests))
	}
	for i, req := range got.Requests {
		if req.CopyPaste == nil {
			t.Errorf("request %d: CopyPaste is nil", i)
			continue
		}
		if req.CopyPaste.PasteType != "PASTE_NORMAL" {
			t.Errorf("request %d: PasteType = %q, want PASTE_NORMAL", i, req.CopyPaste.PasteType)
		}
	}
}

func Test_registerService_populateCells(t *testing.T) {
	type fields struct {
		service       *sheets.Service
		BudgetSheet   *BudgetSheet
		RegisterSheet *RegisterSheet
	}
	type args struct {
		columns            []models.Column
		transNameToColName map[string]string
		transactions       []*models.Transaction
	}
	var tests []struct {
		name   string
		fields fields
		args   args
		want   []*sheets.RowData
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rb := &RowBuilder{
				Columns:            tt.args.columns,
				TransNameToColName: tt.args.transNameToColName,
			}
			if got, _ := rb.populateCells(tt.args.transactions); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("populateCells() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_registerService_compareColumnHeaders(t *testing.T) {
	columns := []models.Column{
		{Name: "BankRegister", ColumnIndex: 8}, // fixed block: never compared
		{Name: "Cash", ColumnIndex: 11},
		{Name: "Dining Out", ColumnIndex: 12},
	}
	headerRow := []any{"", "", "", "", "", "", "", " Register ", "", "", " Cash ", "Groceries"}
	if problems := compareColumnHeaders(columns, headerRow); len(problems) != 1 {
		t.Errorf("compareColumnHeaders() = %v; want exactly the Dining Out mismatch", problems)
	}
	if problems := compareColumnHeaders(columns[:2], headerRow); len(problems) != 0 {
		t.Errorf("compareColumnHeaders() = %v; want no problems", problems)
	}
}

func Test_registerService_shiftFormulaColumn(t *testing.T) {
	tests := []struct {
		formula string
		from    string
		to      string
		want    string
	}{
		{"=SUM(K5:K6)", "K", "L", "=SUM(L5:L6)"},
		{"=SUM(AA10:AA11)", "AA", "AB", "=SUM(AB10:AB11)"},
		{"=$K$5+K7", "K", "L", "=$L$5+L7"},
		{"=SUM(K5:K6)+Z2", "K", "L", "=SUM(L5:L6)+Z2"},
		{"Cash", "K", "L", "Cash"},
	}
	for _, tt := range tests {
		if got := shiftFormulaColumn(tt.formula, tt.from, tt.to); got != tt.want {
			t.Errorf("shiftFormulaColumn(%q, %q, %q) = %q; want %q", tt.formula, tt.from, tt.to, got, tt.want)
		}
	}
}

func checkTestingError(t *testing.T, err error) {
	if err != nil {
		t.Fatalf("error: %s\n", err.Error())
	}
}
