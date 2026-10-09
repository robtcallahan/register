package cmd

import (
	"testing"

	"register/pkg/banking"
	"register/pkg/models"
)

func TestDedupeAgainstSheet(t *testing.T) {
	client := &Client{BankClient: &banking.Client{}}
	recorded := map[string]bool{
		"amazon:12/31/25:25.00":     true,
		"wellsfargo:01/05/26:100.00": true,
	}
	trans := []*models.Transaction{
		{Key: "amazon:01/01/26:25.00"},      // +1 day shift: duplicate
		{Key: "amazon:12/30/25:25.00"},      // -1 day shift: duplicate
		{Key: "wellsfargo:01/05/26:100.00"}, // exact match: duplicate
		{Key: "amazon:01/01/26:30.00"},      // different amount: new
		{Key: "chase:01/01/26:25.00"},       // different source: new
	}

	got := dedupeAgainstSheet(client, trans, recorded)

	if len(got) != 2 {
		t.Fatalf("expected 2 new transactions, got %d", len(got))
	}
	if got[0].Key != "amazon:01/01/26:30.00" || got[1].Key != "chase:01/01/26:25.00" {
		t.Errorf("unexpected survivors: %s, %s", got[0].Key, got[1].Key)
	}
}

func TestFuzzyKeySet(t *testing.T) {
	keys := map[string]bool{
		"amazon:12/31/25:25.00": true,
		"badkey":                true,
		"chase:01/01/26:xyz":    true, // amount is opaque; date still shifts
	}
	fuzzy := fuzzyKeySet(keys)

	for _, want := range []string{
		"amazon:12/30/25:25.00",
		"amazon:12/31/25:25.00",
		"amazon:01/01/26:25.00", // year boundary
		"badkey",
		"chase:12/31/25:xyz",
		"chase:01/02/26:xyz",
	} {
		if !fuzzy[want] {
			t.Errorf("missing variant %q", want)
		}
	}
	if fuzzy["amazon:01/02/26:25.00"] {
		t.Error("did not expect a +2 day variant")
	}
}
