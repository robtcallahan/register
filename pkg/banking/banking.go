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
	"bufio"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"register/pkg/config"
	repo "register/pkg/repository"

	"github.com/plaid/plaid-go/plaid"
)

// Keys ...
type Keys struct {
	Products     string
	CountryCodes string
}

// Link ...
type Link struct {
	Version      string
	Token        string
	OpenID       string
	PersistentID string
	SessionID    string
}

// ClientOptions ...
type ClientOptions struct {
	Keys          *Keys
	StartDate     string
	EndDate       string
	BankInfo      map[string]config.BankInfo
	PlaidClientID string
	PlaidSecret   string
	Merchants     map[string]string
	Debug         bool
}

// Client ...
type Client struct {
	Keys        *Keys
	Link        *Link
	AccessToken string
	PublicToken string
	ItemID      string
	RequestID   string
	PlaidClient *plaid.Client
	StartDate   string
	EndDate     string
	BankInfo    map[string]config.BankInfo
	ClientID    string
	Secret      string
	Debug       bool
}

// Transaction ...
type Transaction struct {
	Key          string
	Source       string
	Date         string
	Name         string
	BankName     string
	MerchantName string
	Amount       float64
	Withdrawal   float64
	Deposit      float64
	CreditCard   float64
	ColumnIndex  int
	Color        string
	IsCategory   bool
}

// New ...
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

// SetBank ...
func (c *Client) SetBank(b config.BankInfo) {
	c.ItemID = b.PlaidItemID
	c.AccessToken = b.PlaidAccessToken
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

// GetTransactions ...
func (c *Client) GetTransactions() []*Transaction {
	transactions := []*Transaction{}
	i := 1
	for _, cfg := range c.BankInfo {
		if i == 1 {
			i++
			continue
		}
		fmt.Printf("    %s...", cfg.Name)

		c.SetBank(cfg)
		transResp := c.getPlaidTransactions(cfg, c.StartDate, c.EndDate)

		c.WriteCSV(cfg.FileName, transResp.Transactions)

		for _, t := range transResp.Transactions {
			tran := &Transaction{
				Date:         formatDate(t.Date),
				Name:         "",
				BankName:     t.Name,
				MerchantName: t.MerchantName,
			}

			switch cfg.Name {
			case "Wells Fargo Checking":
				tran.Source = "-"
				tran.Amount = t.Amount
				if t.Amount < 0 {
					tran.Deposit = -1 * t.Amount
				} else {
					tran.Withdrawal = t.Amount
				}
				tran.Key = fmt.Sprintf("-:%s:%.2f", formatDate(t.Date), -1*t.Amount)
			case "Fidelity Visa":
				tran.Source = "VISA"
				tran.Amount = t.Amount
				tran.CreditCard = t.Amount
				tran.Key = fmt.Sprintf("%s:%s:%.2f", tran.Source, formatDate(t.Date), -1*t.Amount)
			case "Chase Visa":
				tran.Source = "CHASE"
				tran.Amount = t.Amount
				tran.CreditCard = t.Amount
				tran.Key = fmt.Sprintf("%s:%s:%.2f", tran.Source, formatDate(t.Date), -1*t.Amount)
			case "Citi Visa":
				tran.Source = "CITI"
				tran.Amount = t.Amount
				tran.CreditCard = t.Amount
				tran.Key = fmt.Sprintf("%s:%s:%.2f", tran.Source, formatDate(t.Date), -1*t.Amount)
			}
			transactions = append(transactions, tran)
		}
		fmt.Println("done")
	}
	return transactions
}

// SortTransactions ...
func (c *Client) SortTransactions(trans []*Transaction) []*Transaction {
	sort.Slice(trans, func(i, j int) bool {
		if trans[i].Date == trans[j].Date {
			return trans[i].Name < trans[j].Name
		}
		return trans[i].Date < trans[j].Date
	})
	return trans
}

// PrintTransactionHead ...
func (c *Client) PrintTransactionHead() {
	fmt.Printf("    [Num] %-25s %-32s %-40s %-40s %12s %12s %12s %12s %7s %5s\n",
		"Key", "Name", "Bank Name", "Merchant Name", "Withdrawal", "Deposit", "Credit Card", "Amount", "ColIndx", "Color")
}

// PrintTransaction ...
func (t *Transaction) PrintTransaction(n int) {
	fmt.Printf("    [%3d] %-25s %-32s %-40s %-40s %12.2f %12.2f %12.2f %12.2f %7d %-5s\n",
		n+1, t.Key, t.Name, t.BankName, t.MerchantName, t.Withdrawal, t.Deposit, t.CreditCard, t.Amount, t.ColumnIndex, t.Color)
}

// FormatMerchants ...
func (c *Client) FormatMerchants(trans []*Transaction, lookup []*repo.DataRow) []*Transaction {
	for i, t := range trans {
		// if t.Name == "Venmo" && t.Amount == 150.00 {
		// 	trans[i].BankName = "Margie Knight (Venmo)"
		// 	trans[i].ColumnIndex = 46
		// 	trans[i].Color = "blue"
		// } else if t.BankName == "Venmo" && t.Amount == 5.00 {
		// 	trans[i].Name = "AA Meeting (Venmo)"
		// 	trans[i].ColumnIndex = 41
		// 	trans[i].Color = "blue"
		// }

		for _, l := range lookup {
			if strings.Contains(t.BankName, l.BankName) {
				trans[i].Name = l.Name
				trans[i].Color = l.Color
				trans[i].ColumnIndex = l.ColumnIndex
				trans[i].IsCategory = l.IsCategory
			}
		}
	}
	return trans
}

// FilterRows ...
func (c *Client) FilterRows(trans []*Transaction, lookup map[string]bool) []*Transaction {
	filter := []*Transaction{}
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
	licStr = strings.Replace(string(licStr), "LINK_OPEN_ID", c.Link.OpenID, 1)
	licStr = strings.Replace(string(licStr), "LINK_PERSISTENT_ID", c.Link.PersistentID, 1)
	licStr = strings.Replace(string(licStr), "LINK_SESSION_ID", c.Link.SessionID, 1)

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

// GetAccounts ...
func (c *Client) GetAccounts() plaid.GetAccountsResponse {
	res, err := c.PlaidClient.GetAccounts(c.AccessToken)
	checkError(err)
	return res
}

func (c *Client) getCheckingID(accts []plaid.Account) (checkingID string) {
	for _, acct := range accts {
		if acct.Type == "depository" && acct.Subtype == "checking" {
			checkingID = acct.AccountID
		}
	}
	return checkingID
}

func getCode() string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter Code> ")
	text, _ := reader.ReadString('\n')
	text = strings.Replace(text, "\n", "", -1)
	return text
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

func printAccounts(accts []plaid.Account) {
	for i, acct := range accts {
		fmt.Printf("[%d] %s %s %s %s %s\n", i, acct.AccountID, acct.Name, acct.OfficialName, acct.Type, acct.Subtype)
	}
	fmt.Println("")
}

// WriteCSV ...
func (c *Client) WriteCSV(fileName string, trans []plaid.Transaction) {
	f, err := os.Create("csv/" + fileName)
	checkError(err)

	_, err = f.WriteString("Date,Amount,Description\n")
	checkError(err)
	for _, t := range trans {
		_, err = f.WriteString(fmt.Sprintf("%s,%.2f,%s\n", t.Date, t.Amount, t.Name))
		checkError(err)
	}
	f.Sync()
}

func formatDate(date string) string {
	re := regexp.MustCompile(`(20)?(\d\d)-(\d\d)-(\d\d)`)
	m := re.FindAllStringSubmatch(date, -1)
	yy, _ := strconv.Atoi(m[0][2])
	mm, _ := strconv.Atoi(m[0][3])
	dd, _ := strconv.Atoi(m[0][4])
	d := fmt.Sprintf("%02d/%02d/%02d", mm, dd, yy)
	return d
}

func printTrans(trans []plaid.Transaction) {
	for _, t := range trans {
		fmt.Printf("%s,%.2f,%s\n", t.Date, -t.Amount, t.Name)
	}
}

func checkError(err error) {
	if err != nil {
		panic(err)
	}
}

// client.Link.Token = client.createLinkToken()
// resp := client.getLinkClient()
// client.Link.SessionID = resp.LinkSessionID
// client.RequestID = resp.RequestID
// resp2 := client.linkItemCreate()
// client.PublicToken = resp2.PublicToken
// client.RequestID = resp2.RequestID
// if len(resp2.DeviceList) > 0 {
//     mfaReqResp := client.linkItemMFA()
//     fmt.Printf("MFA req resp msg: %s\n", mfaReqResp.Device.DisplayMessage)
//     code := getCode()
//     sentCodeResp := client.sendMFACode(code)
//     client.AccessToken, client.ItemID = client.getAccessToken()
//     client.printIdent()
//     acctsResp := client.getAccounts()
//     printAccounts(acctsResp.Accounts)
//     checkingID := client.getCheckingID(acctsResp.Accounts)
//     transResp := client.getTransactions(checkingID)
//     writeCSV(transResp.Transactions)
// }
