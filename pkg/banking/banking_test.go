package banking

import (
	"strings"
	"testing"

	"register/pkg/config"
	"register/pkg/models"

	"github.com/plaid/plaid-go/v15/plaid"
)

func TestNewClient(t *testing.T) {
	banks := map[string]config.Bank{
		"wellsfargo": {ID: "wellsfargo", Name: "Wells Fargo"},
	}
	sources := []config.IncomeSource{{Name: "Taylor House Paycheck"}}
	c := NewClient("user1", banks, sources, "client-id", "secret", plaid.Sandbox, nil)
	if c.UserID != "user1" || c.ClientID != "client-id" || c.Secret != "secret" {
		t.Errorf("identity = %q/%q/%q, want user1/client-id/secret", c.UserID, c.ClientID, c.Secret)
	}
	if c.Environment != plaid.Sandbox {
		t.Errorf("Environment = %v, want Sandbox", c.Environment)
	}
	if len(c.Banks) != 1 || c.Banks["wellsfargo"].Name != "Wells Fargo" {
		t.Errorf("Banks = %v, want wellsfargo entry", c.Banks)
	}
	if len(c.IncomeSources) != 1 || c.IncomeSources[0].Name != "Taylor House Paycheck" {
		t.Errorf("IncomeSources = %v, want one source", c.IncomeSources)
	}
	if c.PlaidClient == nil {
		t.Errorf("PlaidClient = nil, want non-nil client")
	}
	if c.Progress != nil {
		t.Errorf("Progress = non-nil, want nil as passed")
	}
}

func TestClient_GetBalances(t *testing.T) {
	c := NewClient("user1", map[string]config.Bank{
		"wellsfargo": {ID: "wellsfargo", Name: "Wells Fargo"},
	}, nil, "cid", "sec", plaid.Sandbox, nil)
	t.Run("no banks", func(t *testing.T) {
		got := c.GetBalances(nil)
		if len(got) != 0 {
			t.Errorf("GetBalances(nil) = %v, want empty map", got)
		}
	})
	t.Run("bank without token", func(t *testing.T) {
		got := c.GetBalances([]string{"wellsfargo"})
		bal, ok := got["wellsfargo"]
		if !ok {
			t.Fatalf("GetBalances() missing wellsfargo entry")
		}
		if bal.BankName != "Wells Fargo" {
			t.Errorf("BankName = %q, want Wells Fargo", bal.BankName)
		}
		if bal.Error == nil || !strings.Contains(bal.Error.Error(), "no Plaid access token") {
			t.Errorf("Error = %v, want missing-token error", bal.Error)
		}
	})
	t.Run("unknown bank id", func(t *testing.T) {
		got := c.GetBalances([]string{"nope"})
		bal, ok := got["nope"]
		if !ok {
			t.Fatalf("GetBalances() missing nope entry")
		}
		if bal.Error == nil {
			t.Errorf("Error = nil, want missing-token error for unknown bank")
		}
	})
}

func TestSortTransactions(t *testing.T) {
	mk := func(date, name string) *models.Transaction {
		return &models.Transaction{Date: date, Name: name}
	}
	tests := []struct {
		name string
		in   []*models.Transaction
		want []string
	}{
		{
			name: "by date",
			in:   []*models.Transaction{mk("12/08/20", "B"), mk("12/07/20", "A"), mk("12/09/20", "C")},
			want: []string{"A", "B", "C"},
		},
		{
			name: "same date by name",
			in:   []*models.Transaction{mk("12/07/20", "TARGET"), mk("12/07/20", "AMAZON")},
			want: []string{"AMAZON", "TARGET"},
		},
		{
			name: "already sorted",
			in:   []*models.Transaction{mk("12/07/20", "A"), mk("12/08/20", "B")},
			want: []string{"A", "B"},
		},
		{
			name: "empty",
			in:   []*models.Transaction{},
			want: []string{},
		},
		{
			name: "single",
			in:   []*models.Transaction{mk("12/07/20", "A")},
			want: []string{"A"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SortTransactions(tt.in)
			if len(got) != len(tt.want) {
				t.Fatalf("len = %d, want %d", len(got), len(tt.want))
			}
			for i, w := range tt.want {
				if got[i].Name != w {
					t.Errorf("position %d = %q, want %q", i, got[i].Name, w)
				}
			}
		})
	}
}
