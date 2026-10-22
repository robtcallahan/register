package update_service

import (
	stdcsv "encoding/csv"
	"fmt"
	"os"
	"register/api/services/sheets_service"
	cfg "register/pkg/config"
	"register/pkg/csv"
	"register/pkg/repository"
	"strconv"
	"strings"
	"time"

	"register/pkg/banking"
	"register/pkg/models"
)

type UpdateService struct {
	RegisterService *sheets_service.RegisterService
	BudgetService   *sheets_service.BudgetService
	BankClient      *banking.Client
	QueryRepo       repository.QueryRepo
	Config          *cfg.Config
	Options         *cfg.Options
}

func NewUpdateService(registerService *sheets_service.RegisterService, budgetService *sheets_service.BudgetService, bankClient *banking.Client, queryRepo repository.QueryRepo, config *cfg.Config, options *cfg.Options) *UpdateService {
	return &UpdateService{
		RegisterService: registerService,
		BudgetService:   budgetService,
		BankClient:      bankClient,
		QueryRepo:       queryRepo,
		Config:          config,
		Options:         options,
	}
}

// PersistTransactions interactively fills in any missing names and notes,
// then saves the transactions to the DB. All questions come first, so the rest
// of the run is unattended. Skipped entirely on --dry-run runs.
func (s *UpdateService) PersistTransactions(transactions []*models.Transaction) ([]*models.Transaction, error) {
	if s.Options.DryRun {
		return transactions, nil
	}

	fmt.Println("Updating transactions table...")
	if err := s.QueryRepo.UpdateTransactionTables(transactions); err != nil {
		return nil, err
	}
	return transactions, nil
}

// FetchTransactions gathers transactions from every bank: Fidelity always
// comes from its CSV (Plaid no longer supports it), while Wells Fargo and
// Chase come from CSV or Plaid depending on the --csv flag.
func (s *UpdateService) FetchTransactions(bankIDs []string, startDate, endDate string) ([]*models.Transaction, error) {
	fmt.Println("Getting Fidelity transactions (CSV)...")
	transactions, err := getCSVTransactions(s.Config, []string{"fidelity"})
	if err != nil {
		return nil, err
	}

	var newTrans []*models.Transaction
	if s.Options.UseCSVFiles {
		fmt.Println("Getting Wells Fargo & Chase transactions (CSV)...")
		newTrans, err = getCSVTransactions(s.Config, bankIDs)
	} else {
		fmt.Println("Getting Wells Fargo & Chase transactions (Plaid)...")
		newTrans, err = getTransactions(s.BankClient, bankIDs, startDate, endDate)
	}
	if err != nil {
		return nil, err
	}
	return append(transactions, newTrans...), nil
}

// WriteRawTransactionsCSV dumps the fetched transactions to transactions.csv
// in the finance directory. It runs before merchant normalization and sheet
// dedupe, so the file holds every transaction pulled from the banks,
// including ones already recorded in the register. encoding/csv quotes the
// fields, so bank names containing commas can't shift the columns.
func (s *UpdateService) WriteRawTransactionsCSV(transactions []*models.Transaction) error {
	fmt.Println("Writing raw transactions CSV file...")
	file, err := os.Create(s.Config.FinanceDir + "/transactions.csv")
	if err != nil {
		return fmt.Errorf("error creating file %s: %w", s.Config.FinanceDir+"/transactions.csv", err)
	}
	defer file.Close()

	w := stdcsv.NewWriter(file)
	for _, t := range transactions {
		record := []string{
			t.Source, t.Date, t.BankName,
			strconv.FormatFloat(t.Amount, 'f', 2, 64),
			strconv.FormatFloat(t.Deposit, 'f', 2, 64),
			strconv.FormatFloat(t.Withdrawal, 'f', 2, 64),
			strconv.FormatFloat(t.CreditCard, 'f', 2, 64),
		}
		if err := w.Write(record); err != nil {
			return fmt.Errorf("error writing to file: %w", err)
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return fmt.Errorf("error writing to file: %w", err)
	}
	return nil
}

// NormalizeMerchants assigns each transaction its display name, color, and
// budget column from the merchant lookup data.
func (s *UpdateService) NormalizeMerchants(transactions []*models.Transaction) ([]*models.Transaction, error) {
	fmt.Println("Updating merchants...")
	lookupData, err := s.QueryRepo.GetLookupData()
	if err != nil {
		return nil, err
	}

	matcher, err := banking.NewMerchantMatcher(lookupData)
	if err != nil {
		return nil, err
	}

	income, err := s.BankClient.IncomeMatcher()
	if err != nil {
		return nil, err
	}

	columns, err := s.QueryRepo.GetColumns()
	if err != nil {
		return nil, err
	}

	transactions = s.BankClient.FormatMerchantNames(transactions, matcher, income, columns)
	return transactions, nil
}

// DedupeAgainstSheet drops transactions already recorded in the register
// sheet. Matching tolerates a ±1-day date shift, because some sources (notably
// Amazon) report the same transaction on a different day from one pull to the
// next, which changes its dedupe key.
func (s *UpdateService) DedupeAgainstSheet(transactions []*models.Transaction, keysMap map[string]bool) []*models.Transaction {
	fmt.Println("Filtering out register transactions...")
	return s.BankClient.FilterRecordedTransactions(transactions, fuzzyKeySet(keysMap))
}

// UpdateSpreadsheet writes the transactions into the register sheet and
// refreshes the bank balances. Skipped when there is nothing to write or on
// --dry-run runs.
func (s *UpdateService) UpdateSpreadsheet(transactions []*models.Transaction) error {
	if len(transactions) == 0 || s.Options.DryRun {
		fmt.Println("No updates needed")
		return nil
	}

	registerSheet, err := s.RegisterService.NewRegisterSheet(s.Config, nil)
	if err != nil {
		return err
	}
	budgetSheet, err := s.BudgetService.NewBudgetSheet(s.Config)
	if err != nil {
		return err
	}

	fmt.Printf("Reading Budget...\n")
	if _, err := s.BudgetService.ReadBudgetSheet(budgetSheet); err != nil {
		return err
	}

	fmt.Printf("    (%3s) %-12s %-10s %8s %-30s %s\n", "Num", "Source", "Date", "Amount", "Name", "Note")
	fmt.Printf("    (%3s) %-12s %-10s %8s %-30s %s\n", dashes(3), dashes(12), dashes(10), dashes(8), dashes(30), dashes(15))
	for i, r := range transactions {
		fmt.Printf("    (%3d) %-12s %-10s %8.2f %-30s %s\n", i+1, r.Source, r.Date, -1*r.Amount, r.Name, r.Note)
	}

	// add the needed number of rows for transactions
	fmt.Println("Adding rows...")
	if err := s.RegisterService.CopyRows(*registerSheet, len(transactions)); err != nil {
		return err
	}

	fmt.Printf("Transaction updates...\n")

	fmt.Printf("Updating spreadsheet...\n")
	columns, err := s.QueryRepo.GetColumns()
	if err != nil {
		return err
	}
	transNameToColName, err := s.QueryRepo.GetNameMapToColumn()
	if err != nil {
		return err
	}

	if err := s.RegisterService.UpdateRows(*registerSheet, *budgetSheet, columns, transNameToColName, transactions); err != nil {
		return err
	}

	lastRowUpdated := registerSheet.SheetCoords.FirstRowToUpdate + int64(len(transactions)*2) + 1
	if _, err := s.RegisterService.WriteCell(*registerSheet, sheets_service.LastUpdateCell, time.Now().Format("01/02/2006")); err != nil {
		return err
	}
	if _, err := s.RegisterService.WriteCell(*registerSheet, sheets_service.CheckingDeltaCell, fmt.Sprintf("=SUM(%s-I%d)", sheets_service.CheckingBalanceCell, lastRowUpdated)); err != nil {
		return err
	}
	return nil
}

// getTransactions connects to the Plaid interface to get the bank's transactions
func getTransactions(bankClient *banking.Client, bankIDs []string, startDate, endDate string) ([]*models.Transaction, error) {
	transactions, err := bankClient.GetTransactions(bankIDs, startDate, endDate)
	if err != nil {
		return nil, err
	}
	return transactions, nil
}

// getCSVTransactions reads local CSV files for banks that Plaid cannot talk with
// or if all CSV files are to be read using the proper flag, currently -c
func getCSVTransactions(config *cfg.Config, bankIDs []string) ([]*models.Transaction, error) {
	csvClient := csv.New(csv.ConfigOptions{
		FinanceDir: config.FinanceDir,
		Banks:      config.Banks,
		BankIDs:    bankIDs,
	})

	transactions, err := csvClient.GetTransactions()
	if err != nil {
		return nil, err
	}
	return transactions, nil
}

// fuzzyKeySet returns a copy of keys with each key's date also shifted by -1
// and +1 days. Keys that don't parse as source:date:amount are kept as-is.
func fuzzyKeySet(keys map[string]bool) map[string]bool {
	fuzzy := make(map[string]bool, len(keys)*3)
	for k := range keys {
		parts := strings.SplitN(k, ":", 3)
		if len(parts) != 3 {
			fuzzy[k] = true
			continue
		}
		d, err := time.Parse("01/02/06", parts[1])
		if err != nil {
			fuzzy[k] = true
			continue
		}
		amt, err := strconv.ParseFloat(parts[2], 64)
		if err != nil {
			fuzzy[k] = true
			continue
		}
		for _, shift := range []int{-1, 0, 1} {
			date := d.AddDate(0, 0, shift).Format("01/02/06")
			fuzzy[models.TransactionKey(parts[0], date, amt)] = true
		}
	}
	return fuzzy
}

// dashes is a simple helper to print out "count" number of dashes to be printed under a column header
func dashes(count int) string {
	return strings.Repeat("-", count)
}

// UpdateBalances takes the Wells Fargo checking account balance and updates the CheckingBalanceCell in the Register spreadsheet
// as it is used to determine if reconciliation has occurred.
func (s *UpdateService) UpdateBalances(registerSheet sheets_service.RegisterSheet, balances map[string]banking.Balance) error {
	if balances[banking.WellsFargoID].Error == nil {
		if _, err := s.RegisterService.WriteCell(registerSheet, sheets_service.CheckingBalanceCell, balances[banking.WellsFargoID].Amount); err != nil {
			return err
		}
	}
	return nil
}
