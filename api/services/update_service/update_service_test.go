package update_service

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"register/api/providers/sheets_provider"
	"register/api/services/sheets_service"
	"register/pkg/banking"
	"register/pkg/config"
	"register/pkg/models"
	"register/pkg/repository"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/plaid/plaid-go/v15/plaid"
	"google.golang.org/api/sheets/v4"
)

type RepoMock struct {
	repository.QueryRepo
	count int
	trans []*models.Transaction
}

func (db *RepoMock) UpdateTransactionTables(trans []*models.Transaction) error {
	db.count = db.count + 1
	db.trans = trans
	return nil
}

func Test_updateService_PersistTransactions(t *testing.T) {
	trans := &models.Transaction{
		Source:         "FooBank",
		Name:           "Amazon",
		CreditPurchase: 299.99,
	}
	type Want struct {
		count int
		trans []*models.Transaction
	}
	tests := []struct {
		dryRun bool
		trans  []*models.Transaction
		want   *Want
	}{
		{
			dryRun: true,
			trans:  []*models.Transaction{trans},
			want: &Want{
				count: 0,
				trans: nil,
			},
		},
		{
			dryRun: false,
			trans:  []*models.Transaction{trans},
			want: &Want{
				count: 1,
				trans: []*models.Transaction{trans},
			},
		},
	}

	for _, tt := range tests {
		mock := &RepoMock{}
		s := NewUpdateService(nil, nil, nil, mock, nil, &config.Options{DryRun: tt.dryRun})
		_, err := s.PersistTransactions(tt.trans)
		if err != nil {
			t.Fatal(err)
		}
		if mock.count != tt.want.count {
			t.Errorf("count = %d, want %d", mock.count, tt.want.count)
		}
		if !reflect.DeepEqual(mock.trans, tt.want.trans) {
			t.Errorf("trans = %v want %v", mock.trans, tt.want.trans)
		}
	}

}

func Test_updateService_DedupeAgainstSheet(t *testing.T) {
	entries := []*sheets_service.RegisterEntry{
		{
			TransactionID: "24011346003100057563677; 05812; ; ; ; ;",
			Key:           "fidelity:01/01/26:10.00",
			Date:          "01/01/26",
			Amount:        10.00,
			Name:          "Martins",
		},
	}
	trans := []*models.Transaction{
		{ // match on TransID: duplicate
			TransactionID: "24011346003100057563677; 05812; ; ; ; ;",
			Key:           "fidelity:01/01/26:10.00",
			Date:          "01/01/26",
			Amount:        10.00,
			Name:          "Martins",
		},
		{ // match on Key: duplicate
			TransactionID: "",
			Key:           "fidelity:01/01/26:10.00",
			Date:          "01/01/26",
			Amount:        10.00,
			Name:          "Martins",
		},
		{ // no TransID, no Key: new
			TransactionID: "",
			Key:           "",
			Date:          "02/01/26",
			Amount:        10.00,
			Name:          "Weis",
		},
		{ // TransID, different date: duplicate but keep and update date in Register
			TransactionID: "24011346003100057563677; 05812; ; ; ; ;",
			Key:           "fidelity:01/01/26:10.00",
			Date:          "01/01/26",
			Amount:        10.00,
			Name:          "Martins",
		},
	}

	want := []*models.Transaction{
		{
			TransactionID: "",
			Key:           "",
			Date:          "02/01/26",
			Amount:        10.00,
			Name:          "Weis",
		},
	}

	sheetIDs := make(map[string]bool)
	sheetKeys := make(map[string]bool)
	for _, e := range entries {
		if e.TransactionID != "" {
			sheetIDs[e.TransactionID] = true
		}
		sheetKeys[e.Key] = true
	}

	bankClient := &banking.Client{}
	updateService := NewUpdateService(nil, nil, bankClient, nil, nil, &config.Options{DryRun: false})
	got := updateService.DedupeAgainstSheet(trans, sheetIDs, sheetKeys)

	if len(got) != 1 {
		t.Fatalf("expected 1 new transaction, got %d", len(got))
	}
	if got[0].Date != "02/01/26" || got[0].Amount != 10.00 || got[0].Name != "Weis" {
		t.Errorf("got: Date %s, Amount %0.2f Name %s, want: Date %s, Amount %0.2f Name %s",
			got[0].Date, got[0].Amount, got[0].Name, want[0].Date, want[0].Amount, want[0].Name)
	}
}

type fakeSheetsProvider struct {
	getValuesErr error
	getValues    [][]any
}

func (p *fakeSheetsProvider) GetValues(range_ string) ([][]any, error) {
	if strings.Contains(range_, "A4:J4") {
		// wants the header values, so return them with nil error
		return [][]any{{"Transaction ID", "Source", "Date", "Description", "Withdrawals", "Deposits", "Credit Purchases", "Register", "Cleared", "Delta"}}, nil
	}
	return p.getValues, p.getValuesErr
}
func (p *fakeSheetsProvider) GetFormula(range_ string) ([][]any, error) {
	return nil, fmt.Errorf("could not get formula")
}
func (p *fakeSheetsProvider) GetSpreadsheet(tabName string) (*sheets_provider.Spreadsheet, error) {
	return nil, fmt.Errorf("could not get spreadsheet")
}
func (p *fakeSheetsProvider) BatchUpdate(updateReq *sheets.BatchUpdateSpreadsheetRequest) (*sheets.BatchUpdateSpreadsheetResponse, error) {
	return nil, fmt.Errorf("could not batch update")
}
func (p *fakeSheetsProvider) Update(writeRange string, vRange *sheets.ValueRange) (*sheets.UpdateValuesResponse, error) {
	return nil, fmt.Errorf("could not update value")
}
func (p *fakeSheetsProvider) ClearValues(range_ string) error {
	return fmt.Errorf("could not clear values")
}

type fakeMysqlQueryRepo struct {
	lookupData   []*models.DataRow
	lookupErr    error
	columns      []models.Column
	columnsErr   error
	updateErr    error
	updateCalls  int
	updatedTrans []*models.Transaction
	nameMap      map[string]string
	nameMapErr   error
}

func (q *fakeMysqlQueryRepo) GetLookupData() ([]*models.DataRow, error) {
	return q.lookupData, q.lookupErr
}
func (q *fakeMysqlQueryRepo) GetColumns() ([]models.Column, error) {
	return q.columns, q.columnsErr
}

func (q *fakeMysqlQueryRepo) AddColumn(col *models.Column) error     { return nil }
func (q *fakeMysqlQueryRepo) RenameColumn(id int, name string) error { return nil }
func (q *fakeMysqlQueryRepo) DeleteColumn(id int) error              { return nil }
func (q *fakeMysqlQueryRepo) MoveColumn(id int, newIndex int) error  { return nil }
func (q *fakeMysqlQueryRepo) GetMerchantsByColumn(columnID int) ([]models.Merchant, error) {
	return nil, nil
}
func (q *fakeMysqlQueryRepo) CreateMerchant(m *models.Merchant) error        { return nil }
func (q *fakeMysqlQueryRepo) GetMaxMerchantPriority() (int, error)           { return 0, nil }
func (q *fakeMysqlQueryRepo) GetTransactions() ([]models.Transaction, error) { return nil, nil }
func (q *fakeMysqlQueryRepo) UpdateTransactionTables(trans []*models.Transaction) error {
	q.updateCalls++
	q.updatedTrans = trans
	return q.updateErr
}
func (q *fakeMysqlQueryRepo) GetNameMapToColumn() (map[string]string, error) {
	return q.nameMap, q.nameMapErr
}

func Test_LoadTransactions_SheetReadError(t *testing.T) {
	fakeErr := errors.New("could not get values")
	spm := &fakeSheetsProvider{getValuesErr: fakeErr}
	rs := &sheets_service.RegisterService{Provider: spm}
	us := &UpdateService{RegisterService: rs}
	sheet := &sheets_service.RegisterSheet{}

	_, _, got := us.LoadTransactions(sheet)
	if got == nil {
		t.Fatalf("got: %v, want: %v", got, nil)
	}
	if !errors.Is(got, fakeErr) {
		t.Fatalf("got: %v, want: %v", got, fakeErr)
	}
}

func Test_LoadTransactions_NoTransactions(t *testing.T) {
	// create the CSV files, headers only, in the tmp dir
	tmpDir := t.TempDir()
	fidelityFile := "Fidelity.csv"
	chaseFile := "Chase.csv"
	wfFile := `Wells Fargo.csv`
	f, err := os.CreateTemp(tmpDir, fidelityFile)
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer f.Close()
	c, err := os.CreateTemp(tmpDir, chaseFile)
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer c.Close()
	w, err := os.CreateTemp(tmpDir, wfFile)
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer w.Close()

	err = os.WriteFile(filepath.Join(tmpDir, fidelityFile), []byte(`"Date","Transaction","Name","Memo","Amount"`), 0644)
	if err != nil {
		t.Fatalf("failed to write %s file: %v", fidelityFile, err)
	}
	err = os.WriteFile(filepath.Join(tmpDir, chaseFile), []byte(`Transaction Date,Post Date,Description,Category,Type,Amount,Memo`), 0644)
	if err != nil {
		t.Fatalf("failed to write %s file: %v", chaseFile, err)
	}
	err = os.WriteFile(filepath.Join(tmpDir, wfFile), []byte(``), 0644)
	if err != nil {
		t.Fatalf("failed to write %s file: %v", wfFile, err)
	}

	spm := &fakeSheetsProvider{
		getValues: [][]any{{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}},
	}
	rs := &sheets_service.RegisterService{Provider: spm}

	us := &UpdateService{
		RegisterService: rs,
		Progress:        func(_ string) {},
		Config: &config.Config{
			FinanceDir: tmpDir,
			Banks: map[string]config.Bank{
				"fidelity": {
					ID:          "fidelity",
					Name:        "Fidelity",
					CSVFileName: fidelityFile,
				},
				"chase": {
					ID:          "chase",
					Name:        "Chase",
					CSVFileName: chaseFile,
				},
				"wellsfargo": {
					ID:          "wellsfargo",
					Name:        "Wells Fargo",
					CSVFileName: wfFile,
				},
			},
		},
		Options: &config.Options{UseCSVFiles: true},
	}
	sheet := &sheets_service.RegisterSheet{}

	_, _, got := us.LoadTransactions(sheet)
	if got != nil {
		t.Fatalf("unexpected error returned. got: %v, want: %v", got, nil)
	}
}

func Test_LoadTransactions_HasTransactions(t *testing.T) {
	// create the CSV files, headers only, in the tmp dir
	tmpDir := t.TempDir()
	fidelityFile := "Fidelity.csv"
	chaseFile := "Chase.csv"
	wfFile := `Wells Fargo.csv`
	f, err := os.CreateTemp(tmpDir, fidelityFile)
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer f.Close()
	c, err := os.CreateTemp(tmpDir, chaseFile)
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer c.Close()
	w, err := os.CreateTemp(tmpDir, wfFile)
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer w.Close()

	err = os.WriteFile(filepath.Join(tmpDir, fidelityFile),
		[]byte(`"Date","Transaction","Name","Memo","Amount"
"2026-01-02","DEBIT","SQ *PLUME HOUSE VAPE C Winchester    VA","24692165365100996015344; 05999; ; ; ; ;","-26.32"`), 0644)
	if err != nil {
		t.Fatalf("failed to write %s file: %v", fidelityFile, err)
	}
	err = os.WriteFile(filepath.Join(tmpDir, chaseFile), []byte(`Transaction Date,Post Date,Description,Category,Type,Amount,Memo
10/01/2026,10/02/2026,Amazon web services,Bills & Utilities,Sale,-0.16,
10/01/2026,10/02/2026,Amazon.com*596611C21,Shopping,Sale,-52.48,`), 0644)
	if err != nil {
		t.Fatalf("failed to write %s file: %v", chaseFile, err)
	}
	err = os.WriteFile(filepath.Join(tmpDir, wfFile), []byte(``), 0644)
	if err != nil {
		t.Fatalf("failed to write %s file: %v", wfFile, err)
	}

	spm := &fakeSheetsProvider{
		getValues: [][]any{{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}},
	}
	rs := &sheets_service.RegisterService{Provider: spm}

	us := &UpdateService{
		RegisterService: rs,
		Progress:        func(_ string) {},
		Config: &config.Config{
			FinanceDir: tmpDir,
			Banks: map[string]config.Bank{
				"fidelity": {
					ID:          "fidelity",
					Name:        "Fidelity",
					CSVFileName: fidelityFile,
				},
				"chase": {
					ID:          "chase",
					Name:        "Chase",
					CSVFileName: chaseFile,
				},
				"wellsfargo": {
					ID:          "wellsfargo",
					Name:        "Wells Fargo",
					CSVFileName: wfFile,
				},
			},
		},
		Options: &config.Options{UseCSVFiles: true},
	}
	sheet := &sheets_service.RegisterSheet{}

	trans, _, got := us.LoadTransactions(sheet)
	if got != nil {
		t.Fatalf("unexpected error returned. got: %v, want: %v", got, nil)
	}
	if got != nil {
		t.Fatalf("unexpected error returned. got: %v, want: %v", got, nil)
	}
	if len(trans) != 3 {
		t.Fatalf("got %d transactions, want 3", len(trans))
	}
	if trans[0].Amount != -26.32 || trans[1].BankName != "Amazon web services" || trans[2].Date != "10/01/26" {
		t.Fatalf("got wrong transaction values. want: %v, %v, %v, got: %v, %v, %v", -26.32, "Amazon web services", "10/01/26",
			trans[0].Amount, trans[1].BankName, trans[2].Date)
	}
}

func Test_LoadTransactions_CheckTransactionsCSV(t *testing.T) {
	// create the CSV files, headers only, in the tmp dir
	tmpDir := t.TempDir()
	fidelityFile := "Fidelity.csv"
	chaseFile := "Chase.csv"
	wfFile := `Wells Fargo.csv`
	transFile := "transactions.csv"
	f, err := os.CreateTemp(tmpDir, fidelityFile)
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer f.Close()
	c, err := os.CreateTemp(tmpDir, chaseFile)
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer c.Close()
	w, err := os.CreateTemp(tmpDir, wfFile)
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer w.Close()

	err = os.WriteFile(filepath.Join(tmpDir, fidelityFile),
		[]byte(`"Date","Transaction","Name","Memo","Amount"
"2026-01-02","DEBIT","SQ *PLUME HOUSE VAPE C Winchester    VA","24692165365100996015344; 05999; ; ; ; ;","-26.32"`), 0644)
	if err != nil {
		t.Fatalf("failed to write %s file: %v", fidelityFile, err)
	}
	err = os.WriteFile(filepath.Join(tmpDir, chaseFile), []byte(`Transaction Date,Post Date,Description,Category,Type,Amount,Memo
10/01/2026,10/02/2026,Amazon web services,Bills & Utilities,Sale,-0.16,
10/01/2026,10/02/2026,Amazon.com*596611C21,Shopping,Sale,-52.48,`), 0644)
	if err != nil {
		t.Fatalf("failed to write %s file: %v", chaseFile, err)
	}
	err = os.WriteFile(filepath.Join(tmpDir, wfFile), []byte(``), 0644)
	if err != nil {
		t.Fatalf("failed to write %s file: %v", wfFile, err)
	}

	spm := &fakeSheetsProvider{
		getValues: [][]any{{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}},
	}
	rs := &sheets_service.RegisterService{Provider: spm}

	us := &UpdateService{
		RegisterService: rs,
		Progress:        func(_ string) {},
		Config: &config.Config{
			FinanceDir: tmpDir,
			Banks: map[string]config.Bank{
				"fidelity": {
					ID:          "fidelity",
					Name:        "Fidelity",
					CSVFileName: fidelityFile,
				},
				"chase": {
					ID:          "chase",
					Name:        "Chase",
					CSVFileName: chaseFile,
				},
				"wellsfargo": {
					ID:          "wellsfargo",
					Name:        "Wells Fargo",
					CSVFileName: wfFile,
				},
			},
		},
		Options: &config.Options{UseCSVFiles: true},
	}
	sheet := &sheets_service.RegisterSheet{}

	_, _, got := us.LoadTransactions(sheet)
	if got != nil {
		t.Fatalf("unexpected error returned. got: %v, want: %v", got, nil)
	}
	file, err := os.OpenFile(filepath.Join(tmpDir, transFile), os.O_RDONLY, 0644)
	if err != nil {
		t.Fatalf("failed to open %s file: %v", transFile, err)
	}
	defer file.Close()
	contents := bufio.NewReader(file)
	var want = [3][]byte{
		[]byte(`Fidelity,01/02/26,SQ *PLUME HOUSE VAPE C Winchester    VA,-26.32,0.00,0.00,26.32`),
		[]byte(`Chase,10/01/26,Amazon web services,-0.16,0.00,0.00,0.16`),
		[]byte(`Chase,10/01/26,Amazon.com*596611C21,-52.48,0.00,0.00,52.48`),
	}
	for i := range 3 {
		gotLine, _, err := contents.ReadLine()
		if err != nil {
			t.Fatalf("failed to read %s file: %v", transFile, err)
		}
		if !bytes.Equal(want[i], gotLine) {
			t.Fatalf("got wrong %s value. want: %v, got: %v",
				transFile, string(want[i]), string(gotLine))
		}
	}
}

// newPlaidTestClient builds a banking Client whose Plaid API calls go to
// the given test server instead of Plaid. plaid-go honors the
// Configuration.Servers override (its OperationServers map is empty by
// default), so no production code changes are needed to redirect it.
func newPlaidTestClient(t *testing.T, handler http.HandlerFunc) (*banking.Client, *httptest.Server) {
	t.Helper()
	ts := httptest.NewServer(handler)
	t.Cleanup(ts.Close)
	cfg := plaid.NewConfiguration()
	cfg.Servers = plaid.ServerConfigurations{{URL: ts.URL}}
	return &banking.Client{
		PlaidClient: plaid.NewAPIClient(cfg),
		Banks: map[string]config.Bank{
			"wellsfargo": {ID: "wellsfargo", Name: "Wells Fargo", AccessToken: "token-wf"},
			"chase":      {ID: "chase", Name: "Chase", AccessToken: "token-chase"},
		},
		Progress: func(_ string) {},
	}, ts
}

func writeJSON(t *testing.T, w http.ResponseWriter, status int, v any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		t.Fatalf("encode fake plaid response: %v", err)
	}
}

func Test_LoadTransactions_Plaid(t *testing.T) {
	tmpDir := t.TempDir()
	fidelityFile := "Fidelity.csv"
	f, err := os.CreateTemp(tmpDir, fidelityFile)
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer f.Close()
	err = os.WriteFile(filepath.Join(tmpDir, fidelityFile),
		[]byte(`"Date","Transaction","Name","Memo","Amount"
"2026-10-02","DEBIT","SQ *PLUME HOUSE VAPE C Winchester    VA","24692165365100996015344; 05999; ; ; ; ;","-26.32"`), 0644)
	if err != nil {
		t.Fatalf("failed to write %s file: %v", fidelityFile, err)
	}

	banks := map[string]config.Bank{
		"fidelity": {
			ID:          "fidelity",
			Name:        "Fidelity",
			CSVFileName: fidelityFile,
		},
		"chase": {
			ID:   "chase",
			Name: "Chase",
		},
		"wellsfargo": {
			ID:   "wellsfargo",
			Name: "Wells Fargo",
		},
	}
	spm := &fakeSheetsProvider{
		getValues: [][]any{{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}},
	}
	rs := &sheets_service.RegisterService{Provider: spm}
	sheet := &sheets_service.RegisterSheet{}
	c, _ := newPlaidTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/transactions/get" {
			t.Errorf("path = %q, want /transactions/get", r.URL.Path)
		}
		writeJSON(t, w, http.StatusOK, map[string]any{
			"transactions": []map[string]any{
				{
					"name":           "Spotify",
					"date":           "2026-09-24",
					"amount":         11.99,
					"pending":        false,
					"transaction_id": "lPNjeW1nR6CDn5okmGQ6hEpMo4lLNoSrzqDje",
				},
				{
					"name":           "Plaid",
					"date":           "2026-10-04",
					"amount":         5.00,
					"pending":        true,
					"transaction_id": "mPNjeW1nR6CDn5okmGQ6hEpMo4lLNoSrzqDje",
				},
			},
			"total_transactions": 2,
		})
	})
	us := &UpdateService{
		RegisterService: rs, // your fake sheet provider, as before
		BankClient:      c,  // the fake-server client
		Progress:        func(_ string) {},
		Config:          &config.Config{FinanceDir: tmpDir, Banks: banks}, // Fidelity fixture still needed
		Options:         &config.Options{UseCSVFiles: false},
	}

	t.Run("success", func(t *testing.T) {
		gotTrans, gotPending, err := us.LoadTransactions(sheet)
		if err != nil {
			t.Fatalf("GetTransactions = %v, want nil", err)
		}
		if len(gotTrans) != 3 {
			t.Fatalf("got %d posted transactions, want 3", len(gotTrans))
		}
		if len(gotPending) != 2 {
			t.Fatalf("got %d pending transactions, want 2", len(gotPending))
		}
		if gotTrans[0].Name != "SQ *PLUME HOUSE VAPE C Winchester    VA" || gotTrans[0].Date != "10/02/26" || gotTrans[0].Pending != false {
			t.Errorf("[0] Name: %v, Date: %v, Pending: %v, want Name: %v, Date: %v, Pending: %v",
				gotTrans[0].Name, gotTrans[0].Date, gotTrans[0].Pending, "SQ *PLUME HOUSE VAPE C Winchester    VA", "10/02/26", false)
		}
		if gotTrans[1].Name != "Spotify" || gotTrans[1].Date != "09/24/26" || gotTrans[1].Pending != false {
			t.Errorf("[1] Name: %v, Date: %v, Pending: %v, want Name: %v, Date: %v, Pending: %v",
				gotTrans[1].Name, gotTrans[1].Date, gotTrans[1].Pending, "Spotify", "09/24/26", false)
		}
		if gotPending[0].Name != "Plaid" || gotPending[0].Date != "10/04/26" || gotPending[0].Pending != true {
			t.Errorf("1 Name: %v, Date: %v, Pending: %v, want Name: %v, Date: %v, Pending: %v",
				gotPending[0].Name, gotPending[0].Date, gotPending[0].Pending, "Plaid", "10/04/26", true)
		}
	})
}

func Test_PrepareTransactions(t *testing.T) {
	fakeErr := errors.New("could not get lookup data")

	us := UpdateService{
		BankClient: &banking.Client{
			IncomeSources: []config.IncomeSource{
				{
					Name:         "Taylor House Paycheck",
					MatchPattern: "Winchester Eats PAYROLL",
					Frequency:    "biweekly",
				},
				{
					Name:         "Social Security Administration",
					MatchPattern: "Social Security Administration",
					Frequency:    "monthly",
				},
			},
		},
		QueryRepo: &fakeMysqlQueryRepo{lookupErr: fakeErr},
		Progress:  func(_ string) {},
	}

	t.Run("error", func(t *testing.T) {
		_, _, err := us.PrepareTransactions(nil, nil)
		if !errors.Is(err, fakeErr) {
			t.Errorf("PrepareTransactions error = %v, want %v", err, fakeErr)
		}
	})

	// The remaining subtests need the lookup to succeed.
	us.QueryRepo = &fakeMysqlQueryRepo{}

	// Income source matching
	t.Run("income source matching", func(t *testing.T) {
		var entries []*sheets_service.RegisterEntry
		trans := []*models.Transaction{
			{
				Source:        "Wells Fargo",
				Name:          "",
				BankName:      "Winchester Eats PAYROLL",
				Date:          "10/02/26",
				TransactionID: "1001",
				Key:           "wellsfargo:10/02/26:1000",
			},
		}

		got, gotCors, err := us.PrepareTransactions(trans, entries)
		if err != nil {
			t.Fatalf("PrepareTransactions: error = %v, want nil", err)
		}
		if len(got) != 1 {
			t.Fatalf("PrepareTransactions: got %d posted transactions, want 1", len(got))
		}
		if got[0].Name != "Taylor House Paycheck" {
			t.Errorf("PrepareTransactions: got[0].Name = %v, want Taylor House Paycheck", got[0].Name)
		}
		if len(gotCors) != 0 {
			t.Errorf("PrepareTransactions: got %d corrected transactions, want 0", len(gotCors))
		}
	})

	// Date Correction
	t.Run("date correction", func(t *testing.T) {
		entries := []*sheets_service.RegisterEntry{
			{
				RowID:         20,
				Source:        "Chase",
				Name:          "Amazon",
				Date:          "10/04/26",
				Key:           "chase:10/04/26:1000",
				TransactionID: "1002",
			},
		}
		trans := []*models.Transaction{
			{
				Name:          "",
				Source:        "Chase",
				BankName:      "Amazon.com*5P59L1ZW2",
				Date:          "10/06/26",
				TransactionID: "1002",
				Key:           "chase:10/06/26:50.00",
			},
		}

		got, gotCors, err := us.PrepareTransactions(trans, entries)
		if err != nil {
			t.Fatalf("PrepareTransactions: error = %v, want nil", err)
		}
		if got != nil {
			t.Fatalf("PrepareTransactions: got %v posted transactions, want nil", got)
		}
		if len(gotCors) != 1 {
			t.Fatalf("PrepareTransactions: got %d corrected transactions, want 1", len(gotCors))
		}
		if gotCors[0].NewDate != "10/06/26" {
			t.Errorf("PrepareTransactions: gotCors[0].NewDate = %v, want 10/06/26", gotCors[0].NewDate)
		}
	})

	// Dedupe drop
	t.Run("dedupe drop", func(t *testing.T) {
		entries := []*sheets_service.RegisterEntry{
			{
				RowID:         20,
				Source:        "Chase",
				Name:          "Amazon",
				Date:          "10/04/26",
				Key:           "chase:10/04/26:1000",
				TransactionID: "1002",
			},
		}
		trans := []*models.Transaction{
			{
				Name:          "",
				BankName:      "Amazon.com*5P59L1ZW2",
				Date:          "10/04/26",
				TransactionID: "1002",
				Key:           "chase:10/04/26:50.00",
			},
		}

		got, gotCors, err := us.PrepareTransactions(trans, entries)
		if err != nil {
			t.Fatalf("PrepareTransactions: error = %v, want nil", err)
		}
		if got != nil {
			t.Fatalf("PrepareTransactions: got posted transactions %v, want nil", got)
		}
		if gotCors != nil {
			t.Fatalf("PrepareTransactions: got %d corrected transactions, want nil", len(gotCors))
		}
	})

	// Chase payment drop
	t.Run("chase payment drop", func(t *testing.T) {
		var entries []*sheets_service.RegisterEntry
		trans := []*models.Transaction{
			{
				Source:        "Chase",
				Name:          "",
				BankName:      "Payment Thank You Bill Pa",
				Date:          "10/21/26",
				TransactionID: "2020",
				Key:           "chase:10/21/26:78.00",
			},
		}

		got, gotCors, err := us.PrepareTransactions(trans, entries)
		if err != nil {
			t.Fatalf("PrepareTransactions: error = %v, want nil", err)
		}
		if got != nil || len(got) != 0 {
			t.Fatalf("PrepareTransactions: got %v posted transactions, want nil", len(got))
		}
		if gotCors != nil {
			t.Fatalf("PrepareTransactions: got %d corrected transactions, want nil", len(gotCors))
		}
	})

	// survivors sorted
	t.Run("survivors sorted", func(t *testing.T) {
		var entries []*sheets_service.RegisterEntry
		trans := []*models.Transaction{
			{
				Source:        "Fidelity",
				Name:          "Better Business Bureau",
				BankName:      "Better Business Bureau",
				Date:          "10/02/26",
				TransactionID: "1001",
				Key:           "fidelity:10/02/26:78.00",
			},
			{
				Source:        "Chase",
				Name:          "AAA",
				BankName:      "AAA",
				Date:          "10/03/26",
				TransactionID: "1002",
				Key:           "chase:10/03/26:78.00",
			},
			{
				Source:        "Wells Fargo",
				Name:          "Catoctin Zoo",
				BankName:      "Catoctin Zoo",
				Date:          "10/01/26",
				TransactionID: "1003",
				Key:           "wellsfargo:10/01/26:78.00",
			},
		}

		got, gotCors, err := us.PrepareTransactions(trans, entries)
		if err != nil {
			t.Fatalf("PrepareTransactions: error = %v, want nil", err)
		}
		if got == nil {
			t.Fatalf("PrepareTransactions: got nil posted transactions, want non-nil")
		}
		if len(got) != 3 {
			t.Fatalf("PrepareTransactions: got %d posted transactions, want 3", len(got))
		}
		if gotCors != nil {
			t.Fatalf("PrepareTransactions: got %d corrected transactions, want nil", len(gotCors))
		}
		if got[0].BankName != "Catoctin Zoo" {
			t.Errorf("PrepareTransactions: got[0].BankName = %v, want Catoctin Zoo", got[0].BankName)
		}
	})

	//date correction guard: same date → no correction
	t.Run("same date → no correction", func(t *testing.T) {
		entries := []*sheets_service.RegisterEntry{
			{
				Source:        "Chase",
				Name:          "AAA",
				Date:          "10/03/26",
				Key:           "chase:10/03/26:1000",
				TransactionID: "",
			},
		}
		trans := []*models.Transaction{
			{
				Source:        "Chase",
				Name:          "AAA",
				BankName:      "AAA",
				Date:          "10/03/26",
				TransactionID: "",
				Key:           "chase:10/03/26:1000",
			},
		}

		got, gotCors, err := us.PrepareTransactions(trans, entries)
		if err != nil {
			t.Fatalf("PrepareTransactions: error = %v, want nil", err)
		}
		if got != nil {
			t.Fatalf("PrepareTransactions: got posted transactions %v, want nil", got)
		}
		if gotCors != nil {
			t.Fatalf("PrepareTransactions: got %d corrected transactions, want nil", len(gotCors))
		}
	})

	//isCheck guard: IsCheck → no correction
	t.Run("IsCheck → no correction", func(t *testing.T) {
		entries := []*sheets_service.RegisterEntry{
			{
				Source:        "1001",
				Name:          "Check",
				Date:          "10/03/26",
				Key:           "wellsfargo:10/03/26:1000",
				TransactionID: "2020",
			},
		}
		trans := []*models.Transaction{
			{
				Source:        "1002",
				Name:          "Check",
				BankName:      "CHECK",
				Date:          "10/03/26",
				Key:           "wellsfargo:10/03/26:1000",
				TransactionID: "2020",
			},
		}

		got, gotCors, err := us.PrepareTransactions(trans, entries)
		if err != nil {
			t.Fatalf("PrepareTransactions: error = %v, want nil", err)
		}
		if got != nil {
			t.Fatalf("PrepareTransactions: got posted transactions %v, want nil", got)
		}
		if gotCors != nil {
			t.Fatalf("PrepareTransactions: got %d corrected transactions, want nil", len(gotCors))
		}
	})
}

// ---- 18.7 error-path and bulk coverage additions ----

// Sentinels for the error branches below; production code wraps them with
// %w, so errors.Is stays ungated.
var (
	errFakeSpreadsheet = errors.New("could not get spreadsheet")
	errFakeBudget      = errors.New("could not get budget values")
	errFakeRegister    = errors.New("could not get register values")
	errFakeBatch       = errors.New("could not batch update")
	errFakeUpdate      = errors.New("could not update value")
	errFakeColumns     = errors.New("could not get columns")
	errFakeNameMap     = errors.New("could not get name map")
	errFakeDB          = errors.New("could not update transaction tables")
)

// recordedUpdate is one Provider.Update call.
type recordedUpdate struct {
	range_ string
	values [][]any
}

// recordingSheetsProvider is a sheets provider fake whose reads are routed
// by caller-supplied functions and whose writes are recorded, so tests can
// drive RegisterService/BudgetService through UpdateSpreadsheet and friends
// without touching the real Sheets API. Pointer receiver: the recordings
// must survive the call.
type recordingSheetsProvider struct {
	getSpreadsheetFn  func(tabName string) (*sheets_provider.Spreadsheet, error)
	getValuesFn       func(range_ string) ([][]any, error)
	getFormulaFn      func(range_ string) ([][]any, error)
	batchUpdateErr    error
	batchUpdateFailOn int // 1-based call number to fail; 0 = never
	updateErr         error
	updateFailOn      int // 1-based Update call number to fail; 0 = every call

	batchUpdateCalls int
	updates          []recordedUpdate
}

func (p *recordingSheetsProvider) GetValues(range_ string) ([][]any, error) {
	if p.getValuesFn != nil {
		return p.getValuesFn(range_)
	}
	return nil, nil
}

func (p *recordingSheetsProvider) GetFormula(range_ string) ([][]any, error) {
	if p.getFormulaFn != nil {
		return p.getFormulaFn(range_)
	}
	return nil, nil
}

func (p *recordingSheetsProvider) GetSpreadsheet(tabName string) (*sheets_provider.Spreadsheet, error) {
	if p.getSpreadsheetFn != nil {
		return p.getSpreadsheetFn(tabName)
	}
	return &sheets_provider.Spreadsheet{SheetID: 7, RowCount: 1000, ColumnCount: 10}, nil
}

func (p *recordingSheetsProvider) BatchUpdate(updateReq *sheets.BatchUpdateSpreadsheetRequest) (*sheets.BatchUpdateSpreadsheetResponse, error) {
	p.batchUpdateCalls++
	if p.batchUpdateErr != nil && (p.batchUpdateFailOn == 0 || p.batchUpdateCalls == p.batchUpdateFailOn) {
		return nil, p.batchUpdateErr
	}
	return &sheets.BatchUpdateSpreadsheetResponse{}, nil
}

func (p *recordingSheetsProvider) Update(writeRange string, vRange *sheets.ValueRange) (*sheets.UpdateValuesResponse, error) {
	p.updates = append(p.updates, recordedUpdate{range_: writeRange, values: vRange.Values})
	if p.updateErr != nil && (p.updateFailOn == 0 || len(p.updates) == p.updateFailOn) {
		return nil, p.updateErr
	}
	return &sheets.UpdateValuesResponse{}, nil
}

func (p *recordingSheetsProvider) ClearValues(range_ string) error {
	return nil
}

// spreadsheetFixture wires an UpdateService whose sheet reads come from
// canned values: one budget row (Groceries) and one register row, under the
// fixed A-J register headers.
type spreadsheetFixture struct {
	provider *recordingSheetsProvider
	repo     *fakeMysqlQueryRepo
	svc      *UpdateService
}

func spreadsheetFixtureValues(t *testing.T) func(string) ([][]any, error) {
	t.Helper()
	return func(range_ string) ([][]any, error) {
		switch {
		case range_ == "Budget!A1:Z1" || range_ == "Budget!A1:C1":
			return [][]any{{"Category", "Weekly", "Bi-Weekly", "Monthly"}}, nil
		case strings.HasPrefix(range_, "Budget!A5:"):
			return [][]any{{"Groceries", 120.0, 240.0, 520.0}}, nil
		case range_ == "Register!A4:J4":
			return [][]any{{"Transaction ID", "Source", "Date", "Description", "Withdrawals", "Deposits", "Credit Purchases", "Register", "Cleared", "Delta"}}, nil
		case strings.HasPrefix(range_, "Register!A5:"):
			return [][]any{{"txn-1", "WellsFargo", "10/01/26", "Amazon", 0.0, 0.0, 0.0, 0.0, 0.0, 0.0}}, nil
		default:
			t.Errorf("unexpected GetValues range %q", range_)
			return nil, fmt.Errorf("unexpected GetValues range %q", range_)
		}
	}
}

func newSpreadsheetFixture(t *testing.T) *spreadsheetFixture {
	t.Helper()
	p := &recordingSheetsProvider{
		getValuesFn: spreadsheetFixtureValues(t),
		getFormulaFn: func(range_ string) ([][]any, error) {
			return [][]any{{"=SUM(H5:H5)", "=SUM(I5:I5)", "=SUM(J5:J5)"}}, nil
		},
	}
	repo := &fakeMysqlQueryRepo{
		columns: []models.Column{{ID: 1, Name: "Groceries", ColumnIndex: 11, Color: "#ffffff"}},
		nameMap: map[string]string{"Amazon": "Groceries"},
	}
	svc := &UpdateService{
		RegisterService: &sheets_service.RegisterService{Provider: p},
		BudgetService:   &sheets_service.BudgetService{Provider: p},
		QueryRepo:       repo,
		Progress:        func(_ string) {},
		Config: &config.Config{
			RegisterStartRow: 5,
			RegisterEndRow:   500,
			BudgetStartRow:   5,
			BudgetEndRow:     50,
		},
		Options: &config.Options{},
	}
	return &spreadsheetFixture{provider: p, repo: repo, svc: svc}
}

// writeCSVFixture writes one CSV fixture file into dir.
func writeCSVFixture(t *testing.T, dir, name, contents string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(contents), 0644); err != nil {
		t.Fatalf("failed to write %s: %v", name, err)
	}
}

func csvBanks() map[string]config.Bank {
	return map[string]config.Bank{
		"fidelity":   {ID: "fidelity", Name: "Fidelity", CSVFileName: "Fidelity.csv"},
		"chase":      {ID: "chase", Name: "Chase", CSVFileName: "Chase.csv"},
		"wellsfargo": {ID: "wellsfargo", Name: "Wells Fargo", CSVFileName: "Wells Fargo.csv"},
	}
}

const fidelityCSVFixture = `"Date","Transaction","Name","Memo","Amount"
"2026-01-02","DEBIT","SQ *PLUME HOUSE VAPE C Winchester    VA","24692165365100996015344; 05999; ; ; ; ;","-26.32"`

const chaseCSVHeader = `Transaction Date,Post Date,Description,Category,Type,Amount,Memo`

// newPlaidErrorClient builds a banking Client whose Plaid calls hit a test
// server that always answers 500 with the given error message in the body.
// Plaid echoes the body into the returned error, so the message survives
// into GetTransactions' aggregated error.
func newPlaidErrorClient(t *testing.T, message string) *banking.Client {
	t.Helper()
	c, _ := newPlaidTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, http.StatusInternalServerError, map[string]any{
			"error_type":    "API_ERROR",
			"error_code":    "INTERNAL_SERVER_ERROR",
			"error_message": message,
		})
	})
	return c
}

// newBalanceTestClient builds a banking Client whose Plaid balance calls
// hit a test server returning the given Wells Fargo checking balance.
func newBalanceTestClient(t *testing.T, amount float64) *banking.Client {
	t.Helper()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/accounts/balance/get" {
			t.Errorf("path = %q, want /accounts/balance/get", r.URL.Path)
		}
		writeJSON(t, w, http.StatusOK, map[string]any{
			"accounts": []map[string]any{
				{"account_id": "acc1", "balances": map[string]any{"current": amount, "iso_currency_code": "USD"}},
			},
		})
	}))
	t.Cleanup(ts.Close)
	cfg := plaid.NewConfiguration()
	cfg.Servers = plaid.ServerConfigurations{{URL: ts.URL}}
	return &banking.Client{
		PlaidClient: plaid.NewAPIClient(cfg),
		Banks: map[string]config.Bank{
			"wellsfargo": {ID: "wellsfargo", Name: "Wells Fargo", AccessToken: "token-wf", AccountID: "acc1"},
		},
		Progress: func(_ string) {},
	}
}

// newNoTokenBankClient builds a banking Client with no Plaid access token,
// so GetBalances reports the Wells Fargo balance as unavailable.
func newNoTokenBankClient() *banking.Client {
	return &banking.Client{
		Banks: map[string]config.Bank{
			"wellsfargo": {ID: "wellsfargo", Name: "Wells Fargo"},
		},
		Progress: func(_ string) {},
	}
}

// 18.7: a Plaid failure in getTransactions must propagate out of
// FetchTransactions instead of being swallowed.
func Test_FetchTransactions_PlaidError(t *testing.T) {
	tmpDir := t.TempDir()
	writeCSVFixture(t, tmpDir, "Fidelity.csv", fidelityCSVFixture)

	us := &UpdateService{
		BankClient: newPlaidErrorClient(t, "boom"),
		Progress:   func(_ string) {},
		Config: &config.Config{
			FinanceDir: tmpDir,
			Banks:      csvBanks(),
		},
		Options: &config.Options{UseCSVFiles: false},
	}

	trans, pending, err := us.FetchTransactions([]string{models.WellsFargoID, models.ChaseID}, "2026-08-01", "2026-11-05")
	if err == nil {
		t.Fatalf("FetchTransactions = nil error, want Plaid failure to propagate")
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Errorf("FetchTransactions error = %v, want it to contain the Plaid failure", err)
	}
	if trans != nil || pending != nil {
		t.Errorf("FetchTransactions = (%v, %v), want (nil, nil) on error", trans, pending)
	}
}

// 18.7: the same Plaid failure must propagate all the way out of
// LoadTransactions.
func Test_LoadTransactions_PlaidError(t *testing.T) {
	tmpDir := t.TempDir()
	writeCSVFixture(t, tmpDir, "Fidelity.csv", fidelityCSVFixture)

	spm := &fakeSheetsProvider{
		getValues: [][]any{{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}},
	}
	us := &UpdateService{
		RegisterService: &sheets_service.RegisterService{Provider: spm},
		BankClient:      newPlaidErrorClient(t, "boom"),
		Progress:        func(_ string) {},
		Config: &config.Config{
			FinanceDir: tmpDir,
			Banks:      csvBanks(),
		},
		Options: &config.Options{UseCSVFiles: false},
	}

	trans, pending, err := us.LoadTransactions(&sheets_service.RegisterSheet{})
	if err == nil {
		t.Fatalf("LoadTransactions = nil error, want Plaid failure to propagate")
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Errorf("LoadTransactions error = %v, want it to contain the Plaid failure", err)
	}
	if trans != nil || pending != nil {
		t.Errorf("LoadTransactions = (%v, %v), want (nil, nil) on error", trans, pending)
	}
}

// The CSV side of FetchTransactions: a missing Fidelity CSV must fail the
// fetch instead of returning partial results.
func Test_FetchTransactions_CSVError(t *testing.T) {
	tmpDir := t.TempDir() // no Fidelity.csv on disk

	us := &UpdateService{
		Progress: func(_ string) {},
		Config: &config.Config{
			FinanceDir: tmpDir,
			Banks:      csvBanks(),
		},
		Options: &config.Options{UseCSVFiles: false},
	}

	_, _, err := us.FetchTransactions([]string{models.WellsFargoID}, "2026-08-01", "2026-11-05")
	if err == nil {
		t.Fatalf("FetchTransactions = nil error, want missing Fidelity CSV to fail")
	}
	if !strings.Contains(err.Error(), "could not read CSV file") {
		t.Errorf("FetchTransactions error = %v, want it to mention the unreadable CSV", err)
	}
}

// LoadTransactions must surface a failure writing transactions.csv.
func Test_LoadTransactions_WriteCSVError(t *testing.T) {
	tmpDir := t.TempDir()
	writeCSVFixture(t, tmpDir, "Fidelity.csv", fidelityCSVFixture)
	writeCSVFixture(t, tmpDir, "Chase.csv", chaseCSVHeader)
	writeCSVFixture(t, tmpDir, "Wells Fargo.csv", ``)
	// transactions.csv exists as a directory, so os.Create fails.
	if err := os.Mkdir(filepath.Join(tmpDir, "transactions.csv"), 0755); err != nil {
		t.Fatalf("mkdir transactions.csv: %v", err)
	}

	spm := &fakeSheetsProvider{
		getValues: [][]any{{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}},
	}
	us := &UpdateService{
		RegisterService: &sheets_service.RegisterService{Provider: spm},
		Progress:        func(_ string) {},
		Config: &config.Config{
			FinanceDir: tmpDir,
			Banks:      csvBanks(),
		},
		Options: &config.Options{UseCSVFiles: true},
	}

	_, _, err := us.LoadTransactions(&sheets_service.RegisterSheet{})
	if err == nil {
		t.Fatalf("LoadTransactions = nil error, want transactions.csv create failure to propagate")
	}
	if !strings.Contains(err.Error(), "error creating file") {
		t.Errorf("LoadTransactions error = %v, want it to mention creating the file", err)
	}
}

func Test_PersistTransactions_Error(t *testing.T) {
	repo := &fakeMysqlQueryRepo{updateErr: errFakeDB}
	s := NewUpdateService(nil, nil, nil, repo, nil, &config.Options{DryRun: false})
	trans := []*models.Transaction{{Source: "FooBank", Name: "Amazon", CreditPurchase: 299.99}}

	got, err := s.PersistTransactions(trans)
	if !errors.Is(err, errFakeDB) {
		t.Fatalf("PersistTransactions error = %v, want %v", err, errFakeDB)
	}
	if got != nil {
		t.Errorf("PersistTransactions = %v, want nil on error", got)
	}
	if repo.updateCalls != 1 {
		t.Errorf("updateCalls = %d, want 1", repo.updateCalls)
	}
}

func Test_NormalizeMerchants_Errors(t *testing.T) {
	tests := []struct {
		name        string
		repo        *fakeMysqlQueryRepo
		bankClient  *banking.Client
		wantErr     error
		wantContain string
	}{
		{
			name:        "bad regex merchant rule",
			repo:        &fakeMysqlQueryRepo{lookupData: []*models.DataRow{{MatchType: "regex", Pattern: "["}}},
			bankClient:  &banking.Client{},
			wantContain: "bad regex",
		},
		{
			name:        "bad regex income source",
			repo:        &fakeMysqlQueryRepo{},
			bankClient:  &banking.Client{IncomeSources: []config.IncomeSource{{Name: "Paycheck", MatchPattern: "[", MatchType: "regex"}}},
			wantContain: "bad regex",
		},
		{
			name:        "columns error",
			repo:        &fakeMysqlQueryRepo{columnsErr: errFakeColumns},
			bankClient:  &banking.Client{},
			wantErr:     errFakeColumns,
			wantContain: "could not get columns",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			us := &UpdateService{BankClient: tt.bankClient, QueryRepo: tt.repo, Progress: func(_ string) {}}
			_, err := us.NormalizeMerchants(nil)
			if err == nil {
				t.Fatalf("NormalizeMerchants = nil error, want error")
			}
			if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Errorf("NormalizeMerchants error = %v, want %v", err, tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantContain) {
				t.Errorf("NormalizeMerchants error = %v, want it to contain %q", err, tt.wantContain)
			}
		})
	}
}

func Test_WriteRawTransactionsCSV_Errors(t *testing.T) {
	one := []*models.Transaction{{
		Source: "Fidelity", Date: "01/02/26", BankName: "SQ *PLUME",
		Amount: -26.32, CreditPurchase: 26.32,
	}}

	t.Run("create error", func(t *testing.T) {
		tmpDir := t.TempDir()
		if err := os.Mkdir(filepath.Join(tmpDir, "transactions.csv"), 0755); err != nil {
			t.Fatalf("mkdir transactions.csv: %v", err)
		}
		us := &UpdateService{Config: &config.Config{FinanceDir: tmpDir}}
		err := us.WriteRawTransactionsCSV(one)
		if err == nil {
			t.Fatalf("WriteRawTransactionsCSV = nil, want create error")
		}
		if !strings.Contains(err.Error(), "error creating file") {
			t.Errorf("WriteRawTransactionsCSV error = %v, want create error", err)
		}
	})

	// /dev/full accepts the open but fails every write with ENOSPC.
	// It does not exist on macOS, where the open instead tries to create
	// it in /dev and fails with "operation not permitted".
	t.Run("flush error", func(t *testing.T) {
		if runtime.GOOS == "darwin" {
			t.Skip("/dev/full does not exist on macOS")
		}
		tmpDir := t.TempDir()
		if err := os.Symlink("/dev/full", filepath.Join(tmpDir, "transactions.csv")); err != nil {
			t.Fatalf("symlink /dev/full: %v", err)
		}
		us := &UpdateService{Config: &config.Config{FinanceDir: tmpDir}}
		err := us.WriteRawTransactionsCSV(one)
		if err == nil {
			t.Fatalf("WriteRawTransactionsCSV = nil, want write error")
		}
		if !strings.Contains(err.Error(), "error writing to file") {
			t.Errorf("WriteRawTransactionsCSV error = %v, want write error", err)
		}
	})

	t.Run("write error", func(t *testing.T) {
		if runtime.GOOS == "darwin" {
			t.Skip("/dev/full does not exist on macOS")
		}
		tmpDir := t.TempDir()
		if err := os.Symlink("/dev/full", filepath.Join(tmpDir, "transactions.csv")); err != nil {
			t.Fatalf("symlink /dev/full: %v", err)
		}
		us := &UpdateService{Config: &config.Config{FinanceDir: tmpDir}}
		var many []*models.Transaction
		for i := 0; i < 200; i++ {
			many = append(many, &models.Transaction{
				Source: "Fidelity", Date: "01/02/26",
				BankName:       "SQ *PLUME HOUSE VAPE C Winchester VA with a very long merchant name to fill the buffer",
				Amount:         -26.32,
				CreditPurchase: 26.32,
			})
		}
		err := us.WriteRawTransactionsCSV(many)
		if err == nil {
			t.Fatalf("WriteRawTransactionsCSV = nil, want write error")
		}
		if !strings.Contains(err.Error(), "error writing to file") {
			t.Errorf("WriteRawTransactionsCSV error = %v, want write error", err)
		}
	})
}

func Test_ApplyDateCorrections(t *testing.T) {
	corrections := []*DateCorrection{
		{NewDate: "10/06/26", Entry: &sheets_service.RegisterEntry{RowID: 20}},
		{NewDate: "10/07/26", Entry: &sheets_service.RegisterEntry{RowID: 42}},
	}

	t.Run("writes each corrected date", func(t *testing.T) {
		p := &recordingSheetsProvider{}
		us := &UpdateService{RegisterService: &sheets_service.RegisterService{Provider: p}}
		if err := us.ApplyDateCorrections("Register", corrections); err != nil {
			t.Fatalf("ApplyDateCorrections = %v, want nil", err)
		}
		want := []recordedUpdate{
			{range_: "Register!C20", values: [][]any{{"10/06/26"}}},
			{range_: "Register!C42", values: [][]any{{"10/07/26"}}},
		}
		if !reflect.DeepEqual(p.updates, want) {
			t.Errorf("updates = %v, want %v", p.updates, want)
		}
	})

	t.Run("update error propagates", func(t *testing.T) {
		spm := &fakeSheetsProvider{}
		us := &UpdateService{RegisterService: &sheets_service.RegisterService{Provider: spm}}
		err := us.ApplyDateCorrections("Register", corrections)
		if err == nil {
			t.Fatalf("ApplyDateCorrections = nil, want error")
		}
		if !strings.Contains(err.Error(), "could not update value") {
			t.Errorf("ApplyDateCorrections error = %v, want update failure", err)
		}
	})
}

func Test_UpdateBalances(t *testing.T) {
	t.Run("happy path writes the checking balance", func(t *testing.T) {
		p := &recordingSheetsProvider{}
		us := &UpdateService{RegisterService: &sheets_service.RegisterService{Provider: p}}
		balances := map[string]banking.Balance{
			models.WellsFargoID: {BankName: "Wells Fargo", Amount: 1234.56},
		}
		if err := us.UpdateBalances("Register", balances); err != nil {
			t.Fatalf("UpdateBalances = %v, want nil", err)
		}
		if len(p.updates) != 1 {
			t.Fatalf("updates = %d, want 1", len(p.updates))
		}
		if p.updates[0].range_ != "Register!G1:G1" {
			t.Errorf("updates[0].range = %q, want %q", p.updates[0].range_, "Register!G1:G1")
		}
		if got := p.updates[0].values[0][0]; got != 1234.56 {
			t.Errorf("updates[0] value = %v, want 1234.56", got)
		}
	})

	t.Run("balance error skips the write", func(t *testing.T) {
		p := &recordingSheetsProvider{}
		us := &UpdateService{RegisterService: &sheets_service.RegisterService{Provider: p}}
		balances := map[string]banking.Balance{
			models.WellsFargoID: {BankName: "Wells Fargo", Error: errors.New("no Plaid access token configured")},
		}
		if err := us.UpdateBalances("Register", balances); err != nil {
			t.Fatalf("UpdateBalances = %v, want nil", err)
		}
		if len(p.updates) != 0 {
			t.Errorf("updates = %d, want 0", len(p.updates))
		}
	})

	t.Run("write error propagates", func(t *testing.T) {
		p := &recordingSheetsProvider{updateErr: errFakeUpdate}
		us := &UpdateService{RegisterService: &sheets_service.RegisterService{Provider: p}}
		balances := map[string]banking.Balance{
			models.WellsFargoID: {BankName: "Wells Fargo", Amount: 1234.56},
		}
		err := us.UpdateBalances("Register", balances)
		if !errors.Is(err, errFakeUpdate) {
			t.Fatalf("UpdateBalances error = %v, want %v", err, errFakeUpdate)
		}
	})
}

func Test_UpdateSpreadsheet(t *testing.T) {
	trans := []*models.Transaction{{
		Source: "WellsFargo", Name: "Amazon", BankName: "Amazon",
		Date: "10/02/26", Withdrawal: 50.00, Amount: 50.00, Budget: -50.00,
	}}

	t.Run("happy path", func(t *testing.T) {
		f := newSpreadsheetFixture(t)
		if err := f.svc.UpdateSpreadsheet(trans); err != nil {
			t.Fatalf("UpdateSpreadsheet = %v, want nil", err)
		}
		if f.provider.batchUpdateCalls != 2 {
			t.Errorf("batchUpdateCalls = %d, want 2 (CopyRows + UpdateRows)", f.provider.batchUpdateCalls)
		}
		if len(f.provider.updates) != 2 {
			t.Fatalf("updates = %d, want 2 (LastUpdate + CheckingDelta cells)", len(f.provider.updates))
		}
		if f.provider.updates[0].range_ != "Register!F1:F1" {
			t.Errorf("updates[0].range = %q, want %q", f.provider.updates[0].range_, "Register!F1:F1")
		}
		wantDate := time.Now().Format("01/02/2006")
		if got := f.provider.updates[0].values[0][0]; got != wantDate {
			t.Errorf("updates[0] value = %v, want %v", got, wantDate)
		}
		if f.provider.updates[1].range_ != "Register!G2:G2" {
			t.Errorf("updates[1].range = %q, want %q", f.provider.updates[1].range_, "Register!G2:G2")
		}
		if got := f.provider.updates[1].values[0][0]; got != "=SUM(G1-I9)" {
			t.Errorf("updates[1] value = %v, want %v", got, "=SUM(G1-I9)")
		}
	})

	errTests := []struct {
		name    string
		setup   func(t *testing.T, f *spreadsheetFixture)
		wantErr error
	}{
		{
			name: "NewRegisterSheet error",
			setup: func(t *testing.T, f *spreadsheetFixture) {
				f.provider.getSpreadsheetFn = func(tabName string) (*sheets_provider.Spreadsheet, error) {
					return nil, errFakeSpreadsheet
				}
			},
			wantErr: errFakeSpreadsheet,
		},
		{
			name: "NewBudgetSheet error",
			setup: func(t *testing.T, f *spreadsheetFixture) {
				f.provider.getSpreadsheetFn = func(tabName string) (*sheets_provider.Spreadsheet, error) {
					if tabName == sheets_service.BudgetTabName {
						return nil, errFakeSpreadsheet
					}
					return &sheets_provider.Spreadsheet{SheetID: 7, RowCount: 1000, ColumnCount: 10}, nil
				}
			},
			wantErr: errFakeSpreadsheet,
		},
		{
			name: "ReadBudgetSheet error",
			setup: func(t *testing.T, f *spreadsheetFixture) {
				orig := f.provider.getValuesFn
				f.provider.getValuesFn = func(range_ string) ([][]any, error) {
					if strings.HasPrefix(range_, "Budget!A5:") {
						return nil, errFakeBudget
					}
					return orig(range_)
				}
			},
			wantErr: errFakeBudget,
		},
		{
			name: "ReadRegisterSheet error",
			setup: func(t *testing.T, f *spreadsheetFixture) {
				orig := f.provider.getValuesFn
				f.provider.getValuesFn = func(range_ string) ([][]any, error) {
					if strings.HasPrefix(range_, "Register!A5:") {
						return nil, errFakeRegister
					}
					return orig(range_)
				}
			},
			wantErr: errFakeRegister,
		},
		{
			name: "CopyRows error",
			setup: func(t *testing.T, f *spreadsheetFixture) {
				f.provider.batchUpdateErr = errFakeBatch
			},
			wantErr: errFakeBatch,
		},
		{
			name: "GetColumns error",
			setup: func(t *testing.T, f *spreadsheetFixture) {
				f.repo.columnsErr = errFakeColumns
			},
			wantErr: errFakeColumns,
		},
		{
			name: "GetNameMapToColumn error",
			setup: func(t *testing.T, f *spreadsheetFixture) {
				f.repo.nameMapErr = errFakeNameMap
			},
			wantErr: errFakeNameMap,
		},
		{
			name: "UpdateRows error",
			setup: func(t *testing.T, f *spreadsheetFixture) {
				f.provider.batchUpdateErr = errFakeBatch
				f.provider.batchUpdateFailOn = 2
			},
			wantErr: errFakeBatch,
		},
		{
			name: "WriteCell error",
			setup: func(t *testing.T, f *spreadsheetFixture) {
				f.provider.updateErr = errFakeUpdate
			},
			wantErr: errFakeUpdate,
		},
		{
			name: "WriteCell error on delta cell",
			setup: func(t *testing.T, f *spreadsheetFixture) {
				f.provider.updateErr = errFakeUpdate
				f.provider.updateFailOn = 2
			},
			wantErr: errFakeUpdate,
		},
	}

	for _, tt := range errTests {
		t.Run(tt.name, func(t *testing.T) {
			f := newSpreadsheetFixture(t)
			tt.setup(t, f)
			err := f.svc.UpdateSpreadsheet(trans)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("UpdateSpreadsheet error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func Test_ApplyTransactions(t *testing.T) {
	trans := []*models.Transaction{{
		Source: "WellsFargo", Name: "Amazon", BankName: "Amazon",
		Date: "10/02/26", Withdrawal: 50.00, Amount: 50.00, Budget: -50.00,
	}}
	corrections := []*DateCorrection{{
		NewDate: "10/06/26",
		Entry:   &sheets_service.RegisterEntry{RowID: 20},
	}}
	balanceClient := func(t *testing.T) *banking.Client { return newBalanceTestClient(t, 1234.56) }
	noTokenClient := func(t *testing.T) *banking.Client { return newNoTokenBankClient() }

	tests := []struct {
		name        string
		dryRun      bool
		trans       []*models.Transaction
		corrections []*DateCorrection
		bankClient  func(t *testing.T) *banking.Client
		setup       func(f *spreadsheetFixture)
		wantErr     error
		wantErrText string
		check       func(t *testing.T, f *spreadsheetFixture)
	}{
		{
			name:        "dry run skips db and sheet writes",
			dryRun:      true,
			trans:       trans,
			corrections: corrections,
			bankClient:  balanceClient,
			check: func(t *testing.T, f *spreadsheetFixture) {
				if f.repo.updateCalls != 0 {
					t.Errorf("updateCalls = %d, want 0", f.repo.updateCalls)
				}
				if f.provider.batchUpdateCalls != 0 {
					t.Errorf("batchUpdateCalls = %d, want 0", f.provider.batchUpdateCalls)
				}
				if len(f.provider.updates) != 1 {
					t.Fatalf("updates = %d, want 1 (balance cell only)", len(f.provider.updates))
				}
				if f.provider.updates[0].range_ != "Register!G1:G1" {
					t.Errorf("updates[0].range = %q, want %q", f.provider.updates[0].range_, "Register!G1:G1")
				}
			},
		},
		{
			name:       "persist error",
			trans:      trans,
			bankClient: balanceClient,
			setup: func(f *spreadsheetFixture) {
				f.repo.updateErr = errFakeDB
			},
			wantErr: errFakeDB,
		},
		{
			name:        "date corrections applied",
			trans:       nil,
			corrections: corrections,
			bankClient:  balanceClient,
			check: func(t *testing.T, f *spreadsheetFixture) {
				if f.provider.batchUpdateCalls != 0 {
					t.Errorf("batchUpdateCalls = %d, want 0 (no spreadsheet update)", f.provider.batchUpdateCalls)
				}
				if len(f.provider.updates) != 2 {
					t.Fatalf("updates = %d, want 2 (date correction + balance cell)", len(f.provider.updates))
				}
				want := recordedUpdate{range_: "Register!C20", values: [][]any{{"10/06/26"}}}
				if !reflect.DeepEqual(f.provider.updates[0], want) {
					t.Errorf("updates[0] = %v, want %v", f.provider.updates[0], want)
				}
			},
		},
		{
			name:        "date corrections error",
			trans:       nil,
			corrections: corrections,
			bankClient:  balanceClient,
			setup: func(f *spreadsheetFixture) {
				f.provider.updateErr = errFakeUpdate
			},
			wantErr: errFakeUpdate,
		},
		{
			name:       "spreadsheet update",
			trans:      trans,
			bankClient: balanceClient,
			check: func(t *testing.T, f *spreadsheetFixture) {
				if f.provider.batchUpdateCalls != 2 {
					t.Errorf("batchUpdateCalls = %d, want 2 (CopyRows + UpdateRows)", f.provider.batchUpdateCalls)
				}
				if len(f.provider.updates) != 3 {
					t.Errorf("updates = %d, want 3 (LastUpdate + CheckingDelta + balance cells)", len(f.provider.updates))
				}
			},
		},
		{
			name:        "balance unavailable",
			dryRun:      true,
			bankClient:  noTokenClient,
			wantErrText: "balance unavailable",
		},
		{
			name:       "update balances error",
			trans:      nil,
			bankClient: balanceClient,
			setup: func(f *spreadsheetFixture) {
				f.provider.updateErr = errFakeUpdate
			},
			wantErr: errFakeUpdate,
		},
		{
			name:       "spreadsheet update error",
			trans:      trans,
			bankClient: balanceClient,
			setup: func(f *spreadsheetFixture) {
				f.provider.batchUpdateErr = errFakeBatch
			},
			wantErr: errFakeBatch,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newSpreadsheetFixture(t)
			f.svc.BankClient = tt.bankClient(t)
			f.svc.Options = &config.Options{DryRun: tt.dryRun}
			if tt.setup != nil {
				tt.setup(f)
			}
			err := f.svc.ApplyTransactions("Register", tt.trans, tt.corrections)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("ApplyTransactions error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if tt.wantErrText != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErrText) {
					t.Fatalf("ApplyTransactions error = %v, want it to contain %q", err, tt.wantErrText)
				}
				return
			}
			if err != nil {
				t.Fatalf("ApplyTransactions = %v, want nil", err)
			}
			if tt.check != nil {
				tt.check(t, f)
			}
		})
	}
}
