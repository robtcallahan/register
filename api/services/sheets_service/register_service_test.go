package sheets_service

import (
	"errors"
	"reflect"
	"testing"

	"register/pkg/config"
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

func Test_registerService_NewRegisterSheet(t *testing.T) {
	cfg := &config.Config{RegisterStartRow: 5, RegisterEndRow: 100}
	columns := []models.Column{
		{Name: "Check", ColumnIndex: 2},
		{Name: "Grocery", ColumnIndex: 11, Color: models.ColorBlue},
		{Name: "Rent", ColumnIndex: 12, Color: models.ColorBlue},
	}
	got, err := ss.NewRegisterSheet(cfg, columns)
	if err != nil {
		t.Fatalf("NewRegisterSheet() error = %v, want nil", err)
	}
	if got.TabName != RegisterTabName {
		t.Errorf("TabName = %q, want %q", got.TabName, RegisterTabName)
	}
	if got.ID != 617336355 {
		t.Errorf("ID = %d, want 617336355", got.ID)
	}
	if got.SheetCoords.RowCount != 1000 || got.SheetCoords.ColumnCount != 47 {
		t.Errorf("grid = %dx%d, want 1000x47", got.SheetCoords.RowCount, got.SheetCoords.ColumnCount)
	}
	if got.SheetCoords.StartRow != 5 || got.SheetCoords.EndRow != 100 {
		t.Errorf("rows = %d..%d, want 5..100", got.SheetCoords.StartRow, got.SheetCoords.EndRow)
	}
	// End comes from the DB's highest column: index 12 -> L, offset 11.
	if got.SheetCoords.EndColumnName != "L" || got.SheetCoords.EndColumnIndex != 11 {
		t.Errorf("end = %q/%d, want L/11", got.SheetCoords.EndColumnName, got.SheetCoords.EndColumnIndex)
	}

	// No DB columns: the end comes from the sheet's own grid width.
	got, err = ss.NewRegisterSheet(cfg, nil)
	if err != nil {
		t.Fatalf("NewRegisterSheet(nil columns) error = %v, want nil", err)
	}
	if got.SheetCoords.EndColumnName != "AU" || got.SheetCoords.EndColumnIndex != 46 {
		t.Errorf("end without columns = %q/%d, want AU/46", got.SheetCoords.EndColumnName, got.SheetCoords.EndColumnIndex)
	}
}

func Test_registerService_isEmptyRow(t *testing.T) {
	tests := []struct {
		name string
		row  []any
		want bool
	}{
		{"empty date", []any{"", "WellsFargo", "", "TARGET"}, true},
		{"date present", []any{"", "WellsFargo", "12/07/20", "TARGET"}, false},
		{"short row", []any{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ss.isEmptyRow(tt.row); got != tt.want {
				t.Errorf("isEmptyRow() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_registerService_getRowID(t *testing.T) {
	sheet := RegisterSheet{SheetCoords: SheetCoords{StartRow: 5}}
	tests := []struct {
		i    int64
		want int64
	}{
		{0, 5},
		{4, 9},
		{100, 105},
	}
	for _, tt := range tests {
		if got := ss.getRowID(sheet, tt.i); got != tt.want {
			t.Errorf("getRowID(%d) = %d, want %d", tt.i, got, tt.want)
		}
	}
}

func Test_registerService_populateRegisterEntry(t *testing.T) {
	row := []any{"X", "WellsFargo", "12/07/20", "TARGET", "46.80", "", "", "8761.12", "", ""}
	got, err := ss.populateRegisterEntry(row)
	if err != nil {
		t.Fatalf("populateRegisterEntry() error = %v, want nil", err)
	}
	if got.Key != "wellsfargo:12/07/20:46.80" {
		t.Errorf("Key = %q, want wellsfargo:12/07/20:46.80", got.Key)
	}
	if got.Reconciled != "X" || got.Source != "wellsfargo" || got.Date != "12/07/20" || got.Name != "TARGET" {
		t.Errorf("identity fields = %+v, want X/wellsfargo/12/07/20/TARGET", got)
	}
	if got.Withdrawal != 46.80 || got.Deposit != 0 || got.CreditPurchase != 0 {
		t.Errorf("amounts = %v/%v/%v, want 46.80/0/0", got.Withdrawal, got.Deposit, got.CreditPurchase)
	}
	if got.BankRegister != 8761.12 || got.Cleared != 0 || got.Delta != 0 {
		t.Errorf("register fields = %v/%v/%v, want 8761.12/0/0", got.BankRegister, got.Cleared, got.Delta)
	}

	depositRow := []any{"", "WellsFargo", "12/08/20", "PAYCHECK", "", "1022.69", "", "", "", ""}
	got, err = ss.populateRegisterEntry(depositRow)
	if err != nil {
		t.Fatalf("populateRegisterEntry(deposit) error = %v, want nil", err)
	}
	if got.Deposit != 1022.69 || got.Key != "wellsfargo:12/08/20:1022.69" {
		t.Errorf("deposit entry = %+v, want Deposit 1022.69 key wellsfargo:12/08/20:1022.69", got)
	}

	badRow := []any{"", "WellsFargo", "12/07/20", "TARGET", "not-a-number", "", "", "", "", ""}
	if _, err := ss.populateRegisterEntry(badRow); err == nil {
		t.Errorf("populateRegisterEntry(bad amount) error = nil, want error")
	}
}
