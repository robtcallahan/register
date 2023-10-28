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
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/plaid/plaid-go/v15/plaid"
	"golang.org/x/net/context"
	"register/pkg/config"
	"register/pkg/models"
)

const (
	WellsFargoID    = "wellsfargo"
	FidelityID      = "fidelity"
	ChaseID         = "chase"
	BankOfAmericaID = "boa"
	CitiID          = "citi"
	AllyID          = "ally"
	ETradeID        = "etrade"
	BettermentID    = "betterment"
)

type PlaidHttpBodyResponse struct {
	DisplayMessage   plaid.NullableString `json:"display_message"`
	DocumentationUrl string               `json:"documentation_url"`
	ErrorCode        string               `json:"error_code"`
	ErrorMessage     string               `json:"error_message"`
	ErrorType        string               `json:"error_type"`
	RequestId        string               `json:"request_id"`
	SuggestedAction  plaid.NullableString `json:"suggested_action"`
}

type ClientOptions struct {
	PlaidClientID    string
	PlaidSecret      string
	PlaidEnvironment plaid.Environment
	PlaidTokensDir   string
	UserID           string
	Banks            map[string]config.Bank
	Debug            bool
	Verbose          bool
}

type Client struct {
	PlaidClient *plaid.APIClient
	ClientID    string
	Secret      string
	Environment plaid.Environment
	TokensDir   string
	UserID      string
	Banks       map[string]config.Bank
	Debug       bool
	Verbose     bool
}

type Balance struct {
	BankName string
	Amount   float64
	Error    error
}

func NewClient(o *ClientOptions) *Client {
	c := &Client{
		ClientID:    o.PlaidClientID,
		Secret:      o.PlaidSecret,
		Environment: o.PlaidEnvironment,
		TokensDir:   o.PlaidTokensDir,
		UserID:      o.UserID,
		Banks:       o.Banks,
		Debug:       o.Debug,
		Verbose:     o.Debug,
	}
	configuration := plaid.NewConfiguration()
	configuration.AddDefaultHeader("PLAID-CLIENT-ID", o.PlaidClientID)
	configuration.AddDefaultHeader("PLAID-SECRET", o.PlaidSecret)
	configuration.UseEnvironment(o.PlaidEnvironment)
	client := plaid.NewAPIClient(configuration)
	c.PlaidClient = client
	return c
}

func (c *Client) GetAccounts(accessToken string, ctx context.Context) ([]plaid.AccountBase, error) {
	accountsGetRequest := plaid.NewAccountsGetRequest(accessToken)
	accountsGetResponse, httpResponse, err := c.PlaidClient.PlaidApi.AccountsGet(ctx).AccountsGetRequest(*accountsGetRequest).Execute()

	if err != nil {
		buf := new(bytes.Buffer)
		_, err2 := buf.ReadFrom(httpResponse.Body)
		if err2 != nil {
			return nil, err2
		}
		return nil, fmt.Errorf("%s\n: %s", err.Error(), buf.Bytes())
	}
	return accountsGetResponse.GetAccounts(), err
}

func (c *Client) GetBalances(bankIDs []string) map[string]Balance {
	ctx := context.Background()
	balances := make(map[string]Balance, 4)
	balance := Balance{}

	for _, id := range bankIDs {
		bank := c.Banks[id]

		tok, err := os.ReadFile(c.TokensDir + "/" + bank.Source + "AccessToken.txt")
		checkError(err)
		accessToken := strings.ReplaceAll(string(tok), "\n", "")

		amount, err := c.GetBalance(accessToken, id, ctx)
		if err != nil {
			balance = Balance{
				BankName: bank.Name,
				Error:    err,
			}
		} else {
			balance = Balance{
				BankName: bank.Name,
				Amount:   *amount,
			}
		}
		balances[id] = balance
	}
	return balances
}

func (c *Client) GetBalance(accessToken string, bankID string, ctx context.Context) (*float64, error) {
	balancesGetReq := plaid.NewAccountsBalanceGetRequest(accessToken)
	balancesGetReq.SetOptions(plaid.AccountsBalanceGetRequestOptions{
		AccountIds: &[]string{c.Banks[bankID].AccountID},
	})

	balancesGetResp, httpResponse, err := c.PlaidClient.PlaidApi.AccountsBalanceGet(ctx).AccountsBalanceGetRequest(
		*balancesGetReq,
	).Execute()
	if err != nil {
		buf := new(bytes.Buffer)
		_, err2 := buf.ReadFrom(httpResponse.Body)
		if err2 != nil {
			return nil, err2
		}
		return nil, fmt.Errorf("error with %s\n%s: %s", bankID, err.Error(), buf.Bytes())
	}
	nullFloat64 := balancesGetResp.Accounts[0].Balances.Current
	if !nullFloat64.IsSet() {
		return nil, fmt.Errorf("available balance is not set")
	}
	return nullFloat64.Get(), err
}

func (c *Client) GetBankStatus(bankID string) (*plaid.Institution, error) {
	var ctx context.Context
	bankConfig := c.Banks[bankID]
	req := plaid.InstitutionsGetByIdRequest{
		InstitutionId: bankConfig.Institution,
		ClientId:      &c.ClientID,
		Secret:        &c.Secret,
		CountryCodes:  []plaid.CountryCode{plaid.COUNTRYCODE_US},
	}
	resp, httpResp, err := c.PlaidClient.PlaidApi.InstitutionsGetById(ctx).InstitutionsGetByIdRequest(req).Execute()
	if err != nil {
		buf := new(bytes.Buffer)
		_, err2 := buf.ReadFrom(httpResp.Body)
		if err2 != nil {
			return nil, err2
		}
		return nil, fmt.Errorf("%s\n%s", err.Error(), buf.String())
	}
	return &resp.Institution, nil
}

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
			//re := regexp.MustCompile(`(payment)\s?-?\s?(thank you)?`)
			re := regexp.MustCompile(`.*(thank you).*`)
			m := re.FindStringSubmatch(strings.ToLower(trans.Name))
			if len(m) > 0 {
				continue
			}
			register := c.buildTransaction(bankConfig.ID, trans)
			transactions = append(transactions, register)
			//printPlaidTransaction(t, bankID)
			//printTransaction(r)
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
	var count int32 = 50
	var offset int32 = 0
	resp, httpResp, err := c.PlaidClient.PlaidApi.TransactionsGet(ctx).TransactionsGetRequest(plaid.TransactionsGetRequest{
		ClientId:    &c.ClientID,
		AccessToken: bankConfig.AccessToken,
		Secret:      &c.Secret,
		StartDate:   startDate,
		EndDate:     endDate,
		Options: &plaid.TransactionsGetRequestOptions{
			AccountIds: &[]string{bankConfig.AccountID},
			Count:      &count,
			Offset:     &offset,
		},
	}).Execute()
	if err != nil {
		buf := new(bytes.Buffer)
		_, err2 := buf.ReadFrom(httpResp.Body)
		if err2 != nil {
			return nil, err2
		}
		return nil, fmt.Errorf("%s\n%s", err.Error(), buf.String())
	}
	return resp.Transactions, nil
}

func (c *Client) buildTransaction(bankID string, p plaid.Transaction) *models.Transaction {
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
		}

		tran.Amount = p.Amount
		if p.Amount < 0 {
			tran.Deposit = -1 * p.Amount // covert to positive
		} else {
			tran.Withdrawal = p.Amount
		}
		tran.Budget = -1 * p.Amount
		tran.Key = fmt.Sprintf("%s:%s:%.2f", tran.Source, readDateValue(p.Date), -1*p.Amount)
	case FidelityID:
		tran.Source = "Fidelity"
		tran.Amount = p.Amount         // amount stays as is
		tran.CreditCard = p.Amount     // keep positive
		tran.CreditPurchase = p.Amount // keep positive
		tran.Budget = -1 * p.Amount    // budget category column negative
		tran.Key = fmt.Sprintf("%s:%s:%.2f", tran.Source, readDateValue(p.Date), -1*p.Amount)
	case ChaseID:
		tran.Source = "Chase"
		tran.Amount = p.Amount         // amount stays as is
		tran.CreditCard = p.Amount     // keep positive
		tran.CreditPurchase = p.Amount // keep positive
		tran.Budget = -1 * p.Amount    // budget category column negative
		tran.Key = fmt.Sprintf("%s:%s:%.2f", tran.Source, readDateValue(p.Date), -1*p.Amount)
	}
	return tran
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

// TODO: add substring check

func (c *Client) FormatMerchantNames(trans []*models.Transaction, lookup []*models.DataRow) []*models.Transaction {
	for i, t := range trans {
		if t.Name == "CHECK" {
			trans[i].Color = "white"
			trans[i].ColumnIndex = 10
			trans[i].IsCategory = false
			trans[i].TaxDeductible = false
			continue
			//} else if t.BankName == "Venmo" {
			//	if t.Amount == 150.00 {
			//		trans[i].Name = "Margie Knight (Venmo)"
			//		trans[i].Color = "blue"
			//		// this index is not used. Refer to the merchants table instead
			//		//trans[i].ColumnIndex = 41
			//		trans[i].IsCategory = true
			//	}
		} else {
			for _, l := range lookup {
				if strings.Contains(strings.ToUpper(t.BankName), strings.ToUpper(l.BankName)) {
					trans[i].Name = l.Name
					trans[i].Color = l.Color
					trans[i].ColumnIndex = l.ColumnIndex
					trans[i].IsCategory = l.IsCategory
					trans[i].TaxDeductible = l.TaxDeductible

					//if l.Name == "CrowdStrike Salary" && t.Amount > -3000 {
					//	trans[i].Name = "CrowdStrike Bonus"
					//}
				}
			}
		}
		if c.Debug {
			fmt.Printf("key: %s, name: %s, bankName: %s, amt: %.2f \n", trans[i].Key, trans[i].Name, trans[i].BankName, trans[i].Amount)
		}
	}
	return trans
}

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

//func (c *Client) getCheckingID(accounts []plaid.Account) (checkingID string) {
//	for _, acct := range accounts {
//		if acct.Type == "depository" && acct.Subtype == "checking" {
//			checkingID = acct.AccountID
//		}
//	}
//	return checkingID
//}

func (c *Client) WriteCSV(fileName string, trans []plaid.Transaction) {
	f, err := os.Create("csv/" + fileName)
	checkError(err)

	_, err = f.WriteString("Date,Amount,Description\n")
	checkError(err)
	for _, t := range trans {
		_, err = f.WriteString(fmt.Sprintf("%s,%.2f,%s,%v\n", t.Date, t.Amount, t.Name, t.MerchantName))
		checkError(err)
	}
	_ = f.Sync()
}

func readDateValue(date string) string {
	re := regexp.MustCompile(`(20)?(\d\d)-(\d\d)-(\d\d)`)
	m := re.FindStringSubmatch(date)
	yy, _ := strconv.Atoi(m[2])
	mm, _ := strconv.Atoi(m[3])
	dd, _ := strconv.Atoi(m[4])
	d := fmt.Sprintf("%02d/%02d/%02d", mm, dd, yy)
	return d
}

func checkError(err error) {
	if err != nil {
		panic(err)
	}
}
