package sheets_service

import (
	"testing"

	"register/pkg/config"
	"register/pkg/models"
)

func Test_rowBuilder_populateCells(t *testing.T) {
	columns := []models.Column{
		{Name: "Register", ColumnIndex: 8},
		{Name: "Cleared", ColumnIndex: 9},
		{Name: "Delta", ColumnIndex: 10},
		{Name: "Grocery", ColumnIndex: 11, Color: models.ColorBlue},
	}
	formulas := func(rowIndex int64) ([]string, error) {
		return []string{"=F1", "=F2", "=F3"}, nil
	}
	t.Run("checking transaction", func(t *testing.T) {
		rb := &RowBuilder{
			Columns:            columns,
			TransNameToColName: map[string]string{"TARGET": "Grocery"},
			FirstRowToUpdate:   5,
			ReadFormulas:       formulas,
		}
		trans := []*models.Transaction{
			{Source: "WellsFargo", Date: "12/07/20", Name: "TARGET", Withdrawal: 46.80, Budget: 46.80},
		}
		rows, err := rb.populateCells(trans)
		if err != nil {
			t.Fatalf("populateCells() error = %v, want nil", err)
		}
		if len(rows) != 2 {
			t.Fatalf("len(rows) = %d, want 2 (data + empty second row)", len(rows))
		}
		data := rows[0]
		if len(data.Values) != 11 {
			t.Fatalf("len(data cells) = %d, want 11", len(data.Values))
		}
		if got := *data.Values[3].UserEnteredValue.StringValue; got != "TARGET" {
			t.Errorf("name cell = %q, want TARGET", got)
		}
		if got := *data.Values[4].UserEnteredValue.NumberValue; got != 46.80 {
			t.Errorf("withdrawal cell = %v, want 46.80", got)
		}
		for i, want := range []string{"=F1", "=F2", "=F3"} {
			if got := *data.Values[7+i].UserEnteredValue.FormulaValue; got != want {
				t.Errorf("formula cell[%d] = %q, want %q", i, got, want)
			}
		}
		if got := *data.Values[10].UserEnteredValue.NumberValue; got != 46.80 {
			t.Errorf("Grocery cell = %v, want 46.80", got)
		}
		if len(rows[1].Values) != 0 {
			t.Errorf("second row len = %d, want 0 (no note)", len(rows[1].Values))
		}
	})
	t.Run("income transaction uses salary allocation", func(t *testing.T) {
		rb := &RowBuilder{
			Columns:       columns,
			IncomeSources: []config.IncomeSource{{Name: "Taylor House Paycheck", Frequency: "biweekly"}},
			CategoriesMap: map[string]*BudgetEntry{
				"Grocery": {Category: "Grocery", BiWeekly: 240},
			},
			FirstRowToUpdate: 5,
			ReadFormulas:     formulas,
		}
		trans := []*models.Transaction{
			{Source: "WellsFargo", Date: "12/07/20", Name: "Taylor House Paycheck", Deposit: 1000},
		}
		rows, err := rb.populateCells(trans)
		if err != nil {
			t.Fatalf("populateCells() error = %v, want nil", err)
		}
		if len(rows) != 2 {
			t.Fatalf("len(rows) = %d, want 2", len(rows))
		}
		if got := *rows[0].Values[10].UserEnteredValue.NumberValue; got != 240 {
			t.Errorf("Grocery salary cell = %v, want 240 (biweekly allocation)", got)
		}
	})
	t.Run("note transaction adds a note row", func(t *testing.T) {
		rb := &RowBuilder{
			Columns:            columns,
			TransNameToColName: map[string]string{},
			FirstRowToUpdate:   5,
			ReadFormulas:       formulas,
		}
		trans := []*models.Transaction{
			{Source: "WellsFargo", Date: "12/07/20", Name: "TARGET", Withdrawal: 10, Note: "call bank"},
		}
		rows, err := rb.populateCells(trans)
		if err != nil {
			t.Fatalf("populateCells() error = %v, want nil", err)
		}
		if len(rows) != 2 {
			t.Fatalf("len(rows) = %d, want 2", len(rows))
		}
		note := rows[1]
		if len(note.Values) != 4 {
			t.Fatalf("len(note row) = %d, want 4", len(note.Values))
		}
		if got := *note.Values[3].UserEnteredValue.StringValue; got != "call bank" {
			t.Errorf("note cell = %q, want call bank", got)
		}
	})
	t.Run("formula read error", func(t *testing.T) {
		rb := &RowBuilder{
			Columns:      columns,
			ReadFormulas: func(rowIndex int64) ([]string, error) { return nil, errTestProvider },
		}
		trans := []*models.Transaction{
			{Source: "WellsFargo", Date: "12/07/20", Name: "TARGET", Withdrawal: 10},
		}
		if _, err := rb.populateCells(trans); err == nil {
			t.Errorf("populateCells() error = nil, want error")
		}
	})
}

func Test_rowBuilder_makeNoteRow(t *testing.T) {
	rb := &RowBuilder{EndColumnIndex: 12}
	got := rb.makeNoteRow("call bank")
	if len(got.Values) != 4 {
		t.Fatalf("len(cells) = %d, want 4", len(got.Values))
	}
	for i := 0; i < 3; i++ {
		if got := *got.Values[i].UserEnteredValue.StringValue; got != "" {
			t.Errorf("cell[%d] = %q, want empty", i, got)
		}
	}
	if got := *got.Values[3].UserEnteredValue.StringValue; got != "call bank" {
		t.Errorf("note cell = %q, want call bank", got)
	}
}
