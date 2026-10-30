package csv

import (
	"testing"

	"register/pkg/models"
)

func Test_processWellsFargoData(t *testing.T) {
	tests := []struct {
		name     string
		rows     []*WellsFargo
		bankID   string
		checkIdx int
		checkTx  *models.Transaction
	}{
		{
			name: "withdrawal (negative amount)",
			rows: []*WellsFargo{
				{Date: "12/07/2020", Description: "TARGET", Amount: -46.80, CheckNumber: "", Status: "posted"},
			},
			bankID:   models.WellsFargoID,
			checkIdx: 0,
			checkTx: &models.Transaction{
				Source:     "WellsFargo",
				Date:       "12/07/20",
				Amount:     -46.80,
				BankName:   "TARGET",
				Name:       "TARGET",
				Budget:     -46.80,
				Withdrawal: 46.80,
				Key:        "wellsfargo:12/07/20:-46.80",
			},
		},
		{
			name: "deposit (positive amount)",
			rows: []*WellsFargo{
				{Date: "12/01/2020", Description: "DIRECT DEPOSIT", Amount: 1500.00, CheckNumber: "", Status: "posted"},
			},
			bankID:   models.WellsFargoID,
			checkIdx: 0,
			checkTx: &models.Transaction{
				Source:   "WellsFargo",
				Date:     "12/01/20",
				Amount:   1500.00,
				BankName: "DIRECT DEPOSIT",
				Name:     "DIRECT DEPOSIT",
				Budget:   1500.00,
				Deposit:  1500.00,
				Key:      "wellsfargo:12/01/20:1500.00",
			},
		},
		{
			name: "check transaction",
			rows: []*WellsFargo{
				{Date: "12/15/2020", Description: "CHECK # 1234", Amount: -250.00, CheckNumber: "CHECK # 1234", Status: "posted"},
			},
			bankID:   models.WellsFargoID,
			checkIdx: 0,
			checkTx: &models.Transaction{
				Source:     "1234",
				Date:       "12/15/20",
				Amount:     -250.00,
				Name:       models.CheckRegisterName,
				BankName:   models.CheckTransactionName,
				IsCheck:    true,
				Budget:     -250.00,
				Withdrawal: 250.00,
				Key:        "wellsfargo:12/15/20:-250.00",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := processWellsFargoData(tt.rows, tt.bankID)
			if len(got) != 1 {
				t.Fatalf("got %d transactions, want 1", len(got))
			}
			tx := got[tt.checkIdx]
			want := tt.checkTx

			if tx.Source != want.Source {
				t.Errorf("Source = %q, want %q", tx.Source, want.Source)
			}
			if tx.Date != want.Date {
				t.Errorf("Date = %q, want %q", tx.Date, want.Date)
			}
			if tx.Amount != want.Amount {
				t.Errorf("Amount = %v, want %v", tx.Amount, want.Amount)
			}
			if tx.BankName != want.BankName {
				t.Errorf("BankName = %q, want %q", tx.BankName, want.BankName)
			}
			if tx.Name != want.Name {
				t.Errorf("Name = %q, want %q", tx.Name, want.Name)
			}
			if tx.IsCheck != want.IsCheck {
				t.Errorf("IsCheck = %v, want %v", tx.IsCheck, want.IsCheck)
			}
			if tx.Budget != want.Budget {
				t.Errorf("Budget = %v, want %v", tx.Budget, want.Budget)
			}
			if tx.Withdrawal != want.Withdrawal {
				t.Errorf("Withdrawal = %v, want %v", tx.Withdrawal, want.Withdrawal)
			}
			if tx.Deposit != want.Deposit {
				t.Errorf("Deposit = %v, want %v", tx.Deposit, want.Deposit)
			}
			if tx.Key != want.Key {
				t.Errorf("Key = %q, want %q", tx.Key, want.Key)
			}
		})
	}
}

func Test_processFidelityData(t *testing.T) {
	tests := []struct {
		name     string
		rows     []*FidelityVisa
		bankID   string
		checkIdx int
		checkTx  *models.Transaction
	}{
		{
			// "Date","Transaction","Name","Memo","Amount"
			// "2026-09-28","DEBIT","Netflix.com            netflix.com   CA","24906416269268023334511; 04899; ; 5409ROBERTCALLAHAN; ; ;","-19.99"
			name: "charge (positive amount)",
			rows: []*FidelityVisa{
				{Date: "2026-09-28", Transaction: "DEBIT", Name: "Netflix.com            netflix.com   CA", Memo: "24906416269268023334511; 04899; ; 5409ROBERTCALLAHAN; ; ;", Amount: -19.99},
			},
			bankID:   models.FidelityID,
			checkIdx: 0,
			checkTx: &models.Transaction{
				Source:         "Fidelity",
				Date:           "09/28/26",
				Amount:         -19.99,
				Name:           "Netflix.com            netflix.com   CA",
				BankName:       "Netflix.com            netflix.com   CA",
				CreditPurchase: 19.99,
				Key:            "fidelity:09/28/26:19.99",
			},
		},
		{
			// "Date","Transaction","Name","Memo","Amount"
			// "2026-10-20","CREDIT","PROV FR CR CFK8M96SC4          # 7515","SDSPRBPD6292214351; 05999; ; 5409ROBERTCALLAHAN; ; ;","26.01"
			name: "credit (negative amount)",
			rows: []*FidelityVisa{
				{Date: "2026-10-20", Transaction: "CREDIT", Name: "PROV FR CR CFK8M96SC4          # 7515", Memo: "SDSPRBPD6292214351; 05999; ; 5409ROBERTCALLAHAN; ; ;", Amount: 26.01},
			},
			bankID:   models.FidelityID,
			checkIdx: 0,
			checkTx: &models.Transaction{
				Source:         "Fidelity",
				Date:           "10/20/26",
				Amount:         26.01,
				Name:           "PROV FR CR CFK8M96SC4          # 7515",
				BankName:       "PROV FR CR CFK8M96SC4          # 7515",
				CreditPurchase: -26.01,
				Key:            "fidelity:10/20/26:-26.01",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := processFidelityData(tt.rows, tt.bankID)
			if len(got) != 1 {
				t.Fatalf("got %d transactions, want 1", len(got))
			}
			tx := got[tt.checkIdx]
			want := tt.checkTx

			if tx.Source != want.Source {
				t.Errorf("Source = %q, want %q", tx.Source, want.Source)
			}
			if tx.Date != want.Date {
				t.Errorf("Date = %q, want %q", tx.Date, want.Date)
			}
			if tx.Amount != want.Amount {
				t.Errorf("Amount = %v, want %v", tx.Amount, want.Amount)
			}
			if tx.Name != want.Name {
				t.Errorf("Name = %q, want %q", tx.Name, want.Name)
			}
			if tx.BankName != want.BankName {
				t.Errorf("BankName = %q, want %q", tx.BankName, want.BankName)
			}
			if tx.CreditPurchase != want.CreditPurchase {
				t.Errorf("CreditPurchase = %v, want %v", tx.Deposit, want.Deposit)
			}
			if tx.Key != want.Key {
				t.Errorf("Key = %q, want %q", tx.Key, want.Key)
			}
		})
	}
}

func Test_processCheck(t *testing.T) {
	tests := []struct {
		name         string
		checkNumber  string
		wantSource   string
		wantIsCheck  bool
		wantName     string
		wantBankName string
	}{
		{
			name:         "four digit check",
			checkNumber:  "CHECK # 1234",
			wantSource:   "1234",
			wantIsCheck:  true,
			wantName:     models.CheckRegisterName,
			wantBankName: models.CheckTransactionName,
		},
		{
			name:         "three digit check",
			checkNumber:  "CHECK # 123",
			wantSource:   "123",
			wantIsCheck:  true,
			wantName:     models.CheckRegisterName,
			wantBankName: models.CheckTransactionName,
		},
		{
			name:         "no check number",
			checkNumber:  "",
			wantSource:   "WellsFargo",
			wantIsCheck:  false,
			wantName:     "",
			wantBankName: "TARGET",
		},
		{
			name:         "non-check description",
			checkNumber:  "TARGET",
			wantSource:   "WellsFargo",
			wantIsCheck:  false,
			wantName:     "",
			wantBankName: "TARGET",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tx := &models.Transaction{
				Source:   "WellsFargo",
				BankName: "TARGET",
			}
			got := processCheck(tt.checkNumber, tx)

			if got.Source != tt.wantSource {
				t.Errorf("Source = %q, want %q", got.Source, tt.wantSource)
			}
			if got.IsCheck != tt.wantIsCheck {
				t.Errorf("IsCheck = %v, want %v", got.IsCheck, tt.wantIsCheck)
			}
			if got.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", got.Name, tt.wantName)
			}
			if got.BankName != tt.wantBankName {
				t.Errorf("BankName = %q, want %q", got.BankName, tt.wantBankName)
			}
		})
	}
}

func Test_processWellsFargoData_Multiple(t *testing.T) {
	rows := []*WellsFargo{
		{Date: "12/07/2020", Description: "TARGET", Amount: -46.80},
		{Date: "12/08/2020", Description: "AMAZON", Amount: -29.66},
		{Date: "12/01/2020", Description: "PAYCHECK", Amount: 2000.00},
	}

	got := processWellsFargoData(rows, models.WellsFargoID)

	if len(got) != 3 {
		t.Fatalf("got %d transactions, want 3", len(got))
	}

	// Verify keys are unique
	keys := make(map[string]bool)
	for _, tx := range got {
		if keys[tx.Key] {
			t.Errorf("duplicate key %q", tx.Key)
		}
		keys[tx.Key] = true
	}
}
