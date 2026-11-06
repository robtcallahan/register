package repository

import (
	"testing"

	"register/pkg/config"
	"register/pkg/models"
)

func TestConnectUnreachable(t *testing.T) {
	// Port 1 on loopback refuses immediately, so no live database is
	// needed and the test does not wait on a dial timeout. gorm.Open
	// pings after opening, so an unreachable host surfaces here.
	_, err := Connect(&config.DBConfig{
		Username: "register",
		Password: "register",
		Host:     "127.0.0.1",
		Port:     "1",
		Name:     "register",
	})
	if err == nil {
		t.Fatal("Connect() to unreachable host err = nil; want non-nil")
	}
}

func TestColumnNames(t *testing.T) {
	columns := []models.Column{
		{Name: "Grocery"},
		{Name: "Dining Out"},
		{Name: "Emergency Fund"},
	}
	got := ColumnNames(columns)
	want := []string{"Grocery", "Dining Out", "Emergency Fund"}
	if len(got) != len(want) {
		t.Fatalf("ColumnNames() = %v, want %v", got, want)
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("ColumnNames()[%d] = %q, want %q", i, got[i], w)
		}
	}
	if got := ColumnNames(nil); len(got) != 0 {
		t.Errorf("ColumnNames(nil) len = %d, want 0", len(got))
	}
}
