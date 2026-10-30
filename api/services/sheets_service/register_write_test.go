package sheets_service

import (
	"strings"
	"testing"

	"register/pkg/models"
)

func Test_registerService_UpdateRows(t *testing.T) {
	sheet := RegisterSheet{
		ID:          1,
		TabName:     RegisterTabName,
		SheetCoords: SheetCoords{FirstRowToUpdate: 5, EndColumnIndex: 11},
	}
	budget := BudgetSheet{}
	columns := []models.Column{
		{Name: "Grocery", ColumnIndex: 11, Color: models.ColorBlue},
		{Name: "Rent", ColumnIndex: 12, Color: models.ColorBlue},
	}
	mapping := map[string]string{"TARGET": "Grocery"}
	getFormulaProviderFunc = func(range_ string) ([][]any, error) {
		return [][]any{{"=F1", "=F2", "=F3"}}, nil
	}
	t.Run("happy path", func(t *testing.T) {
		trans := []*models.Transaction{
			{Source: "WellsFargo", Date: "12/07/20", Name: "TARGET", Withdrawal: 46.80, Budget: 46.80},
		}
		if err := ss.UpdateRows(sheet, budget, columns, mapping, trans); err != nil {
			t.Errorf("UpdateRows() error = %v, want nil", err)
		}
	})
	t.Run("bad date fails", func(t *testing.T) {
		trans := []*models.Transaction{
			{Source: "WellsFargo", Date: "bogus", Name: "TARGET", Withdrawal: 46.80, Budget: 46.80},
		}
		if err := ss.UpdateRows(sheet, budget, columns, mapping, trans); err == nil {
			t.Errorf("UpdateRows() error = nil, want error")
		}
	})
	t.Run("no transactions", func(t *testing.T) {
		if err := ss.UpdateRows(sheet, budget, columns, mapping, nil); err != nil {
			t.Errorf("UpdateRows(nil) error = %v, want nil", err)
		}
	})
}

func Test_registerService_getGridCoordinate(t *testing.T) {
	sheet := RegisterSheet{
		ID:          99,
		SheetCoords: SheetCoords{FirstRowToUpdate: 7},
	}
	got := ss.getGridCoordinate(sheet)
	if got.SheetId != 99 || got.RowIndex != 7 || got.ColumnIndex != 0 {
		t.Errorf("getGridCoordinate() = %+v, want {99 7 0}", got)
	}
}

func Test_registerService_PlanCategoryMoves(t *testing.T) {
	columns := []models.Column{
		{Name: "Grocery", ColumnIndex: 11, Color: models.ColorBlue},
		{Name: "Rent", ColumnIndex: 12, Color: models.ColorBlue},
		{Name: "Credit Cards", ColumnIndex: 13, Color: models.ColorGrey},
	}
	target := models.Column{Name: "Rent", ColumnIndex: 12, Color: models.ColorBlue}
	row := func(grocery, rent, card string) []any {
		values := make([]any, 13)
		values[10] = grocery
		values[11] = rent
		values[12] = card
		return values
	}
	sheet := func(entries []*RegisterEntry, values [][]any) RegisterSheet {
		return RegisterSheet{
			TabName:     RegisterTabName,
			SheetCoords: SheetCoords{StartRow: 5},
			Entries:     entries,
			RangeValues: values,
		}
	}
	t.Run("clean move", func(t *testing.T) {
		entries := []*RegisterEntry{{RowID: 5, Name: "TARGET store", Date: "12/07/20", Key: "k1"}}
		moves, notes := ss.PlanCategoryMoves(sheet(entries, [][]any{row("46.80", "", "99.00")}), []string{"target"}, target, columns)
		if len(notes) != 0 {
			t.Fatalf("notes = %v, want none", notes)
		}
		if len(moves) != 1 {
			t.Fatalf("len(moves) = %d, want 1", len(moves))
		}
		m := moves[0]
		if m.Key != "k1" || m.RowID != 5 || m.FromIndex != 11 || m.ToIndex != 12 || m.Value != 46.80 || m.Color != models.ColorBlue {
			t.Errorf("move = %+v, want k1/5/11->12/46.80/blue", m)
		}
	})
	t.Run("no match", func(t *testing.T) {
		entries := []*RegisterEntry{{RowID: 5, Name: "OTHER", Date: "12/07/20", Key: "k1"}}
		moves, notes := ss.PlanCategoryMoves(sheet(entries, [][]any{row("46.80", "", "")}), []string{"target"}, target, columns)
		if len(moves) != 0 || len(notes) != 0 {
			t.Errorf("moves/notes = %v/%v, want empty", moves, notes)
		}
	})
	t.Run("no amount", func(t *testing.T) {
		entries := []*RegisterEntry{{RowID: 5, Name: "TARGET", Date: "12/07/20", Key: "k1"}}
		moves, notes := ss.PlanCategoryMoves(sheet(entries, [][]any{row("", "", "")}), []string{"target"}, target, columns)
		if len(moves) != 0 {
			t.Errorf("moves = %v, want none", moves)
		}
		if len(notes) != 1 || !strings.Contains(notes[0], "no category amount") {
			t.Errorf("notes = %v, want no-amount note", notes)
		}
	})
	t.Run("multiple amounts", func(t *testing.T) {
		entries := []*RegisterEntry{{RowID: 5, Name: "TARGET", Date: "12/07/20", Key: "k1"}}
		moves, notes := ss.PlanCategoryMoves(sheet(entries, [][]any{row("10.00", "20.00", "")}), []string{"target"}, target, columns)
		if len(moves) != 0 {
			t.Errorf("moves = %v, want none", moves)
		}
		if len(notes) != 1 || !strings.Contains(notes[0], "multiple category amounts") {
			t.Errorf("notes = %v, want multiple-amounts note", notes)
		}
	})
	t.Run("already in target", func(t *testing.T) {
		entries := []*RegisterEntry{{RowID: 5, Name: "TARGET", Date: "12/07/20", Key: "k1"}}
		moves, notes := ss.PlanCategoryMoves(sheet(entries, [][]any{row("", "20.00", "")}), []string{"target"}, target, columns)
		if len(moves) != 0 {
			t.Errorf("moves = %v, want none", moves)
		}
		if len(notes) != 1 || !strings.Contains(notes[0], "already in Rent") {
			t.Errorf("notes = %v, want already-in-target note", notes)
		}
	})
}

func Test_registerService_MoveCategoryCells(t *testing.T) {
	sheet := RegisterSheet{ID: 1, TabName: RegisterTabName}
	if err := ss.MoveCategoryCells(sheet, nil); err != nil {
		t.Errorf("MoveCategoryCells(nil) error = %v, want nil", err)
	}
	moves := []CategoryMove{
		{Key: "k1", RowID: 5, FromIndex: 11, ToIndex: 12, Value: 46.80, Color: models.ColorBlue},
	}
	if err := ss.MoveCategoryCells(sheet, moves); err != nil {
		t.Errorf("MoveCategoryCells() error = %v, want nil", err)
	}
}
