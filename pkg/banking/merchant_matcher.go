package banking

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"register/pkg/models"
)

// merchantRule is one compiled matching rule from the merchants table.
type merchantRule struct {
	row     *models.DataRow
	pattern string         // upper-cased BankName, for substring/exact rules
	re      *regexp.Regexp // set only for regex rules
}

// MerchantMatcher matches bank transaction names against merchant rules
// deterministically: rules are tried highest-priority first (lowest merchant
// ID wins a tie) and the first match wins. It replaces the old pair of
// last-match-wins substring passes.
type MerchantMatcher struct {
	rules []merchantRule
}

// NewMerchantMatcher builds a matcher from lookup rows (see
// Query.GetLookupData). Regex rules are compiled once, up front; a bad
// pattern is an error.
func NewMerchantMatcher(rows []*models.DataRow) (*MerchantMatcher, error) {
	sorted := make([]*models.DataRow, len(rows))
	copy(sorted, rows)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].Priority != sorted[j].Priority {
			return sorted[i].Priority > sorted[j].Priority
		}
		return sorted[i].ID < sorted[j].ID
	})

	m := &MerchantMatcher{rules: make([]merchantRule, 0, len(sorted))}
	for _, r := range sorted {
		rule := merchantRule{row: r, pattern: strings.ToUpper(r.BankName)}
		if r.MatchType == "regex" {
			re, err := regexp.Compile(r.BankName)
			if err != nil {
				return nil, fmt.Errorf("merchant rule %q: bad regex: %w", r.BankName, err)
			}
			rule.re = re
		}
		m.rules = append(m.rules, rule)
	}
	return m, nil
}

// Match returns the winning rule for a bank name. Substring and exact rules
// fold case; regex rules see the name as-is (use (?i) in the pattern for
// case-insensitive regexes). The second return value is false when no rule
// matches.
func (m *MerchantMatcher) Match(bankName string) (*models.DataRow, bool) {
	upper := strings.ToUpper(bankName)
	for _, rule := range m.rules {
		switch rule.row.MatchType {
		case "exact":
			if upper == rule.pattern {
				return rule.row, true
			}
		case "regex":
			if rule.re.MatchString(bankName) {
				return rule.row, true
			}
		default: // "" (legacy rows) and "substring"
			if strings.Contains(upper, rule.pattern) {
				return rule.row, true
			}
		}
	}
	return nil, false
}
