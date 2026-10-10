package sheets_service

import "testing"

func Test_extendRangeEnd(t *testing.T) {
	tests := []struct {
		name    string
		formula string
		oldEnd  int
		newEnd  int
		want    string
		changed bool
	}{
		{"simple", "=SUM(D3:D8)", 8, 9, "=SUM(D3:D9)", true},
		{"already extended", "=SUM(D3:D9)", 8, 9, "=SUM(D3:D9)", false},
		{"absolute refs", "=SUM($D$3:$D$8)", 8, 9, "=SUM($D$3:$D$9)", true},
		{"two ranges", "=SUM(D3:D8)+SUM(E3:E8)", 8, 9, "=SUM(D3:D9)+SUM(E3:E9)", true},
		{"other row untouched", "=SUM(D3:D8)-D20", 8, 9, "=SUM(D3:D9)-D20", true},
		{"end below range not touched", "=AVERAGE(D10:D8)", 9, 10, "=AVERAGE(D10:D8)", false},
		{"no range", "=D8*2", 8, 9, "=D8*2", false},
		{"different end", "=SUM(D3:D7)", 8, 9, "=SUM(D3:D7)", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, changed := extendRangeEnd(tt.formula, tt.oldEnd, tt.newEnd)
			if got != tt.want || changed != tt.changed {
				t.Errorf("extendRangeEnd(%q, %d, %d) = (%q, %v), want (%q, %v)",
					tt.formula, tt.oldEnd, tt.newEnd, got, changed, tt.want, tt.changed)
			}
		})
	}
}

func Test_budgetGroupName(t *testing.T) {
	for color, want := range map[string]string{
		"green": "Discretionary", "yellow": "Non-Discretionary", "blue": "Savings Categories",
	} {
		if got, ok := budgetGroupName(color); !ok || got != want {
			t.Errorf("budgetGroupName(%q) = (%q, %v), want (%q, true)", color, got, ok, want)
		}
	}
	if _, ok := budgetGroupName("black"); ok {
		t.Error("budgetGroupName(black) should not map to a group")
	}
}
