package config

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestConfig_IncomeSources(t *testing.T) {
	data := []byte(`{
		"income_sources": [
			{"name": "Taylor House Paycheck", "match_pattern": "TAYLOR HOUSE", "match_type": "substring", "frequency": "biweekly"},
			{"name": "Social Security", "match_pattern": "SSA", "frequency": "monthly"}
		]
	}`)

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatal(err)
	}
	if len(cfg.IncomeSources) != 2 {
		t.Fatalf("got %d income sources, want 2", len(cfg.IncomeSources))
	}
	got := cfg.IncomeSources[0]
	if got.Name != "Taylor House Paycheck" || got.MatchPattern != "TAYLOR HOUSE" || got.MatchType != "substring" || got.Frequency != "biweekly" {
		t.Errorf("first source parsed wrong: %+v", got)
	}
	if cfg.IncomeSources[1].MatchType != "" {
		t.Errorf("omitted match_type should stay empty (substring by default): %+v", cfg.IncomeSources[1])
	}
}

func TestConfig_NoIncomeSources(t *testing.T) {
	var cfg Config
	if err := json.Unmarshal([]byte(`{}`), &cfg); err != nil {
		t.Fatal(err)
	}
	if len(cfg.IncomeSources) != 0 {
		t.Errorf("missing income_sources should mean none configured, got %+v", cfg.IncomeSources)
	}
}

func TestReadConfig(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/config.json"
	data := []byte(`{
		"user_id": "user1",
		"banks": {
			"wellsfargo": {"id": "wellsfargo", "name": "Wells Fargo", "csv_file_name": "wf.csv"}
		},
		"income_sources": [
			{"name": "Taylor House", "match_pattern": "Winchester Eats PAYROLL", "frequency": "biweekly"}
		]
	}`)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := ReadConfig(path)
	if err != nil {
		t.Fatalf("ReadConfig = %v, want nil", err)
	}
	if cfg.UserID != "user1" {
		t.Errorf("UserID = %q, want user1", cfg.UserID)
	}
	bank, ok := cfg.Banks["wellsfargo"]
	if !ok || bank.Name != "Wells Fargo" || bank.CSVFileName != "wf.csv" {
		t.Errorf("Banks[wellsfargo] = %+v, want Wells Fargo entry", bank)
	}
	if len(cfg.IncomeSources) != 1 || cfg.IncomeSources[0].Name != "Taylor House" {
		t.Errorf("IncomeSources = %v, want one Taylor House entry", cfg.IncomeSources)
	}

	if _, err := ReadConfig(dir + "/nope.json"); err == nil {
		t.Error("ReadConfig(nonexistent) = nil, want error")
	} else if !strings.Contains(err.Error(), "could not read config file") {
		t.Errorf("ReadConfig(nonexistent) = %v, want read error", err)
	}

	bad := dir + "/bad.json"
	if err := os.WriteFile(bad, []byte(`{not json`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadConfig(bad); err == nil {
		t.Error("ReadConfig(invalid json) = nil, want error")
	} else if !strings.Contains(err.Error(), "could not unmarshal config file") {
		t.Errorf("ReadConfig(invalid json) = %v, want unmarshal error", err)
	}
}
