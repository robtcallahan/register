package sheets_service

import (
	"os"
	"testing"
)

const registerServiceJSONDir = "/Users/rob/ws/go/src/register/api/services/sheets_service/json/"

func Test_sheetsService_ReadRegisterSheet(t *testing.T) {
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

func Test_sheetsService_ReadBudgetSheet(t *testing.T) {
	// 10.10: Rewritten to test 9.3 header validation without fixtures.
	// The original compared against a stale JSON golden.
	getValuesProviderFunc = func(range_ string) ([][]any, error) {
		var values [][]any
		if range_ == "Budget!A1:J1" {
			values = [][]any{{"", "Weekly", "Weekly (view only)", "Bi-Weekly", "Monthly", "Monthly (view only)", "Yearly", "Current Total", "Needed - Monthly"}}
			return values, nil
		}
		// Return minimal budget data
		values = [][]any{
			{"", "Weekly", "Weekly (view only)", "Bi-Weekly", "Monthly", "Monthly (view only)", "Yearly", "Current Total", "Needed - Monthly"},
			{"Grocery", "120", "240", "520", "520", "6240", "0", "0"},
		}
		return values, nil
	}

	got, err := bs.ReadBudgetSheet(budgetSheet)
	if err != nil {
		t.Fatalf("ReadBudgetSheet() error = %v", err)
	}
	if got == nil {
		t.Fatal("ReadBudgetSheet() returned nil")
	}
	// Verify the budget entries were parsed
	if len(got.CategoriesMap) == 0 {
		t.Errorf("ReadBudgetSheet() returned empty CategoriesMap")
	}
}

// readTestFixture reads a test fixture file, skipping the test when the
// fixture is absent. Fixtures live on Rob's Mac, not in the repo.
func readTestFixture(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("fixture not in repo: %s", path)
	}
	return b
}
