package sheets_service

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"register/pkg/models"
)

func hasNote(trans *models.Transaction) bool {
	if trans.Note != "" {
		return true
	}
	return false
}

func getRegisterToDeltaReadRange(i int64) string {
	return fmt.Sprintf("%s!%s%d:%s%d", RegisterTabName, RegisterColumn, i+1, DeltaColumn, i+1)
}

func isCheckingAccount(trans *models.Transaction) bool {
	if trans.Source == CheckingAccountSourceName || trans.IsCheck {
		return true
	}
	return false
}

func isBudgetColumn(colName string) bool {
	if colName != "" && colName != CreditCardColumnName {
		return true
	}
	return false
}

func isCorrectBudgetColumn(transName, colName string, transNameToColName map[string]string) bool {
	if _, ok := transNameToColName[transName]; ok {
		if colName == transNameToColName[transName] {
			return true
		}
	}
	return false
}

func intInSlice(a int, list []int) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func sortAggregateMapKeys(aggMap *map[string]map[string]float64) *[]string {
	keys := make([]string, 0, len(*aggMap))
	for k := range *aggMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return &keys
}

func isCreditCardTransaction(source, colName string) bool {
	if source != CheckingAccountSourceName && colName == CreditCardColumnName {
		return true
	}
	return false
}

func isRegisterClearedOrDeltaColumn(i int) bool {
	if ok := intInSlice(i, []int{0, 1, 2}); ok {
		return true
	}
	return false
}

func WriteJSONFile(fileName string, data interface{}) error {
	j, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("error: %s\n", err.Error())
	}
	err = os.WriteFile(fileName, j, 0644)
	if err != nil {
		return fmt.Errorf("error: %s\n", err.Error())
	}
	return nil
}
