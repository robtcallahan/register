package config

import (
	"encoding/json"
	"testing"
)

func TestConfig_IncomeSources(t *testing.T) {
	data := []byte(`{
		"income_sources": [
			{"name": "Taylor House Paycheck", "match": "TAYLOR HOUSE", "match_type": "substring", "frequency": "biweekly"},
			{"name": "Social Security", "match": "SSA", "frequency": "monthly"}
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
	if got.Name != "Taylor House Paycheck" || got.Match != "TAYLOR HOUSE" || got.MatchType != "substring" || got.Frequency != "biweekly" {
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
