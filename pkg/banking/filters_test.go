package banking

import (
	"testing"

	"register/pkg/models"
)

func testTransaction(key, bankName string) *models.Transaction {
	return &models.Transaction{
		Key:      key,
		BankName: bankName,
		Source:   "WellsFargo",
		Date:     "12/07/20",
		Name:     bankName,
		Amount:   10.00,
	}
}

func TestFilterRecordedTransactions(t *testing.T) {
	c := &Client{}

	tests := []struct {
		name      string
		trans     []*models.Transaction
		regLookup map[string]bool
		wantKeys  []string
	}{
		{
			name: "keeps unrecorded, drops recorded",
			trans: []*models.Transaction{
				testTransaction("wellsfargo:12/07/20:10.00", "TARGET"),
				testTransaction("wellsfargo:12/07/20:20.00", "AMAZON"),
				testTransaction("wellsfargo:12/07/20:30.00", "COSTCO"),
			},
			regLookup: map[string]bool{
				"wellsfargo:12/07/20:20.00": true,
			},
			wantKeys: []string{
				"wellsfargo:12/07/20:10.00",
				"wellsfargo:12/07/20:30.00",
			},
		},
		{
			name: "empty lookup keeps all",
			trans: []*models.Transaction{
				testTransaction("wellsfargo:12/07/20:10.00", "TARGET"),
				testTransaction("wellsfargo:12/07/20:20.00", "AMAZON"),
			},
			regLookup: map[string]bool{},
			wantKeys: []string{
				"wellsfargo:12/07/20:10.00",
				"wellsfargo:12/07/20:20.00",
			},
		},
		{
			name: "all recorded returns empty",
			trans: []*models.Transaction{
				testTransaction("wellsfargo:12/07/20:10.00", "TARGET"),
			},
			regLookup: map[string]bool{
				"wellsfargo:12/07/20:10.00": true,
			},
			wantKeys: []string{},
		},
		{
			name:      "empty transactions returns empty",
			trans:     []*models.Transaction{},
			regLookup: map[string]bool{"wellsfargo:12/07/20:10.00": true},
			wantKeys:  []string{},
		},
		{
			name:      "nil transactions returns empty",
			trans:     nil,
			regLookup: map[string]bool{},
			wantKeys:  []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := c.FilterRecordedTransactions(tt.trans, tt.regLookup)

			if len(got) != len(tt.wantKeys) {
				t.Fatalf("got %d transactions, want %d", len(got), len(tt.wantKeys))
			}
			for i, wantKey := range tt.wantKeys {
				if got[i].Key != wantKey {
					t.Errorf("transaction %d Key = %q, want %q", i, got[i].Key, wantKey)
				}
			}
		})
	}
}

func TestFilterCreditCardPayments(t *testing.T) {
	c := &Client{}

	tests := []struct {
		name     string
		bankName string
		wantDrop bool
	}{
		{
			name:     "drops Wells Fargo card payment",
			bankName: "PAYMENT MADE BY ACCOUNT ENDING IN:5409",
			wantDrop: true,
		},
		{
			name:     "drops Chase payment with dash",
			bankName: "Payment Thank You - Bill",
			wantDrop: true,
		},
		{
			name:     "drops Chase payment without dash",
			bankName: "Payment Thank You Bill",
			wantDrop: true,
		},
		{
			name:     "keeps regular purchase",
			bankName: "TARGET",
			wantDrop: false,
		},
		{
			name:     "keeps similar but non-matching",
			bankName: "Payment Thank You",
			wantDrop: false,
		},
		{
			name:     "keeps empty bank name",
			bankName: "",
			wantDrop: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trans := []*models.Transaction{
				testTransaction("wellsfargo:12/07/20:10.00", tt.bankName),
			}
			got := c.FilterCreditCardPayments(trans)

			if tt.wantDrop && len(got) != 0 {
				t.Errorf("expected transaction to be dropped, but got %d", len(got))
			}
			if !tt.wantDrop && len(got) != 1 {
				t.Errorf("expected transaction to be kept, but got %d", len(got))
			}
		})
	}
}

func TestFilterCreditCardPayments_Mixed(t *testing.T) {
	c := &Client{}

	trans := []*models.Transaction{
		testTransaction("wellsfargo:12/07/20:10.00", "TARGET"),
		testTransaction("wellsfargo:12/07/20:500.00", "PAYMENT MADE BY ACCOUNT ENDING IN:5409"),
		testTransaction("wellsfargo:12/07/20:20.00", "AMAZON"),
		testTransaction("chase:12/07/20:300.00", "Payment Thank You - Bill"),
		testTransaction("wellsfargo:12/07/20:30.00", "COSTCO"),
	}

	got := c.FilterCreditCardPayments(trans)

	if len(got) != 3 {
		t.Fatalf("got %d transactions, want 3", len(got))
	}

	wantKeys := []string{
		"wellsfargo:12/07/20:10.00",
		"wellsfargo:12/07/20:20.00",
		"wellsfargo:12/07/20:30.00",
	}
	for i, wantKey := range wantKeys {
		if got[i].Key != wantKey {
			t.Errorf("transaction %d Key = %q, want %q", i, got[i].Key, wantKey)
		}
	}
}
