package update_service

import (
	"reflect"
	"testing"

	"register/api/services/sheets_service"
	"register/pkg/banking"
	cfg "register/pkg/config"
	"register/pkg/models"
	"register/pkg/repository"
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
	fuzzyMerchants := map[string]bool{
		"Amazon":             true,
		"Amazon Marketplace": true,
	}
	regEntries := []*sheets_service.RegisterEntry{
		{
			Key:  "amazon:12/31/25:25.00",
			Name: "Amazon",
		}, {
			Key:  "wellsfargo:01/05/26:100.00",
			Name: "T-Mobile",
		},
	}
	trans := []*models.Transaction{
		{Key: "amazon:01/01/26:25.00"},      // +1 day shift: duplicate
		{Key: "amazon:12/30/25:25.00"},      // -1 day shift: duplicate
		{Key: "wellsfargo:01/05/26:100.00"}, // exact match: duplicate
		{Key: "wellsfargo:01/06/26:100.00"}, // ±1 shift, non-listed merchant: new
		{Key: "amazon:01/01/26:30.00"},      // different amount: new
		{Key: "chase:01/01/26:25.00"},       // different source: new
	}

	bankClient := &banking.Client{}
	updateService := NewUpdateService(nil, nil, bankClient, nil, nil, &cfg.Options{DryRun: false})
	fuzziedKeysMap := updateService.ExpandFuzzyKeys(regEntries, fuzzyMerchants)
	got := updateService.DedupeAgainstSheet(trans, fuzziedKeysMap)

	if len(got) != 3 {
		t.Fatalf("expected 3 new transactions, got %d", len(got))
	}
	if got[0].Key != "wellsfargo:01/06/26:100.00" || got[1].Key != "amazon:01/01/26:30.00" || got[2].Key != "chase:01/01/26:25.00" {
		t.Errorf("unexpected survivors: %s, %s, %s", got[0].Key, got[1].Key, got[2].Key)
	}
}

func Test_updateService_FuzzyKeySet(t *testing.T) {
	keys := map[string]bool{
		"amazon:12/31/25:25.00": true,
		"badkey":                true,
		"chase:01/01/26:10.00":  true,
	}
	us := &UpdateService{nil, nil, nil, nil, nil, nil, nil}
	fuzzy := us.FuzzyKeySet(keys)

	for _, want := range []string{
		"amazon:12/30/25:25.00",
		"amazon:12/31/25:25.00",
		"amazon:01/01/26:25.00", // year boundary
		"badkey",
		"chase:12/31/25:10.00",
		"chase:01/02/26:10.00",
	} {
		if !fuzzy[want] {
			t.Errorf("missing variant %q", want)
		}
	}
	if fuzzy["amazon:01/02/26:25.00"] {
		t.Error("did not expect a +2 day variant")
	}
}
