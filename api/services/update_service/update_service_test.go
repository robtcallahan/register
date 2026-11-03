package update_service

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"register/api/providers/sheets_provider"
	"strings"
	"testing"
	"unsafe"

	"register/api/services/sheets_service"
	"register/pkg/banking"
	cfg "register/pkg/config"
	"register/pkg/models"
	"register/pkg/repository"

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
		s := NewUpdateService(nil, nil, nil, mock, nil, &cfg.Options{DryRun: tt.dryRun})
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
	updateService := NewUpdateService(nil, nil, bankClient, nil, nil, &cfg.Options{DryRun: false})
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
		Config: &cfg.Config{
			FinanceDir: tmpDir,
			Banks: map[string]cfg.Bank{
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
		Options: &cfg.Options{UseCSVFiles: true},
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
		Config: &cfg.Config{
			FinanceDir: tmpDir,
			Banks: map[string]cfg.Bank{
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
		Options: &cfg.Options{UseCSVFiles: true},
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
		Config: &cfg.Config{
			FinanceDir: tmpDir,
			Banks: map[string]cfg.Bank{
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
		Options: &cfg.Options{UseCSVFiles: true},
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
	gotLine, _, err := contents.ReadLine()
	if err != nil {
		t.Fatalf("failed to read %s file: %v", transFile, err)
	}
	want := []byte(`Fidelity,01/02/26,SQ *PLUME HOUSE VAPE C Winchester    VA,-26.32,0.00,0.00,26.32`)
	if !bytes.Equal(want, gotLine) {
		t.Fatalf("got wrong %s value. want: %v, got: %v",
			transFile, unsafe.String(&want[0], len(want)), unsafe.String(&gotLine[0], len(gotLine)))
	}
}
