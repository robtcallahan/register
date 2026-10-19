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
	"bytes"
	"context"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"

	"register/pkg/config"
	"register/pkg/models"

	"github.com/plaid/plaid-go/v15/plaid"
)

func (c *Client) GetTransactions(bankIDs []string, startDate, endDate string) ([]*models.Transaction, error) {
	var transactions []*models.Transaction
	var errors string
	for _, bankID := range bankIDs {
		bankConfig := c.Banks[bankID]
		fmt.Printf("    %s...", bankConfig.Name)

		transResp, err := c.getPlaidTransactions(bankConfig, startDate, endDate)
		if err != nil {
			errors += err.Error()
			continue
		}

		for _, trans := range transResp {
			// skip CC payment transaction as these will show up as checking account payments
			register, err := c.buildTransaction(bankConfig.ID, trans)
			if err != nil {
				return nil, err
			}
			transactions = append(transactions, register)
		}
		fmt.Println("done")
	}
	if errors != "" {
		return nil, fmt.Errorf("%s", errors)
	}
	return transactions, nil
}

func printPlaidTransaction(t plaid.Transaction, bankID string) {
	fmt.Printf("%-10s %-20s %8.2f\n", bankID, t.Name, t.Amount)
}

func printTransaction(t *models.Transaction) {
	fmt.Printf("[%-28s] %6.2f %6.2f %6.2f %6.2f %6.2f %6.2f\n", t.Key, t.Amount, t.Withdrawal, t.Deposit, t.CreditPurchase, t.Budget, t.CreditCard)
}

func (c *Client) getPlaidTransactions(bankConfig config.Bank, startDate, endDate string) ([]plaid.Transaction, error) {
	ctx := context.Background()

	request := plaid.NewTransactionsGetRequest(
		bankConfig.AccessToken,
		startDate,
		endDate,
	)
	options := plaid.TransactionsGetRequestOptions{
		Count:  plaid.PtrInt32(100),
		Offset: plaid.PtrInt32(0),
	}
	request.SetOptions(options)

	resp, httpResp, err := c.PlaidClient.PlaidApi.TransactionsGet(ctx).TransactionsGetRequest(*request).Execute()

	if err != nil {
		buf := new(bytes.Buffer)
		_, err2 := buf.ReadFrom(httpResp.Body)
		if err2 != nil {
			return nil, err2
		}
		return nil, fmt.Errorf("%w\n%s", err, buf.String())
	}
	return resp.Transactions, nil

	//options := c.PlaidClient.PlaidApi.TransactionsGet(ctx).NewTransactionsGetRequestOptions(
	//	StartDate: startDate,
	//	EndDate:   endDate
	//)
	//
	//resp, httpResp, err := c.PlaidClient.PlaidApi.TransactionsGet(ctx).TransactionsGetRequest(plaid.TransactionsGetRequest{
	//	ClientId:    &c.ClientID,
	//	AccessToken: bankConfig.AccessToken,
	//	Secret:      &c.Secret,
	//	Options: &plaid.TransactionsGetRequestOptions{
	//		AccountIds: &[]string{bankConfig.AccountID},
	//		StartDate:  startDate,
	//		EndDate:    endDate,
	//	},
	//}).Execute()
	//if err != nil {
	//	buf := new(bytes.Buffer)
	//	_, err2 := buf.ReadFrom(httpResp.Body)
	//	if err2 != nil {
	//		return nil, err2
	//	}
	//	return nil, fmt.Errorf("%s\n%s", err.Error(), buf.String())
	//}
	//return resp.Transactions, nil
}

func (c *Client) buildTransaction(bankID string, p plaid.Transaction) (*models.Transaction, error) {
	tran := &models.Transaction{
		Date:     readDateValue(p.Date),
		Name:     "",
		BankName: p.Name,
	}

	switch bankID {
	case WellsFargoID:
		if p.CheckNumber.IsSet() {
			tran.Source = *p.CheckNumber.Get()
			tran.IsCheck = true
			tran.Name = "CHECK"
			tran.BankName = "CHECK"
		} else {
			tran.Source = "WellsFargo"
			if tran.Name == "" {
				tran.Name = tran.BankName
			}
		}

		tran.Amount = p.Amount
		if p.Amount < 0 {
			tran.Deposit = -1 * p.Amount // covert to positive
			tran.Key = models.TransactionKey(tran.Source, readDateValue(p.Date), tran.Deposit)
		} else {
			tran.Withdrawal = p.Amount
			tran.Key = models.TransactionKey(tran.Source, readDateValue(p.Date), tran.Withdrawal)
		}
		tran.Budget = -1 * p.Amount
	case FidelityID:
		tran.Source = "Fidelity"
		tran.Amount = p.Amount         // amount stays as is (positive)
		cc := p.Amount                 // keep positive
		tran.CreditPurchase = p.Amount // keep positive
		tran.CreditCard = p.Amount     // keep positive
		tran.Budget = -1 * p.Amount    // budget category column negative
		tran.Key = models.TransactionKey(tran.Source, tran.Date, cc)
		if tran.Name == "" {
			tran.Name = tran.BankName
		}
	case ChaseID:
		tran.Source = "Chase"
		tran.Amount = p.Amount         // amount stays as is (positive)
		cc := p.Amount                 // keep positive
		tran.CreditPurchase = p.Amount // keep positive
		tran.CreditCard = p.Amount     // keep positive
		tran.Budget = -1 * p.Amount    // budget category column negative
		tran.Key = models.TransactionKey(tran.Source, tran.Date, cc)
		if tran.Name == "" {
			tran.Name = tran.BankName
		}
	default:
		return nil, fmt.Errorf("unknown bank ID: %s", bankID)
	}
	return tran, nil
}

func (c *Client) SortTransactions(trans []*models.Transaction) []*models.Transaction {
	sort.Slice(trans, func(i, j int) bool {
		if trans[i].Date == trans[j].Date {
			return trans[i].Name < trans[j].Name
		}
		return trans[i].Date < trans[j].Date
	})
	return trans
}

func (c *Client) PrintTransactionHead() {
	fmt.Printf("    [Num] %-25s %-32s %-40s %-40s %12s %12s %12s %12s %7s %5s\n",
		"Key", "Name", "Bank Name", "Merchant Name", "Withdrawal", "Deposit", "Credit Card", "Amount", "ColIndx", "Color")
}

// FilterRecordedTransactions removes transactions that are already contained in the register spreadsheet
func (c *Client) FilterRecordedTransactions(trans []*models.Transaction, regLookup map[string]bool) []*models.Transaction {
	var filtered []*models.Transaction
	i := 0
	for _, t := range trans {
		if _, ok := regLookup[t.Key]; !ok {
			filtered = append(filtered, t)
			i++
			if c.Debug {
				fmt.Printf("    (%2d) NEW [%-28s] %-12s %-10s %8.2f %s\n", i, t.Key, t.Source, t.Date, t.Amount, t.Name)
			}
		}
	}
	return filtered
}

// creditCardPaymentPatterns identify card-payment transactions. Those
// payments also appear as checking-account transactions, so keeping both
// would double-count them.
var creditCardPaymentPatterns = []*regexp.Regexp{
	regexp.MustCompile(`PAYMENT MADE BY ACCOUNT ENDING IN:5409`),
	regexp.MustCompile(`Payment Thank You - Bill`),
	regexp.MustCompile(`Payment Thank You Bill`),
}

// FilterCreditCardPayments drops credit card payment transactions.
func (c *Client) FilterCreditCardPayments(trans []*models.Transaction) []*models.Transaction {
	var newTrans []*models.Transaction

	for _, t := range trans {
		drop := false
		for _, re := range creditCardPaymentPatterns {
			if re.MatchString(t.BankName) {
				drop = true
				break
			}
		}
		if !drop {
			newTrans = append(newTrans, t)
		}
	}
	return newTrans
}

//func (c *Client) getCheckingID(accounts []plaid.Account) (checkingID string) {
//	for _, acct := range accounts {
//		if acct.Type == "depository" && acct.Subtype == "checking" {
//			checkingID = acct.AccountID
//		}
//	}
//	return checkingID
//}

func (c *Client) WriteCSV(fileName string, trans []plaid.Transaction) error {
	f, err := os.Create("csv/" + fileName)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString("Date,Amount,Description\n")
	if err != nil {
		return err
	}
	for _, t := range trans {
		_, err = f.WriteString(fmt.Sprintf("%s,%.2f,%s,%v\n", t.Date, t.Amount, t.Name, t.MerchantName))
		if err != nil {
			return err
		}
	}
	return f.Sync()
}

func readDateValue(date string) string {
	re := regexp.MustCompile(`(20)?(\d\d)-(\d\d)-(\d\d)`)
	m := re.FindStringSubmatch(date)
	if m == nil {
		return ""
	}
	yy, _ := strconv.Atoi(m[2])
	mm, _ := strconv.Atoi(m[3])
	dd, _ := strconv.Atoi(m[4])
	d := fmt.Sprintf("%02d/%02d/%02d", mm, dd, yy)
	return d
}
