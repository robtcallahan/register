package sheets_service

import (
	"testing"

	"register/pkg/models"
)

// fixed block (1-10), then green 11-14, yellow 15-17, black 18-19
// (the card payoff columns), blue 20-24 — Taxes is a savings column.
func moveTestColumns() []models.Column {
	mk := func(i int, name, color string) models.Column {
		return models.Column{ID: i, Name: name, Color: color, ColumnIndex: i}
	}
	cols := []models.Column{}
	for i := 1; i <= 10; i++ {
		cols = append(cols, mk(i, "", ""))
	}
	cols = append(cols,
		mk(11, "Cash", models.ColorGreen), mk(12, "Dining", models.ColorGreen),
		mk(13, "Grocery", models.ColorGreen), mk(14, "Misc", models.ColorGreen),
		mk(15, "Rent", models.ColorYellow), mk(16, "Electric", models.ColorYellow),
		mk(17, "ISP", models.ColorYellow),
		mk(18, "Credit Cards", models.ColorBlack), mk(19, "AppleCard", models.ColorBlack),
		mk(20, "Savings", models.ColorBlue), mk(21, "Gifts", models.ColorBlue),
		mk(22, "Medical", models.ColorBlue), mk(23, "Emergency Fund", models.ColorBlue),
		mk(24, "Taxes", models.ColorBlue),
	)
	return cols
}

func TestPlanColumnMove(t *testing.T) {
	cols := moveTestColumns()
	cases := []struct {
		name        string
		from, after int
		wantTo      int
		wantChanged bool
		wantErr     bool
	}{
		{"within group, moving left", 13, 11, 12, true, false},
		{"within group, moving right", 12, 14, 14, true, false},
		{"already after target", 13, 12, 13, false, false},
		{"move self is an error", 13, 13, 0, false, true},
		{"green into yellow refused", 13, 16, 0, false, true},
		{"yellow into green refused", 15, 12, 0, false, true},
		{"fixed block cannot move", 5, 13, 0, false, true},
		{"after a fixed column refused", 13, 9, 0, false, true},
		{"blue moves within savings", 24, 22, 23, true, false},
		{"black leaving the payoff block refused", 19, 0, 0, false, true},
		{"black splitting blue refused", 18, 21, 0, false, true},
		{"category to the end refused (leaves its group)", 13, 0, 0, false, true},
		{"unknown from index", 99, 11, 0, false, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			to, changed, err := PlanColumnMove(cols, tc.from, tc.after)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("PlanColumnMove(%d, %d) = (%d, %v, nil), want error", tc.from, tc.after, to, changed)
				}
				return
			}
			if err != nil {
				t.Fatalf("PlanColumnMove(%d, %d) error: %v", tc.from, tc.after, err)
			}
			if to != tc.wantTo || changed != tc.wantChanged {
				t.Errorf("PlanColumnMove(%d, %d) = (%d, %v), want (%d, %v)",
					tc.from, tc.after, to, changed, tc.wantTo, tc.wantChanged)
			}
		})
	}
}
