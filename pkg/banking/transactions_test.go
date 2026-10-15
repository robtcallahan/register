package banking

import (
	"testing"

	"register/pkg/models"

	"github.com/plaid/plaid-go/v15/plaid"
)

func plaidTransaction(name string, amount float64, date string) plaid.Transaction {
	return plaid.Transaction{
		Name:   name,
		Amount: amount,
		Date:   date,
	}
}

func plaidCheckTransaction(checkNumber, date string, amount float64) plaid.Transaction {
	t := plaidTransaction("CHECK", amount, date)
	t.CheckNumber.Set(&checkNumber)
	return t
}

func TestBuildTransaction_WellsFargo(t *testing.T) {
	c := &Client{}

	tests := []struct {
		name      string
		plaidTx   plaid.Transaction
		checkTran *models.Transaction
	}{
		{
			name:    "withdrawal",
			plaidTx: plaidTransaction("TARGET", 46.80, "2020-12-07"),
			checkTran: &models.Transaction{
				Source:     "WellsFargo",
				Date:       "12/07/20",
				Name:       "TARGET",
				BankName:   "TARGET",
				Amount:     46.80,
				Withdrawal: 46.80,
				Budget:     -46.80,
				Key:        "wellsfargo:12/07/20:46.80",
			},
		},
		{
			name:    "deposit (negative plaid amount)",
			plaidTx: plaidTransaction("DIRECT DEPOSIT", -1500.00, "2020-12-01"),
			checkTran: &models.Transaction{
				Source:   "WellsFargo",
				Date:     "12/01/20",
				Name:     "DIRECT DEPOSIT",
				BankName: "DIRECT DEPOSIT",
				Amount:   -1500.00,
				Deposit:  1500.00,
				Budget:   1500.00,
				Key:      "wellsfargo:12/01/20:1500.00",
			},
		},
		{
			name:    "check transaction",
			plaidTx: plaidCheckTransaction("1234", "2020-12-15", 250.00),
			checkTran: &models.Transaction{
				Source:     "1234",
				Date:       "12/15/20",
				Name:       "CHECK",
				BankName:   "CHECK",
				IsCheck:    true,
				Amount:     250.00,
				Withdrawal: 250.00,
				Budget:     -250.00,
				Key:        "1234:12/15/20:250.00",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := c.buildTransaction(WellsFargoID, tt.plaidTx)
			if err != nil {
				t.Fatalf("buildTransaction failed: %v", err)
			}
			want := tt.checkTran

			if got.Source != want.Source {
				t.Errorf("Source = %q, want %q", got.Source, want.Source)
			}
			if got.Date != want.Date {
				t.Errorf("Date = %q, want %q", got.Date, want.Date)
			}
			if got.Name != want.Name {
				t.Errorf("Name = %q, want %q", got.Name, want.Name)
			}
			if got.BankName != want.BankName {
				t.Errorf("BankName = %q, want %q", got.BankName, want.BankName)
			}
			if got.IsCheck != want.IsCheck {
				t.Errorf("IsCheck = %v, want %v", got.IsCheck, want.IsCheck)
			}
			if got.Amount != want.Amount {
				t.Errorf("Amount = %v, want %v", got.Amount, want.Amount)
			}
			if got.Withdrawal != want.Withdrawal {
				t.Errorf("Withdrawal = %v, want %v", got.Withdrawal, want.Withdrawal)
			}
			if got.Deposit != want.Deposit {
				t.Errorf("Deposit = %v, want %v", got.Deposit, want.Deposit)
			}
			if got.Budget != want.Budget {
				t.Errorf("Budget = %v, want %v", got.Budget, want.Budget)
			}
			if got.Key != want.Key {
				t.Errorf("Key = %q, want %q", got.Key, want.Key)
			}
		})
	}
}

func TestBuildTransaction_Fidelity(t *testing.T) {
	c := &Client{}

	// Fidelity transactions are credit purchases (positive amounts stay positive)
	plaidTx := plaidTransaction("AMAZON", 29.66, "2020-12-10")
	got, err := c.buildTransaction(FidelityID, plaidTx)
	if err != nil {
		t.Fatalf("buildTransaction failed: %v", err)
	}

	if got.Source != "Fidelity" {
		t.Errorf("Source = %q, want %q", got.Source, "Fidelity")
	}
	if got.Date != "12/10/20" {
		t.Errorf("Date = %q, want %q", got.Date, "12/10/20")
	}
	if got.Name != "AMAZON" {
		t.Errorf("Name = %q, want %q", got.Name, "AMAZON")
	}
	if got.Amount != 29.66 {
		t.Errorf("Amount = %v, want 29.66", got.Amount)
	}
	if got.CreditPurchase != 29.66 {
		t.Errorf("CreditPurchase = %v, want 29.66", got.CreditPurchase)
	}
	if got.CreditCard != 29.66 {
		t.Errorf("CreditCard = %v, want 29.66", got.CreditCard)
	}
	if got.Budget != -29.66 {
		t.Errorf("Budget = %v, want -29.66", got.Budget)
	}
	wantKey := "fidelity:12/10/20:29.66"
	if got.Key != wantKey {
		t.Errorf("Key = %q, want %q", got.Key, wantKey)
	}
	// Fidelity should NOT set Withdrawal or Deposit
	if got.Withdrawal != 0 {
		t.Errorf("Withdrawal = %v, want 0", got.Withdrawal)
	}
	if got.Deposit != 0 {
		t.Errorf("Deposit = %v, want 0", got.Deposit)
	}
}

func TestBuildTransaction_Chase(t *testing.T) {
	c := &Client{}

	// Chase transactions are credit purchases (same as Fidelity)
	plaidTx := plaidTransaction("WHOLE FOODS", 85.42, "2020-12-12")
	got, err := c.buildTransaction(ChaseID, plaidTx)
	if err != nil {
		t.Fatalf("buildTransaction failed: %v", err)
	}

	if got.Source != "Chase" {
		t.Errorf("Source = %q, want %q", got.Source, "Chase")
	}
	if got.Date != "12/12/20" {
		t.Errorf("Date = %q, want %q", got.Date, "12/12/20")
	}
	if got.Name != "WHOLE FOODS" {
		t.Errorf("Name = %q, want %q", got.Name, "WHOLE FOODS")
	}
	if got.CreditPurchase != 85.42 {
		t.Errorf("CreditPurchase = %v, want 85.42", got.CreditPurchase)
	}
	if got.CreditCard != 85.42 {
		t.Errorf("CreditCard = %v, want 85.42", got.CreditCard)
	}
	if got.Budget != -85.42 {
		t.Errorf("Budget = %v, want -85.42", got.Budget)
	}
	wantKey := "chase:12/12/20:85.42"
	if got.Key != wantKey {
		t.Errorf("Key = %q, want %q", got.Key, wantKey)
	}
}

func TestBuildTransaction_UnknownBank(t *testing.T) {
	c := &Client{}
	plaidTx := plaidTransaction("SOME STORE", 10.00, "2020-12-07")
	_, err := c.buildTransaction("unknown", plaidTx)
	if err == nil {
		t.Error("expected error for unknown bank, got nil")
	}
}
