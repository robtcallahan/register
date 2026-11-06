package cmd

import (
	"strings"
	"testing"

	"register/pkg/models"
)

func TestConfirm(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"lowercase y", "y\n", true},
		{"uppercase Y", "Y\n", true},
		{"y with surrounding spaces", "  y  \n", true},
		{"lowercase n", "n\n", false},
		{"uppercase N", "N\n", false},
		{"empty line", "\n", false},
		{"empty input", "", false},
		{"yes is not y", "yes\n", false},
		{"y with trailing text", "yep\n", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var got bool
			captureStdout(t, func() {
				got = confirm(newReader(tc.input), "Delete? [y/N] ")
			})
			if got != tc.want {
				t.Errorf("confirm(%q) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}

func TestFindBudgetNeighbor(t *testing.T) {
	// Budget rows exist for every category except the ones noted.
	withBudget := func(names ...string) map[string]int {
		m := make(map[string]int, len(names))
		for i, n := range names {
			m[n] = i + 5 // row numbers are irrelevant, only presence matters
		}
		return m
	}

	tests := []struct {
		name       string
		final      []models.Column
		moved      string
		color      string
		budget     map[string]int
		wantBefore string
		wantAfter  string
	}{
		{
			name: "neighbors on both sides",
			final: []models.Column{
				{Name: "A", Color: models.ColorGreen},
				{Name: "B", Color: models.ColorGreen},
				{Name: "Moved", Color: models.ColorGreen},
				{Name: "C", Color: models.ColorGreen},
				{Name: "D", Color: models.ColorYellow},
			},
			moved: "Moved", color: models.ColorGreen,
			budget:     withBudget("A", "B", "Moved", "C", "D"),
			wantBefore: "B", wantAfter: "C",
		},
		{
			name: "no same-color neighbors",
			final: []models.Column{
				{Name: "X", Color: models.ColorYellow},
				{Name: "Moved", Color: models.ColorGreen},
				{Name: "Y", Color: models.ColorYellow},
			},
			moved: "Moved", color: models.ColorGreen,
			budget:     withBudget("X", "Moved", "Y"),
			wantBefore: "", wantAfter: "",
		},
		{
			name: "neighbor without a budget row is skipped",
			final: []models.Column{
				{Name: "A", Color: models.ColorGreen},
				{Name: "B", Color: models.ColorGreen},
				{Name: "Moved", Color: models.ColorGreen},
			},
			moved: "Moved", color: models.ColorGreen,
			budget:     withBudget("B", "Moved"),
			wantBefore: "B", wantAfter: "",
		},
		{
			name: "nearest before is the one right ahead, not the far one",
			final: []models.Column{
				{Name: "A", Color: models.ColorGreen},
				{Name: "B", Color: models.ColorGreen},
				{Name: "Moved", Color: models.ColorGreen},
			},
			moved: "Moved", color: models.ColorGreen,
			budget:     withBudget("A", "B", "Moved"),
			wantBefore: "B", wantAfter: "",
		},
		{
			name: "nearest following is the one right after, not the far one",
			final: []models.Column{
				{Name: "Moved", Color: models.ColorGreen},
				{Name: "C", Color: models.ColorGreen},
				{Name: "D", Color: models.ColorGreen},
			},
			moved: "Moved", color: models.ColorGreen,
			budget:     withBudget("Moved", "C", "D"),
			wantBefore: "", wantAfter: "C",
		},
		{
			name: "moved first has only a following neighbor",
			final: []models.Column{
				{Name: "Moved", Color: models.ColorGreen},
				{Name: "A", Color: models.ColorGreen},
			},
			moved: "Moved", color: models.ColorGreen,
			budget:     withBudget("Moved", "A"),
			wantBefore: "", wantAfter: "A",
		},
		{
			name: "moved last has only a before neighbor",
			final: []models.Column{
				{Name: "A", Color: models.ColorGreen},
				{Name: "Moved", Color: models.ColorGreen},
			},
			moved: "Moved", color: models.ColorGreen,
			budget:     withBudget("A", "Moved"),
			wantBefore: "A", wantAfter: "",
		},
		{
			name: "the moved column itself is never its own neighbor",
			final: []models.Column{
				{Name: "Moved", Color: models.ColorGreen},
			},
			moved: "Moved", color: models.ColorGreen,
			budget:     withBudget("Moved"),
			wantBefore: "", wantAfter: "",
		},
		{
			name:  "empty order",
			final: []models.Column{},
			moved: "Moved", color: models.ColorGreen,
			budget:     withBudget(),
			wantBefore: "", wantAfter: "",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			before, following := findBudgetNeighbor(tc.final, tc.moved, tc.color, tc.budget)
			if before != tc.wantBefore || following != tc.wantAfter {
				t.Errorf("findBudgetNeighbor = (%q, %q), want (%q, %q)",
					before, following, tc.wantBefore, tc.wantAfter)
			}
		})
	}
}

func TestAddColumn_Guards(t *testing.T) {
	oldColor, oldAfter := addColor, addAfter
	defer func() { addColor, addAfter = oldColor, oldAfter }()

	tests := []struct {
		name  string
		color string
		col   string
		want  string
	}{
		{"grey is reserved", "grey", "Test", "--color must be green, yellow, or blue"},
		{"empty color", "", "Test", "--color must be green, yellow, or blue"},
		{"unknown color", "red", "Test", "--color must be green, yellow, or blue"},
		{"empty name", "green", "", "column name must not be empty"},
		{"blank name", "green", "   ", "column name must not be empty"},
		{"tab name", "yellow", "\t", "column name must not be empty"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			addColor, addAfter = tc.color, ""
			err := addColumn(tc.col)
			if err == nil {
				t.Fatalf("addColumn(%q) with color %q returned nil error, want %q", tc.col, tc.color, tc.want)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("addColumn error = %q, want it to contain %q", err.Error(), tc.want)
			}
		})
	}
}

func TestRenameColumn_Guards(t *testing.T) {
	for _, newName := range []string{"", "   ", "\t"} {
		err := renameColumn("Grocery", newName)
		if err == nil {
			t.Fatalf("renameColumn(%q) returned nil error, want the empty-name error", newName)
		}
		if !strings.Contains(err.Error(), "new name must not be empty") {
			t.Errorf("renameColumn(%q) error = %q, want it to contain %q",
				newName, err.Error(), "new name must not be empty")
		}
	}
}
