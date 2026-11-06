package banking

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"register/pkg/config"

	"github.com/plaid/plaid-go/v15/plaid"
)

// newPlaidTestClient builds a banking Client whose Plaid API calls go to
// the given test server instead of Plaid. plaid-go honors the
// Configuration.Servers override (its OperationServers map is empty by
// default), so no production code changes are needed to redirect it.
func newPlaidTestClient(t *testing.T, handler http.HandlerFunc) (*Client, *httptest.Server) {
	t.Helper()
	ts := httptest.NewServer(handler)
	t.Cleanup(ts.Close)
	cfg := plaid.NewConfiguration()
	cfg.Servers = plaid.ServerConfigurations{{URL: ts.URL}}
	return &Client{
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

func TestClient_GetAccounts(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		c, _ := newPlaidTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/accounts/get" {
				t.Errorf("path = %q, want /accounts/get", r.URL.Path)
			}
			writeJSON(t, w, http.StatusOK, map[string]any{
				"accounts": []map[string]any{
					{"account_id": "acc1", "name": "Checking", "type": "depository", "subtype": "checking",
						"balances": map[string]any{"available": 100.0, "current": 200.0, "iso_currency_code": "USD"}},
					{"account_id": "acc2", "name": "Savings", "type": "depository", "subtype": "savings",
						"balances": map[string]any{"available": 300.0, "current": 400.0, "iso_currency_code": "USD"}},
				},
			})
		})
		got, err := c.GetAccounts(context.Background(), "token-wf")
		if err != nil {
			t.Fatalf("GetAccounts = %v, want nil", err)
		}
		if len(got) != 2 {
			t.Fatalf("got %d accounts, want 2", len(got))
		}
		if got[0].AccountId != "acc1" || got[0].Name != "Checking" {
			t.Errorf("account 0 = %+v, want acc1/Checking", got[0])
		}
		if got[1].AccountId != "acc2" {
			t.Errorf("account 1 id = %q, want acc2", got[1].AccountId)
		}
	})

	t.Run("plaid error", func(t *testing.T) {
		c, _ := newPlaidTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusBadRequest, map[string]any{
				"error_type": "INVALID_REQUEST", "error_code": "INVALID_ACCESS_TOKEN",
				"error_message": "bad token", "request_id": "req1",
			})
		})
		if _, err := c.GetAccounts(context.Background(), "bad-token"); err == nil {
			t.Error("GetAccounts(error response) = nil, want error")
		}
	})

	t.Run("unreachable server", func(t *testing.T) {
		c, ts := newPlaidTestClient(t, func(w http.ResponseWriter, r *http.Request) {})
		ts.Close() // t.Cleanup will close again; Close on a closed server is a no-op.
		if _, err := c.GetAccounts(context.Background(), "token-wf"); err == nil {
			t.Error("GetAccounts(closed server) = nil, want error")
		}
	})
}

func TestClient_GetBalance(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		c, _ := newPlaidTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/accounts/balance/get" {
				t.Errorf("path = %q, want /accounts/balance/get", r.URL.Path)
			}
			writeJSON(t, w, http.StatusOK, map[string]any{
				"accounts": []map[string]any{
					{"account_id": "acc1", "balances": map[string]any{"current": 1234.56, "iso_currency_code": "USD"}},
				},
			})
		})
		got, err := c.GetBalance(context.Background(), "token-wf", "wellsfargo")
		if err != nil {
			t.Fatalf("GetBalance = %v, want nil", err)
		}
		if got == nil || *got != 1234.56 {
			t.Errorf("GetBalance = %v, want 1234.56", got)
		}
	})

	t.Run("current not set", func(t *testing.T) {
		c, _ := newPlaidTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusOK, map[string]any{
				"accounts": []map[string]any{
					{"account_id": "acc1", "balances": map[string]any{"iso_currency_code": "USD"}},
				},
			})
		})
		if _, err := c.GetBalance(context.Background(), "token-wf", "wellsfargo"); err == nil {
			t.Error("GetBalance(unset current) = nil, want error")
		} else if !strings.Contains(err.Error(), "available balance is not set") {
			t.Errorf("GetBalance(unset current) = %v, want not-set error", err)
		}
	})

	t.Run("plaid error", func(t *testing.T) {
		c, _ := newPlaidTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusBadRequest, map[string]any{
				"error_type": "INVALID_REQUEST", "error_code": "NOPE",
				"error_message": "boom", "request_id": "req1",
			})
		})
		if _, err := c.GetBalance(context.Background(), "token-wf", "wellsfargo"); err == nil {
			t.Error("GetBalance(error response) = nil, want error")
		}
	})

	t.Run("unreachable server", func(t *testing.T) {
		c, ts := newPlaidTestClient(t, func(w http.ResponseWriter, r *http.Request) {})
		ts.Close() // t.Cleanup will close again; Close on a closed server is a no-op.
		if _, err := c.GetBalance(context.Background(), "token-wf", "wellsfargo"); err == nil {
			t.Error("GetBalance(closed server) = nil, want error")
		}
	})
}

func TestClient_GetBankStatus(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		c, _ := newPlaidTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/institutions/get_by_id" {
				t.Errorf("path = %q, want /institutions/get_by_id", r.URL.Path)
			}
			writeJSON(t, w, http.StatusOK, map[string]any{
				"institution": map[string]any{
					"institution_id": "ins_1", "name": "Test Bank",
					"products": []string{"transactions"}, "country_codes": []string{"US"},
				},
			})
		})
		got, err := c.GetBankStatus("wellsfargo")
		if err != nil {
			t.Fatalf("GetBankStatus = %v, want nil", err)
		}
		if got.InstitutionId != "ins_1" || got.Name != "Test Bank" {
			t.Errorf("institution = %+v, want ins_1/Test Bank", got)
		}
	})

	t.Run("plaid error", func(t *testing.T) {
		c, _ := newPlaidTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusBadRequest, map[string]any{
				"error_type": "INVALID_REQUEST", "error_code": "NOPE",
				"error_message": "boom", "request_id": "req1",
			})
		})
		if _, err := c.GetBankStatus("wellsfargo"); err == nil {
			t.Error("GetBankStatus(error response) = nil, want error")
		}
	})

	t.Run("unreachable server", func(t *testing.T) {
		c, ts := newPlaidTestClient(t, func(w http.ResponseWriter, r *http.Request) {})
		ts.Close() // t.Cleanup will close again; Close on a closed server is a no-op.
		if _, err := c.GetBankStatus("wellsfargo"); err == nil {
			t.Error("GetBankStatus(closed server) = nil, want error")
		}
	})
}

// GetBalances fans out to GetBalance per bank: one bank's success and
// another's Plaid error both land in the returned map.
func TestClient_GetBalances_WithFakeServer(t *testing.T) {
	c, _ := newPlaidTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			AccessToken string `json:"access_token"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("decode balance request: %v", err)
		}
		if req.AccessToken == "token-wf" {
			writeJSON(t, w, http.StatusOK, map[string]any{
				"accounts": []map[string]any{
					{"account_id": "acc1", "balances": map[string]any{"current": 1234.56, "iso_currency_code": "USD"}},
				},
			})
			return
		}
		writeJSON(t, w, http.StatusBadRequest, map[string]any{
			"error_type": "INVALID_REQUEST", "error_code": "NOPE",
			"error_message": "boom", "request_id": "req1",
		})
	})
	got := c.GetBalances([]string{"wellsfargo", "chase"})
	wf, ok := got["wellsfargo"]
	if !ok {
		t.Fatalf("GetBalances() missing wellsfargo entry")
	}
	if wf.Error != nil {
		t.Errorf("wellsfargo Error = %v, want nil", wf.Error)
	}
	if wf.Amount != 1234.56 {
		t.Errorf("wellsfargo Amount = %v, want 1234.56", wf.Amount)
	}
	ch, ok := got["chase"]
	if !ok {
		t.Fatalf("GetBalances() missing chase entry")
	}
	if ch.Error == nil {
		t.Error("chase Error = nil, want the Plaid error")
	} else if !strings.Contains(ch.Error.Error(), "chase") {
		t.Errorf("chase Error = %v, want it to name the bank", ch.Error)
	}
}

func plaidTx(id string, amount float64, pending bool, checkNumber string) map[string]any {
	tx := map[string]any{
		"transaction_id": id, "account_id": "acc1", "amount": amount,
		"date": "2026-10-28", "name": "TX " + id, "pending": pending,
	}
	if checkNumber != "" {
		tx["check_number"] = checkNumber
	}
	return tx
}

func TestClient_GetTransactions(t *testing.T) {
	t.Run("splits pending and posted", func(t *testing.T) {
		c, _ := newPlaidTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusOK, map[string]any{
				"accounts": []map[string]any{{"account_id": "acc1"}},
				"transactions": []map[string]any{
					plaidTx("tx-posted", -50.00, false, ""),
					plaidTx("tx-pending", -25.00, true, ""),
				},
				"total_transactions": 2,
			})
		})
		posted, pending, err := c.GetTransactions([]string{"wellsfargo"}, "2026-10-01", "2026-11-08")
		if err != nil {
			t.Fatalf("GetTransactions = %v, want nil", err)
		}
		if len(posted) != 1 || len(pending) != 1 {
			t.Fatalf("posted = %d, pending = %d; want 1, 1", len(posted), len(pending))
		}
		if posted[0].TransactionID != "tx-posted" || posted[0].Pending {
			t.Errorf("posted[0] = %+v, want tx-posted, not pending", posted[0])
		}
		if pending[0].TransactionID != "tx-pending" || !pending[0].Pending {
			t.Errorf("pending[0] = %+v, want tx-pending, pending", pending[0])
		}
		if posted[0].Deposit != 50.00 || posted[0].Key != "wellsfargo:10/28/26:50.00" {
			t.Errorf("posted[0] Deposit = %v, Key = %q; want 50.00 and wellsfargo:10/28/26:50.00", posted[0].Deposit, posted[0].Key)
		}
	})

	t.Run("check transaction", func(t *testing.T) {
		c, _ := newPlaidTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusOK, map[string]any{
				"transactions": []map[string]any{
					plaidTx("tx-check", -250.00, false, "1234"),
				},
				"total_transactions": 1,
			})
		})
		posted, _, err := c.GetTransactions([]string{"wellsfargo"}, "2026-10-01", "2026-11-08")
		if err != nil {
			t.Fatalf("GetTransactions = %v, want nil", err)
		}
		if len(posted) != 1 || !posted[0].IsCheck || posted[0].Source != "1234" {
			t.Errorf("posted = %+v, want one check with source 1234", posted)
		}
	})

	t.Run("unknown bank id", func(t *testing.T) {
		c, _ := newPlaidTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusOK, map[string]any{
				"transactions":       []map[string]any{plaidTx("tx1", -10.00, false, "")},
				"total_transactions": 1,
			})
		})
		c.Banks["nope"] = config.Bank{ID: "nope", Name: "Nope", AccessToken: "token-nope"}
		if _, _, err := c.GetTransactions([]string{"nope"}, "2026-10-01", "2026-11-08"); err == nil {
			t.Error("GetTransactions(unknown bank) = nil, want error")
		} else if !strings.Contains(err.Error(), "unknown bank ID: nope") {
			t.Errorf("GetTransactions(unknown bank) = %v, want unknown-bank error", err)
		}
	})

	t.Run("plaid error accumulates", func(t *testing.T) {
		c, _ := newPlaidTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusBadRequest, map[string]any{
				"error_type": "INVALID_REQUEST", "error_code": "NOPE",
				"error_message": "boom", "request_id": "req1",
			})
		})
		posted, pending, err := c.GetTransactions([]string{"wellsfargo", "chase"}, "2026-10-01", "2026-11-08")
		if err == nil {
			t.Error("GetTransactions(failing plaid) = nil, want error")
		}
		if posted != nil || pending != nil {
			t.Errorf("GetTransactions(failing plaid) = %v, %v; want nil, nil", posted, pending)
		}
	})
}

func Test_getPlaidTransactions_pagination(t *testing.T) {
	const total = 150
	c, _ := newPlaidTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Options struct {
				Count  int32 `json:"count"`
				Offset int32 `json:"offset"`
			} `json:"options"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("decode transactions request: %v", err)
		}
		start := int(req.Options.Offset)
		end := start + int(req.Options.Count)
		if end > total {
			end = total
		}
		var txs []map[string]any
		for i := start; i < end; i++ {
			txs = append(txs, plaidTx(fmt.Sprintf("tx-%03d", i), -10.00, false, ""))
		}
		writeJSON(t, w, http.StatusOK, map[string]any{
			"transactions": txs, "total_transactions": total,
		})
	})
	got, err := c.getPlaidTransactions(config.Bank{ID: "wellsfargo", AccessToken: "token-wf"}, "2026-10-01", "2026-11-08")
	if err != nil {
		t.Fatalf("getPlaidTransactions = %v, want nil", err)
	}
	if len(got) != total {
		t.Fatalf("got %d transactions, want %d", len(got), total)
	}
	if got[0].TransactionId != "tx-000" || got[total-1].TransactionId != "tx-149" {
		t.Errorf("first = %q, last = %q; want tx-000, tx-149", got[0].TransactionId, got[total-1].TransactionId)
	}
}

func Test_getPlaidTransactions_errors(t *testing.T) {
	t.Run("plaid error includes body", func(t *testing.T) {
		c, _ := newPlaidTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusBadRequest, map[string]any{
				"error_type": "INVALID_REQUEST", "error_code": "NOPE",
				"error_message": "boom", "request_id": "req1",
			})
		})
		if _, err := c.getPlaidTransactions(config.Bank{ID: "wellsfargo", AccessToken: "token-wf"}, "2026-10-01", "2026-11-08"); err == nil {
			t.Error("getPlaidTransactions(error response) = nil, want error")
		}
	})

	t.Run("unreachable server", func(t *testing.T) {
		c, ts := newPlaidTestClient(t, func(w http.ResponseWriter, r *http.Request) {})
		ts.Close() // t.Cleanup will close again; Close on a closed server is a no-op.
		if _, err := c.getPlaidTransactions(config.Bank{ID: "wellsfargo", AccessToken: "token-wf"}, "2026-10-01", "2026-11-08"); err == nil {
			t.Error("getPlaidTransactions(closed server) = nil, want error")
		}
	})
}
