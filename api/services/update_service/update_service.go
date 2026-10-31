package update_service

import (
	stdcsv "encoding/csv"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"register/api/services/sheets_service"
	"register/pkg/banking"
	cfg "register/pkg/config"
	"register/pkg/csv"
	"register/pkg/models"
	"register/pkg/repository"

	"google.golang.org/api/sheets/v4"
)

type UpdateService struct {
	RegisterService *sheets_service.RegisterService
	BudgetService   *sheets_service.BudgetService
	BankClient      *banking.Client
	QueryRepo       repository.QueryRepo
	Progress        func(_ string)
	Config          *cfg.Config
	Options         *cfg.Options
}

func NewUpdateService(registerService *sheets_service.RegisterService, budgetService *sheets_service.BudgetService, bankClient *banking.Client, queryRepo repository.QueryRepo, config *cfg.Config, options *cfg.Options) *UpdateService {
	return &UpdateService{
		RegisterService: registerService,
		BudgetService:   budgetService,
		BankClient:      bankClient,
		QueryRepo:       queryRepo,
		Progress:        func(_ string) {},
		Config:          config,
		Options:         options,
	}
}

type DateCorrection struct {
	NewDate string
	Entry   *sheets_service.RegisterEntry
}

// LoadTransactions reads the Register spreadsheet and the bank
// transactions. It returns the new transactions.
func (s *UpdateService) LoadTransactions(registerSheet *sheets_service.RegisterSheet) ([]*models.Transaction, error) {
	_, err := s.RegisterService.ReadRegisterSheet(registerSheet)
	if err != nil {
		return nil, err
	}

	// Todo: account for pagination
	// Due to pagination and no code to account for it, I'm going back about 3 mounths (12 weeks).
	startDate := weeksAgo(12)
	endDate := today()
	bankIDs := []string{models.WellsFargoID, models.ChaseID}
	transactions, err := s.FetchTransactions(bankIDs, startDate, endDate)
	if err != nil {
		return nil, err
	}

	if len(transactions) < 1 {
		s.Progress("No transactions")
		return nil, nil
	}

	s.Progress("  - writing raw transactions CSV file...")
	if err := s.WriteRawTransactionsCSV(transactions); err != nil {
		return nil, err
	}
	return transactions, nil
}

// PrepareTransactions normalizes merchant names, expands keys to include
// dates before and after for config'd merchant names, dedupes agains the
// Register sheet transactions, filters out credit card transactions from the
// credit card institutions since they are already present in the checking
// account, and then sorts the remaining list. It then returns the prepared
// transactions.
func (s *UpdateService) PrepareTransactions(transactions []*models.Transaction, sheetEntries []*sheets_service.RegisterEntry) ([]*models.Transaction, []*DateCorrection, error) {
	var err error

	s.Progress("  - normalizing merchants")
	transactions, err = s.NormalizeMerchants(transactions)
	if err != nil {
		return nil, nil, err
	}

	sheetTransIDs := make(map[string]bool)
	sheetKeys := make(map[string]bool)
	dateCorrectionsByID := make(map[string]*DateCorrection)
	var dateCorrections []*DateCorrection

	for _, e := range sheetEntries {
		if e.TransactionID != "" {
			sheetTransIDs[e.TransactionID] = true
			dateCorrectionsByID[e.TransactionID] = &DateCorrection{Entry: e}
		}
		sheetKeys[e.Key] = true
	}
	for _, t := range transactions {
		if t.TransactionID != "" {
			if dateCorrectionsByID[t.TransactionID] != nil && !t.IsCheck && t.Date != dateCorrectionsByID[t.TransactionID].Entry.Date {
				c := &DateCorrection{Entry: dateCorrectionsByID[t.TransactionID].Entry, NewDate: t.Date}
				dateCorrections = append(dateCorrections, c)
			}
		}
	}

	s.Progress("  - de-duping transactions")
	transactions = s.DedupeAgainstSheet(transactions, sheetTransIDs, sheetKeys)

	s.Progress("  - filtering out credit card payments...")
	transactions = banking.FilterCreditCardPayments(transactions)

	s.Progress("  - Sorting...")
	transactions = banking.SortTransactions(transactions)

	return transactions, dateCorrections, nil
}

// ApplyTransactions takes in transactions, updates the DB, and updates the spreadsheet.
// If
func (s *UpdateService) ApplyTransactions(tabName string, transactions []*models.Transaction, dateCorrections []*DateCorrection) error {
	var err error

	s.Progress("  - updating database")
	transactions, err = s.PersistTransactions(transactions)
	if err != nil {
		return err
	}

	s.Progress("  - updating spreadsheet")

	if len(dateCorrections) == 0 || s.Options.DryRun {
		s.Progress("    > no date changes needed")
	} else {
		if err := s.ApplyDateCorrections(tabName, dateCorrections); err != nil {
			return err
		}
	}

	if len(transactions) == 0 || s.Options.DryRun {
		s.Progress("    > no updates needed")
	} else if err := s.UpdateSpreadsheet(transactions); err != nil {
		return err
	}

	s.Progress("  - getting accounts balances")
	balances := s.BankClient.GetBalances([]string{models.WellsFargoID})
	b := balances[models.WellsFargoID]
	if b.Error != nil {
		return fmt.Errorf("      Wells Fargo: balance unavailable: %v\n", b.Error)
	}
	s.Progress(fmt.Sprintf("      Wells Fargo: $%8.2f\n", b.Amount))

	s.Progress("  - updating balances in spreadsheet")
	if err := s.UpdateBalances(tabName, balances); err != nil {
		return err
	}
	return nil
}

// ApplyDateCorrections updates all spreadsheet row entries where the transaction ID matches with that of new transactions fetched
// but the date has changed.
func (s *UpdateService) ApplyDateCorrections(tabName string, dateCorrections []*DateCorrection) error {
	for _, c := range dateCorrections {
		cellData := &sheets.ValueRange{Values: [][]any{{c.NewDate}}}
		_, err := s.RegisterService.Provider.Update(fmt.Sprintf("%s!C%d", tabName, c.Entry.RowID), cellData)
		if err != nil {
			return err
		}
	}
	return nil
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
	s.Progress("  - getting CSV transactions...")
	transactions, err := getCSVTransactions(s.Config.FinanceDir, s.Config.Banks, []string{models.FidelityID}, s.Progress)
	if err != nil {
		return nil, err
	}

	var newTrans []*models.Transaction
	if s.Options.UseCSVFiles {
		//s.Progress("  - getting CSV transactions...")
		newTrans, err = getCSVTransactions(s.Config.FinanceDir, s.Config.Banks, bankIDs, s.Progress)
	} else {
		s.Progress("  - getting Plaid transactions...")
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
			strconv.FormatFloat(t.CreditPurchase, 'f', 2, 64),
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

	transactions = banking.FormatMerchantNames(transactions, matcher, income, columns)
	return transactions, nil
}

// DedupeAgainstSheet drops transactions already recorded in the register
// sheet. Matching tolerates a ±1-day date shift, because some sources (notably
// Amazon) report the same transaction on a different day from one pull to the
// next, which changes its dedupe key.
func (s *UpdateService) DedupeAgainstSheet(transactions []*models.Transaction, sheetIDs map[string]bool, sheetKeys map[string]bool) []*models.Transaction {
	return banking.FilterRecordedTransactions(transactions, sheetIDs, sheetKeys)
}

// UpdateSpreadsheet writes the transactions into the register sheet and
// refreshes the bank balances. Skipped when there is nothing to write or on
// --dry-run runs.
func (s *UpdateService) UpdateSpreadsheet(transactions []*models.Transaction) error {
	registerSheet, err := s.RegisterService.NewRegisterSheet(s.Config, nil)
	if err != nil {
		return err
	}
	budgetSheet, err := s.BudgetService.NewBudgetSheet(*s.Config)
	if err != nil {
		return err
	}

	if _, err := s.BudgetService.ReadBudgetSheet(budgetSheet); err != nil {
		return err
	}
	registerSheet, err = s.RegisterService.ReadRegisterSheet(registerSheet)
	if err != nil {
		return err
	}

	// add the needed number of rows for transactions
	s.Progress("  - adding new rows at the end...")
	if err := s.RegisterService.CopyRows(*registerSheet, len(transactions)); err != nil {
		return err
	}

	s.Progress("  - inserting new transactions...")
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
	if _, err := s.RegisterService.WriteCell(registerSheet.TabName, sheets_service.LastUpdateCell, time.Now().Format("01/02/2006")); err != nil {
		return err
	}
	if _, err := s.RegisterService.WriteCell(registerSheet.TabName, models.CheckingDeltaCell, fmt.Sprintf("=SUM(%s-I%d)", models.CheckingBalanceCell, lastRowUpdated)); err != nil {
		return err
	}
	return nil
}

// UpdateBalances takes the Wells Fargo checking account balance and updates the CheckingBalanceCell in the Register spreadsheet
// as it is used to determine if reconciliation has occurred.
func (s *UpdateService) UpdateBalances(tabName string, balances map[string]banking.Balance) error {
	if balances[models.WellsFargoID].Error == nil {
		if _, err := s.RegisterService.WriteCell(tabName, models.CheckingBalanceCell, balances[models.WellsFargoID].Amount); err != nil {
			return err
		}
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
func getCSVTransactions(finDir string, banks map[string]cfg.Bank, bankIDs []string, progress func(_ string)) ([]*models.Transaction, error) {
	csvClient := csv.New(finDir, banks, bankIDs, progress)

	transactions, err := csvClient.GetTransactions()
	if err != nil {
		return nil, err
	}
	return transactions, nil
}

// dashes is a simple helper to print out "count" number of dashes to be printed under a column header
func dashes(count int) string {
	return strings.Repeat("-", count)
}
