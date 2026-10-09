package sheets_service

import (
	"errors"
	"fmt"

	"register/pkg/config"

	"google.golang.org/api/sheets/v4"
)

// BudgetEntry is one category row of the Budget tab: the raw allocation at
// each of the two cadences Rob gets paid on. (The Yearly column on the sheet
// is display-only; the app never reads it.)
type BudgetEntry struct {
	Category string
	Weekly   float64
	Monthly  float64
}

type BudgetSheet struct {
	ID            int64
	SpreadsheetID string
	TabName       string
	Spreadsheet   sheets.Spreadsheet
	SheetCoords   SheetCoords
	BudgetEntries []*BudgetEntry
	CategoriesMap map[string]*BudgetEntry
}

func (ss *SheetsService) NewBudgetSheet(cfg *config.Config) error {
	ss.BudgetSheet = &BudgetSheet{
		TabName: "Budget",
		SheetCoords: SheetCoords{
			StartRow:       cfg.BudgetStartRow,
			EndRow:         cfg.BudgetEndRow,
			EndColumnName:  "J",
			EndColumnIndex: 9,
		},
	}
	props, err := ss.getSheetProperties("Budget")
	if err != nil {
		return err
	}
	ss.BudgetSheet.ID = props.SheetId
	return nil
}

func (ss *SheetsService) ReadBudgetSheet() (*BudgetSheet, error) {
	range_ := fmt.Sprintf("%s!B%d:%s%d", ss.BudgetSheet.TabName, ss.BudgetSheet.SheetCoords.StartRow, ss.BudgetSheet.SheetCoords.EndColumnName, ss.BudgetSheet.SheetCoords.EndRow)
	resp, err := ss.Provider.GetValues(range_)
	if err != nil {
		return nil, fmt.Errorf("could not get sheet values: %s\n", err.Error())
	}
	if len(resp.Values) == 0 {
		return nil, errors.New("no values found")
	}

	var entries []*BudgetEntry
	var categoriesMap = make(map[string]*BudgetEntry)
	for _, values := range resp.Values {
		if ss.isEmptyBudgetRow(values) {
			continue
		}
		entry, err := ss.populateBudgetEntry(values)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
		categoriesMap[ss.getBudgetCategory(values)] = entry
	}
	ss.BudgetSheet.BudgetEntries = entries
	ss.BudgetSheet.CategoriesMap = categoriesMap
	return ss.BudgetSheet, nil
}

func (ss *SheetsService) isEmptyBudgetRow(values []interface{}) bool {
	if len(values) < 7 || ss.getBudgetCategory(values) == "" {
		return true
	}
	return false
}

func (ss *SheetsService) getBudgetCategory(values []interface{}) string {
	return fmt.Sprintf("%s", values[0])
}

func (ss *SheetsService) populateBudgetEntry(values []interface{}) (*BudgetEntry, error) {
	weekly, err := readDollarsValue(values[2])
	if err != nil {
		return nil, err
	}
	monthly, err := readDollarsValue(values[3])
	if err != nil {
		return nil, err
	}
	return &BudgetEntry{
		Category: ss.getBudgetCategory(values),
		Weekly:   weekly,
		Monthly:  monthly,
	}, nil
}
