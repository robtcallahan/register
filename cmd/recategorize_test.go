package cmd

import (
	"strings"
	"testing"

	"register/api/services/sheets_service"
	"register/pkg/models"
)

func TestSyncDB(t *testing.T) {
	target := &models.Column{Name: "Rent", ColumnIndex: 12, Color: models.ColorBlue}

	t.Run("moves update the matching transactions", func(t *testing.T) {
		repo := &fakeQueryRepo{transactions: []models.Transaction{
			{Key: "k1", ColumnIndex: 11, Color: models.ColorGreen},
			{Key: "k2", ColumnIndex: 12, Color: models.ColorYellow},
		}}
		moves := []sheets_service.CategoryMove{
			{Key: "k1", RowID: 5, FromIndex: 11, Value: 25},
			{Key: "missing", RowID: 6, FromIndex: 12, Value: 10},
		}
		var err error
		out := captureStdout(t, func() { err = syncDB(repo, moves, target) })
		if err != nil {
			t.Fatalf("syncDB returned error: %v", err)
		}
		if repo.transactions[0].ColumnIndex != 12 || repo.transactions[0].Color != models.ColorBlue {
			t.Errorf("k1 after sync = index %d color %q, want index 12 color %q",
				repo.transactions[0].ColumnIndex, repo.transactions[0].Color, models.ColorBlue)
		}
		if repo.transactions[1].ColumnIndex != 12 || repo.transactions[1].Color != models.ColorYellow {
			t.Errorf("k2 changed unexpectedly: %+v", repo.transactions[1])
		}
		if !strings.Contains(out, "note: row 6 has no matching DB transaction") {
			t.Errorf("output missing the unmatched-row note:\n%s", out)
		}
		if !strings.Contains(out, "moved 2 transaction(s) to Rent") {
			t.Errorf("output missing the moved count:\n%s", out)
		}
	})

	t.Run("no moves moves nothing", func(t *testing.T) {
		repo := &fakeQueryRepo{transactions: []models.Transaction{
			{Key: "k1", ColumnIndex: 11, Color: models.ColorGreen},
		}}
		var err error
		out := captureStdout(t, func() { err = syncDB(repo, nil, target) })
		if err != nil {
			t.Fatalf("syncDB returned error: %v", err)
		}
		if repo.transactions[0].ColumnIndex != 11 {
			t.Errorf("transaction changed unexpectedly: %+v", repo.transactions[0])
		}
		if !strings.Contains(out, "moved 0 transaction(s) to Rent") {
			t.Errorf("output missing the moved count:\n%s", out)
		}
	})

	t.Run("transaction read error is wrapped", func(t *testing.T) {
		repo := &fakeQueryRepo{transactionsErr: errTestBoom}
		moves := []sheets_service.CategoryMove{{Key: "k1", RowID: 5, FromIndex: 11, Value: 25}}
		var err error
		captureStdout(t, func() { err = syncDB(repo, moves, target) })
		if err == nil {
			t.Fatal("syncDB returned nil error, want the read error")
		}
		if !strings.Contains(err.Error(), "could not read transactions for the DB update") {
			t.Errorf("syncDB error = %q, want it to mention the read failure", err.Error())
		}
	})

	t.Run("transaction write error is wrapped", func(t *testing.T) {
		repo := &fakeQueryRepo{
			transactions: []models.Transaction{{Key: "k1", ColumnIndex: 11, Color: models.ColorGreen}},
			updateErr:    errTestBoom,
		}
		moves := []sheets_service.CategoryMove{{Key: "k1", RowID: 5, FromIndex: 11, Value: 25}}
		var err error
		captureStdout(t, func() { err = syncDB(repo, moves, target) })
		if err == nil {
			t.Fatal("syncDB returned nil error, want the write error")
		}
		if !strings.Contains(err.Error(), "DB update failed") {
			t.Errorf("syncDB error = %q, want it to mention the DB update failure", err.Error())
		}
	})
}
