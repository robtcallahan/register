package sheets_service

import (
	"encoding/json"
	"fmt"
	"os"
	"register/pkg/models"
	"slices"
	"sort"
)

func hasNote(trans *models.Transaction) bool {
	return trans.Note != ""
}

func getRegisterToDeltaReadRange(i int64) string {
	return fmt.Sprintf("%s!%s%d:%s%d", RegisterTabName, RegisterColumn, i+1, DeltaColumn, i+1)
}

func isCheckingAccount(trans *models.Transaction) bool {
	return trans.Source == CheckingAccountSourceName || trans.IsCheck
}

func isBudgetColumn(colName string) bool {
	return colName != "" && colName != CreditCardColumnName
}

func isCorrectBudgetColumn(transName, colName string, transNameToColName map[string]string) bool {
	col, ok := transNameToColName[transName]
	return ok && colName == col
}

func sortAggregateMapKeys(aggMap map[string]map[string]float64) []string {
	keys := make([]string, 0, len(aggMap))
	for k := range aggMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func isCreditCardTransaction(source, colName string) bool {
	return source != CheckingAccountSourceName && colName == CreditCardColumnName
}

func isRegisterClearedOrDeltaColumn(i int) bool {
	return slices.Contains([]int{0, 1, 2}, i)
}

func WriteJSONFile(fileName string, data any) error {
	j, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("error: %w\n", err)
	}
	err = os.WriteFile(fileName, j, 0644)
	if err != nil {
		return fmt.Errorf("error: %w\n", err)
	}
	return nil
}
