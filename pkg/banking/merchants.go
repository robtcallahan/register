/*
Copyright © 2020 Rob Callahan <robtcallahan@aol.com>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package banking

import (
	"fmt"

	"register/pkg/models"
)

// ColumnIndexForName resolves a sheet column's position from its name, using
// the columns table contents. Returns 0 when the name is absent.
func ColumnIndexForName(columns []models.Column, name string) int {
	index, _ := models.NewColumnSet(columns).IndexFor(name)
	return index
}

// IncomeMatcher builds the matcher for the income sources configured in
// config.json (see config.IncomeSource). Source order in the config is the
// match order.
func (c *Client) IncomeMatcher() (*MerchantMatcher, error) {
	rows := make([]*models.DataRow, len(c.IncomeSources))
	for i, src := range c.IncomeSources {
		rows[i] = &models.DataRow{BankName: src.Match, Name: src.Name, MatchType: src.MatchType}
	}
	return NewMerchantMatcher(rows)
}

// FormatMerchantNames applies merchant rules to each transaction: a match
// sets the display name, color, and budget column. Rules come from the
// matcher (first-match-wins by priority). The CHECK and income specials
// resolve their column by lookup, not by hardcoded index: checks take the
// position of the "CHECK" merchant rule, income deposits the position of
// the "Emergency Fund" column. The income fan-out at sheet-write time is
// keyed by the configured display names, so this index only records where
// the deposit conceptually lands.
func (c *Client) FormatMerchantNames(trans []*models.Transaction, matcher *MerchantMatcher, income *MerchantMatcher, columns []models.Column) []*models.Transaction {
	for i, t := range trans {
		if t.Name == CheckTransactionName {
			trans[i].Color = models.ColorWhite
			if l, ok := matcher.FindByBankPattern(CheckBankPattern); ok {
				trans[i].ColumnIndex = l.ColumnIndex
			}
			trans[i].IsCategory = false
			trans[i].TaxDeductible = false
		} else if src, ok := income.Match(t.BankName); ok {
			trans[i].Name = src.Name
			trans[i].Color = models.ColorGreen
			trans[i].ColumnIndex = ColumnIndexForName(columns, IncomeColumnName)
			trans[i].IsCategory = false
			trans[i].TaxDeductible = false
		} else if l, ok := matcher.Match(t.BankName); ok {
			trans[i].Name = l.Name
			trans[i].Color = l.Color
			trans[i].ColumnIndex = l.ColumnIndex
			trans[i].IsCategory = l.IsCategory
			trans[i].TaxDeductible = l.TaxDeductible
		}
		if c.Debug {
			fmt.Printf("key: %s, name: %s, bankName: %s, amt: %.2f \n", trans[i].Key, trans[i].Name, trans[i].BankName, trans[i].Amount)
		}
	}
	return trans
}
