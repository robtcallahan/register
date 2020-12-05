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
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"register/pkg/config"
	"register/pkg/models"

	"github.com/plaid/plaid-go/plaid"
)

func New(o *ClientOptions) *Client {
	c := &Client{
		Keys: &Keys{
			Products:     "transactions",
			CountryCodes: "US",
		},
		StartDate: o.StartDate,
		EndDate:   o.EndDate,
		BankInfo:  o.BankInfo,
		Debug:     o.Debug,
		ClientID:  o.PlaidClientID,
		Secret:    o.PlaidSecret,
	}
	pc, _ := plaid.NewClient(plaid.ClientOptions{
		ClientID:    o.PlaidClientID,
		Secret:      o.PlaidSecret,
		Environment: plaid.Development,
		HTTPClient:  &http.Client{},
	})
	c.PlaidClient = pc
	return c
}

func (c *Client) SetBank(b config.BankInfo) {
	c.ItemID = b.PlaidItemID
	c.AccessToken = b.PlaidAccessToken
}

func (c *Client) GetAccount(cfg config.BankInfo, aType string) *plaid.Account {
	res, err := c.PlaidClient.GetAccounts(cfg.PlaidAccessToken)
	checkError(err)

	for i, a := range res.Accounts {
		if a.Type == aType {
			return &res.Accounts[i]
		}
	}
	return nil
}

func (c *Client) getPlaidTransactions(cfg config.BankInfo, start, end string) plaid.GetTransactionsResponse {
	res, err := c.PlaidClient.GetTransactionsWithOptions(c.AccessToken, plaid.GetTransactionsOptions{
		StartDate:  start,
		EndDate:    end,
		AccountIDs: []string{cfg.PlaidAccountID},
		Count:      50,
		Offset:     0,
	})
	checkError(err)
	return res
}

func (c *Client) GetTransactions() []*models.Transaction {
	var transactions []*models.Transaction
	for _, cfg := range c.BankInfo {
		fmt.Printf("    %s...", cfg.Name)

		c.SetBank(cfg)
		transResp := c.getPlaidTransactions(cfg, c.StartDate, c.EndDate)

		c.WriteCSV(cfg.FileName, transResp.Transactions)

		for _, t := range transResp.Transactions {
			if strings.Contains(strings.ToLower(t.Name), "payment thank you") {
				continue
			}
			transactions = append(transactions, c.createTransaction(cfg.Name, t))
		}
		fmt.Println("done")
	}
	return transactions
}

func (c *Client) createTransaction(bankName string, p plaid.Transaction) *models.Transaction {
	tran := &models.Transaction{
		Date:         readDateValue(p.Date),
		Name:         "",
		BankName:     p.Name,
		MerchantName: p.MerchantName,
	}

	switch bankName {
	case "Wells Fargo Checking":
		tran.Source = "WellsFargo"
		tran.Amount = p.Amount
		if p.Amount < 0 {
			tran.Deposit = -1 * p.Amount
		} else {
			tran.Withdrawal = p.Amount
		}
		tran.Key = fmt.Sprintf("%s:%s:%.2f", tran.Source, readDateValue(p.Date), -1*p.Amount)
	case "Fidelity Visa":
		tran.Source = "Fidelity"
		tran.Amount = p.Amount
		tran.CreditCard = p.Amount
		tran.Key = fmt.Sprintf("%s:%s:%.2f", tran.Source, readDateValue(p.Date), -1*p.Amount)
	case "Chase Visa":
		tran.Source = "Chase"
		tran.Amount = p.Amount
		tran.CreditCard = p.Amount
		tran.Key = fmt.Sprintf("%s:%s:%.2f", tran.Source, readDateValue(p.Date), -1*p.Amount)
	case "Citi Visa":
		tran.Source = "Citi"
		tran.Amount = p.Amount
		tran.CreditCard = p.Amount
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

func (c *Client) FormatMerchants(trans []*models.Transaction, lookup []*models.DataRow) []*models.Transaction {
	for i, t := range trans {
		if t.BankName == "Venmo" {
			if t.Amount == 150.00 {
				trans[i].Name = "Margie Knight (Venmo)"
				trans[i].Color = "blue"
				trans[i].ColumnIndex = 46
				trans[i].IsCategory = true
			} else if t.Amount == 5.00 || t.Amount == 10.00 {
				trans[i].Name = "AA Meeting (Venmo)"
				trans[i].Color = "blue"
				trans[i].ColumnIndex = 41
				trans[i].IsCategory = true
			}
		} else {
			for _, l := range lookup {
				if strings.Contains(t.BankName, l.BankName) {
					trans[i].Name = l.Name
					trans[i].Color = l.Color
					trans[i].ColumnIndex = l.ColumnIndex
					trans[i].IsCategory = l.IsCategory
				}
			}
		}
		if c.Debug {
			fmt.Printf("key: %s, name: %s, bankName: %s, amt: %.2f \n", trans[i].Key, trans[i].Name, trans[i].BankName, trans[i].Amount)
		}
	}
	return trans
}

func (c *Client) FilterRows(trans []*models.Transaction, lookup map[string]bool) []*models.Transaction {
	var filter []*models.Transaction
	for _, r := range trans {
		if _, ok := lookup[r.Key]; !ok {
			filter = append(filter, r)
		}
	}
	return filter
}

func (c *Client) createLinkToken() string {
	countryCodes := strings.Split(c.Keys.CountryCodes, ",")
	products := strings.Split(c.Keys.Products, ",")
	configs := plaid.LinkTokenConfigs{
		User: &plaid.LinkTokenUser{
			// This should correspond to a unique id for the current user.
			ClientUserID: "robtcallahan",
		},
		ClientName:        "Plaid Quickstart",
		Products:          products,
		CountryCodes:      countryCodes,
		Language:          "en",
		PaymentInitiation: nil,
	}
	resp, err := c.PlaidClient.CreateLinkToken(configs)
	checkError(err)
	return resp.LinkToken
}

func (c *Client) getLinkClient() (resp *plaid.LinkClientGetResponse) {
	res, err := c.PlaidClient.LinkClientGet(&plaid.LinkClientGetRequest{
		IntegrationMode:  1,
		LinkPersistentID: c.Link.PersistentID,
		LinkToken:        c.Link.Token,
		LinkVersion:      c.Link.Version,
	})
	checkError(err)
	fmt.Printf("res: %+v\n", res)
	return res
}

func (c *Client) linkItemCreate() *plaid.LinkItemCreateResponse {
	lic, err := ioutil.ReadFile("../json/link_item_create_dev.json")
	checkError(err)

	licStr := strings.Replace(string(lic), "LINK_TOKEN", c.Link.Token, 2)
	licStr = strings.Replace(licStr, "LINK_OPEN_ID", c.Link.OpenID, 1)
	licStr = strings.Replace(licStr, "LINK_PERSISTENT_ID", c.Link.PersistentID, 1)
	licStr = strings.Replace(licStr, "LINK_SESSION_ID", c.Link.SessionID, 1)

	res, err := c.PlaidClient.LinkItemCreate([]byte(licStr))
	fmt.Printf("res: %+v\n", res)
	checkError(err)
	fmt.Printf("res: %+v\n", res)
	return res
}

func (c *Client) getAccessToken() (string, string) {
	res, err := c.PlaidClient.ExchangePublicToken(c.PublicToken)
	checkError(err)
	return res.AccessToken, res.ItemID
}

func (c *Client) linkItemMFA() (resp *plaid.LinkItemMFAResponse) {
	resp, err := c.PlaidClient.LinkItemMFA(&plaid.LinkItemMFARequest{
		LinkToken:        c.Link.Token,
		LinkOpenID:       c.Link.OpenID,
		LinkPersistentID: c.Link.PersistentID,
		LinkSessionID:    c.Link.SessionID,
		MFAType:          "device_list",
		PublicToken:      c.PublicToken,
		DisplayLanguage:  "en",
		Responses:        []string{""},
	})
	checkError(err)
	return resp
}

func (c *Client) sendMFACode(code string) (resp *plaid.LinkItemMFASendCodeResponse) {
	resp, err := c.PlaidClient.LinkItemMFASendCode(&plaid.LinkItemMFARequest{
		DisplayLanguage:  "en",
		LinkOpenID:       c.Link.OpenID,
		LinkPersistentID: c.Link.PersistentID,
		LinkSessionID:    c.Link.SessionID,
		LinkToken:        c.Link.Token,
		MFAType:          "device",
		PublicToken:      c.PublicToken,
		Responses:        []string{code},
	})
	checkError(err)
	return resp
}

func (c *Client) GetAccounts() plaid.GetAccountsResponse {
	res, err := c.PlaidClient.GetAccounts(c.AccessToken)
	checkError(err)
	return res
}

func (c *Client) getCheckingID(accounts []plaid.Account) (checkingID string) {
	for _, acct := range accounts {
		if acct.Type == "depository" && acct.Subtype == "checking" {
			checkingID = acct.AccountID
		}
	}
	return checkingID
}

func (c *Client) printIdent() {
	fmt.Println("Ident:")
	fmt.Println("    LinkToken: ", c.Link.Token)
	fmt.Println("    PublicToken: " + c.PublicToken)
	fmt.Println("    RequestID: " + c.RequestID)
	fmt.Println("    AccessToken: " + c.AccessToken)
	fmt.Println("    ItemID: " + c.ItemID)
	fmt.Println("")
}

func (c *Client) WriteCSV(fileName string, trans []plaid.Transaction) {
	f, err := os.Create("csv/" + fileName)
	checkError(err)

	_, err = f.WriteString("Date,Amount,Description\n")
	checkError(err)
	for _, t := range trans {
		_, err = f.WriteString(fmt.Sprintf("%s,%.2f,%s,%s\n", t.Date, t.Amount, t.Name, t.MerchantName))
		checkError(err)
	}
	_ = f.Sync()
}

func readDateValue(date string) string {
	re := regexp.MustCompile(`(20)?(\d\d)-(\d\d)-(\d\d)`)
	m := re.FindAllStringSubmatch(date, -1)
	yy, _ := strconv.Atoi(m[0][2])
	mm, _ := strconv.Atoi(m[0][3])
	dd, _ := strconv.Atoi(m[0][4])
	d := fmt.Sprintf("%02d/%02d/%02d", mm, dd, yy)
	return d
}

func checkError(err error) {
	if err != nil {
		panic(err)
	}
}
