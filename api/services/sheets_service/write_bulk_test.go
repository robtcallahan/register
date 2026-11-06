package sheets_service

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"register/pkg/models"
)

func Test_registerService_UpdateRows_BatchUpdateError(t *testing.T) {
	stub := &stubSheetsProvider{
		getFormula:     [][]any{{"=F1", "=F2", "=F3"}},
		batchUpdateErr: errTestProvider,
	}
	svc := NewRegisterService(stub)
	sheet := RegisterSheet{
		ID:          1,
		TabName:     RegisterTabName,
		SheetCoords: SheetCoords{FirstRowToUpdate: 5, EndColumnIndex: 11},
	}
	columns := []models.Column{
		{Name: "Grocery", ColumnIndex: 11, Color: models.ColorBlue},
		{Name: "Rent", ColumnIndex: 12, Color: models.ColorBlue},
	}
	trans := []*models.Transaction{
		{Source: "WellsFargo", Date: "12/07/20", Name: "TARGET", Withdrawal: 46.80, Budget: 46.80},
	}
	if err := svc.UpdateRows(sheet, BudgetSheet{}, columns, map[string]string{"TARGET": "Grocery"}, trans); !errors.Is(err, errTestProvider) {
		t.Errorf("UpdateRows() error = %v, wantErr %v", err, errTestProvider)
	}
}

func Test_addCategoryCells_FixedBlockAndCreditCard(t *testing.T) {
	formulas := []string{"=F1", "=F2", "=F3"}
	t.Run("fixed block column is skipped", func(t *testing.T) {
		columns := []models.Column{
			{Name: "Deposits", ColumnIndex: 5},
			{Name: "Grocery", ColumnIndex: 11, Color: models.ColorBlue},
			{Name: "Rent", ColumnIndex: 12, Color: models.ColorBlue},
		}
		trans := &models.Transaction{Name: "TARGET", Budget: 46.80}
		got := addCategoryCells(nil, trans, columns, map[string]string{"TARGET": "Grocery"}, formulas)
		// 3 gap fillers (8, 9, 10) + Grocery + Rent = 5 cells.
		if len(got) != 5 {
			t.Fatalf("addCategoryCells() produced %d cells, want 5", len(got))
		}
		if got := *got[3].UserEnteredValue.NumberValue; got != 46.80 {
			t.Errorf("Grocery cell = %v, want 46.80", got)
		}
	})
	t.Run("credit card column takes the purchase", func(t *testing.T) {
		columns := []models.Column{
			{Name: "Credit Cards", ColumnIndex: 11, Color: models.ColorGrey},
			{Name: "Grocery", ColumnIndex: 12, Color: models.ColorBlue},
		}
		trans := &models.Transaction{Source: "Fidelity", Name: "AMAZON", CreditPurchase: 25.00}
		got := addCategoryCells(nil, trans, columns, map[string]string{}, formulas)
		if len(got) != 5 {
			t.Fatalf("addCategoryCells() produced %d cells, want 5", len(got))
		}
		if got := *got[3].UserEnteredValue.NumberValue; got != 25.00 {
			t.Errorf("Credit Cards cell = %v, want 25.00", got)
		}
		if !reflect.DeepEqual(got[3].UserEnteredFormat.BackgroundColor, mkColor(models.ColorGrey)) {
			t.Errorf("Credit Cards cell color = %v, want grey", got[3].UserEnteredFormat.BackgroundColor)
		}
	})
}

func Test_registerService_PlanCategoryMoves_UnparseableCell(t *testing.T) {
	columns := []models.Column{
		{Name: "Grocery", ColumnIndex: 11, Color: models.ColorBlue},
		{Name: "Rent", ColumnIndex: 12, Color: models.ColorBlue},
		{Name: "Credit Cards", ColumnIndex: 13, Color: models.ColorGrey},
	}
	target := models.Column{Name: "Rent", ColumnIndex: 12, Color: models.ColorBlue}
	row := make([]any, 13)
	row[10] = "xx" // unparseable Grocery cell: the read errors, the row is skipped
	sheet := RegisterSheet{
		TabName:     RegisterTabName,
		SheetCoords: SheetCoords{StartRow: 5},
		Entries:     []*RegisterEntry{{RowID: 5, Name: "TARGET", Date: "12/07/20", Key: "k1"}},
		RangeValues: [][]any{row},
	}
	moves, notes := ss.PlanCategoryMoves(sheet, []string{"target"}, target, columns)
	if len(moves) != 0 {
		t.Errorf("moves = %v, want none", moves)
	}
	if len(notes) != 1 || !strings.Contains(notes[0], "no category amount") {
		t.Errorf("notes = %v, want no-amount note", notes)
	}
}

func Test_registerService_MoveCategoryCells_BatchUpdateError(t *testing.T) {
	svc := NewRegisterService(&stubSheetsProvider{batchUpdateErr: errTestProvider})
	sheet := RegisterSheet{ID: 1, TabName: RegisterTabName}
	moves := []CategoryMove{
		{Key: "k1", RowID: 5, FromIndex: 11, ToIndex: 12, Value: 46.80, Color: models.ColorBlue},
	}
	if err := svc.MoveCategoryCells(sheet, moves); err == nil || !strings.Contains(err.Error(), "clear register rows") {
		t.Errorf("MoveCategoryCells() error = %v, want clear-rows error", err)
	}
}

func Test_addSalaryCells_UnbudgetedAndFixedBlock(t *testing.T) {
	t.Run("fixed block column is skipped", func(t *testing.T) {
		rb := &RowBuilder{
			Columns: []models.Column{
				{Name: "Deposits", ColumnIndex: 5},
				{Name: "Grocery", ColumnIndex: 11, Color: models.ColorBlue},
			},
			CategoriesMap: map[string]*BudgetEntry{
				"Grocery": {Category: "Grocery", Weekly: 120.00, Monthly: 520.00},
			},
		}
		got := rb.addSalaryCells(nil, []string{"", "", ""}, "weekly")
		// 3 gap fillers (8, 9, 10) + Grocery = 4 cells.
		if len(got) != 4 {
			t.Fatalf("addSalaryCells() produced %d cells, want 4", len(got))
		}
		if got := *got[3].UserEnteredValue.NumberValue; got != 120 {
			t.Errorf("Grocery salary cell = %v, want 120", got)
		}
	})
	t.Run("category with no budget entry gets a zero cell", func(t *testing.T) {
		rb := &RowBuilder{
			Columns: []models.Column{
				{Name: "Phone", ColumnIndex: 11, Color: models.ColorBlue},
			},
			CategoriesMap: map[string]*BudgetEntry{},
		}
		got := rb.addSalaryCells(nil, []string{"", "", ""}, "weekly")
		if len(got) != 4 {
			t.Fatalf("addSalaryCells() produced %d cells, want 4", len(got))
		}
		if got := *got[3].UserEnteredValue.NumberValue; got != 0 {
			t.Errorf("Phone salary cell = %v, want 0", got)
		}
		if !reflect.DeepEqual(got[3].UserEnteredFormat.BackgroundColor, mkColor(models.ColorBlue)) {
			t.Errorf("Phone salary cell color = %v, want blue", got[3].UserEnteredFormat.BackgroundColor)
		}
	})
	t.Run("credit card column is not a budget column", func(t *testing.T) {
		rb := &RowBuilder{
			Columns: []models.Column{
				{Name: "Credit Cards", ColumnIndex: 11, Color: models.ColorGrey},
			},
			CategoriesMap: map[string]*BudgetEntry{},
		}
		got := rb.addSalaryCells(nil, []string{"", "", ""}, "weekly")
		if len(got) != 4 {
			t.Fatalf("addSalaryCells() produced %d cells, want 4", len(got))
		}
		if got := *got[3].UserEnteredValue.NumberValue; got != 0 {
			t.Errorf("Credit Cards salary cell = %v, want 0", got)
		}
	})
}
