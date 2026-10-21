package update_service

import (
	"fmt"
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
	BankClient *banking.Client
	QueryRepo  repository.QueryRepo
	Config     *cfg.Config
	Options    *cfg.Options
}

func NewUpdateService(config *cfg.Config, options *cfg.Options, bankClient *banking.Client, queryRepo repository.QueryRepo) *UpdateService {
	return &UpdateService{
		BankClient: bankClient,
		QueryRepo:  queryRepo,
		Config:     config,
		Options:    options,
	}
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
