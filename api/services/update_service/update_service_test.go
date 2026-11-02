package update_service

import (
	"errors"
	"fmt"
	"reflect"
	"register/api/providers/sheets_provider"
	"testing"

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
}

func (p *fakeSheetsProvider) GetValues(range_ string) ([][]any, error) {
	return nil, p.getValuesErr
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

func Test_LoadTransactions(t *testing.T) {
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
