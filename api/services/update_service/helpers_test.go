package update_service

import (
	"testing"
	"time"
)

func TestWeeksAgo(t *testing.T) {
	// Todo: startDate and endDate is currently set in update.go before fetchTransactions() due to pagination response from Plaid for > 100 xactions.
	// startDate := weeksAgo(12)

	got := weeksAgo(12)
	gotTime, err := time.Parse(PlaidDateFormat, got)
	if err != nil {
		t.Fatalf("weeksAgo(12) = %q, unparseable: %v", got, err)
	}

	// convert want time to a date string and back so that we can compare days without the time
	want := time.Now().AddDate(0, 0, -84)
	want, _ = time.Parse(PlaidDateFormat, want.Format(PlaidDateFormat))
	if diff := want.Sub(gotTime); diff > 24*time.Hour || diff < -24*time.Hour {
		t.Errorf("weeksAgo(12) = %q, want ~84 days ago", got)
	}
}

func TestToday(t *testing.T) {
	got := today()
	want := time.Now().Format(PlaidDateFormat)
	if got != want {
		t.Errorf("today() = %q, want %q", got, want)
	}

	// Verify the format is YYYY-MM-DD
	if len(got) != 10 || got[4] != '-' || got[7] != '-' {
		t.Errorf("today() = %q, want YYYY-MM-DD format", got)
	}
}

func TestDashes(t *testing.T) {
	tests := []struct {
		count int
		want  string
	}{
		{0, ""},
		{1, "-"},
		{5, "-----"},
		{10, "----------"},
	}

	for _, tt := range tests {
		if got := dashes(tt.count); got != tt.want {
			t.Errorf("dashes(%d) = %q, want %q", tt.count, got, tt.want)
		}
	}
}

func TestDaysAhead(t *testing.T) {
	for _, days := range []int{0, 7} {
		got := daysAhead(days)
		gotTime, err := time.Parse(PlaidDateFormat, got)
		if err != nil {
			t.Fatalf("daysAhead(%d) = %q, unparseable: %v", days, got, err)
		}
		want := time.Now().AddDate(0, 0, days)
		want, _ = time.Parse(PlaidDateFormat, want.Format(PlaidDateFormat))
		if diff := want.Sub(gotTime); diff > 24*time.Hour || diff < -24*time.Hour {
			t.Errorf("daysAhead(%d) = %q, want ~%d days ahead", days, got, days)
		}
		if len(got) != 10 || got[4] != '-' || got[7] != '-' {
			t.Errorf("daysAhead(%d) = %q, want YYYY-MM-DD format", days, got)
		}
	}
}
