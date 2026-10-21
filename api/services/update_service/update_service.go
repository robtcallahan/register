package update_service

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"register/pkg/banking"
	"register/pkg/models"
)

type UpdateService struct {
	BankClient *banking.Client
}

func NewUpdateService(bankClient *banking.Client) *UpdateService {
	return &UpdateService{
		BankClient: bankClient,
	}
}

// dedupeAgainstSheet drops transactions already recorded in the register
// sheet. Matching tolerates a ±1-day date shift, because some sources (notably
// Amazon) report the same transaction on a different day from one pull to the
// next, which changes its dedupe key.
func (s *UpdateService) DedupeAgainstSheet(transactions []*models.Transaction, keysMap map[string]bool) []*models.Transaction {
	fmt.Println("Filtering out register transactions...")
	return s.BankClient.FilterRecordedTransactions(transactions, fuzzyKeySet(keysMap))
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
