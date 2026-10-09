package models

import (
	"fmt"
	"strings"
)

// TransactionKey builds the register dedupe key (source:date:amount) for a
// transaction. The date must already be in MM/DD/YY form. Every producer and
// consumer of dedupe keys — Plaid, CSV import, and the register sheet itself —
// builds them here so the formats can't drift apart.
func TransactionKey(source, date string, amount float64) string {
	return fmt.Sprintf("%s:%s:%.2f", strings.ToLower(source), date, amount)
}
