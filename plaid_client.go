package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/plaid/plaid-go/plaid"
)

// PlaidKeys ...
type PlaidKeys struct {
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

// Client ...
type Client struct {
	PlaidKeys   *PlaidKeys
	Link        *Link
	AccessToken string
	PublicToken string
	ItemID      string
	RequestID   string
	PlaidClient *plaid.Client
}

// client.Link.Token = client.createLinkToken()
// resp := client.getLinkClient()
// client.Link.SessionID = resp.LinkSessionID
// client.RequestID = resp.RequestID

// resp2 := client.linkItemCreate()
// client.PublicToken = resp2.PublicToken
// client.RequestID = resp2.RequestID

// if len(resp2.DeviceList) > 0 {
// 	mfaReqResp := client.linkItemMFA()
// 	fmt.Printf("MFA req resp msg: %s\n", mfaReqResp.Device.DisplayMessage)

// 	code := getCode()
// 	sentCodeResp := client.sendMFACode(code)

// client.AccessToken, client.ItemID = client.getAccessToken()
// client.printIdent()

// acctsResp := client.getAccounts()
// printAccounts(acctsResp.Accounts)
// checkingID := client.getCheckingID(acctsResp.Accounts)

// transResp := client.getTransactions(checkingID)
// writeCSV(transResp.Transactions)
// }

// -------------------------------------------------------------------------------------
// -------------------------------------------------------------------------------------

func (c *Client) setBank(b BankInfo) {
	c.ItemID = b.PlaidItemID
	c.AccessToken = b.PlaidAccessToken
}

func (c *Client) createLinkToken() string {
	countryCodes := strings.Split(c.PlaidKeys.CountryCodes, ",")
	products := strings.Split(c.PlaidKeys.Products, ",")
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
	checkErr(err)
	return resp.LinkToken
}

func (c *Client) getLinkClient() (resp *plaid.LinkClientGetResponse) {
	res, err := c.PlaidClient.LinkClientGet(&plaid.LinkClientGetRequest{
		IntegrationMode:  1,
		LinkPersistentID: c.Link.PersistentID,
		LinkToken:        c.Link.Token,
		LinkVersion:      c.Link.Version,
	})
	checkErr(err)
	fmt.Printf("res: %+v\n", res)
	return res
}

func (c *Client) linkItemCreate() *plaid.LinkItemCreateResponse {
	lic, err := ioutil.ReadFile("../json/link_item_create_dev.json")
	checkErr(err)

	licStr := strings.Replace(string(lic), "LINK_TOKEN", c.Link.Token, 2)
	licStr = strings.Replace(string(licStr), "LINK_OPEN_ID", c.Link.OpenID, 1)
	licStr = strings.Replace(string(licStr), "LINK_PERSISTENT_ID", c.Link.PersistentID, 1)
	licStr = strings.Replace(string(licStr), "LINK_SESSION_ID", c.Link.SessionID, 1)

	res, err := c.PlaidClient.LinkItemCreate([]byte(licStr))
	fmt.Printf("res: %+v\n", res)
	checkErr(err)
	fmt.Printf("res: %+v\n", res)
	return res
}

func (c *Client) getAccessToken() (string, string) {
	res, err := c.PlaidClient.ExchangePublicToken(c.PublicToken)
	checkErr(err)
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
	checkErr(err)
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
	checkErr(err)
	return resp
}

func (c *Client) getAccounts() plaid.GetAccountsResponse {
	res, err := c.PlaidClient.GetAccounts(c.AccessToken)
	checkErr(err)
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

func (c *Client) getTransactions(cfg BankInfo, start, end string) plaid.GetTransactionsResponse {
	res, err := c.PlaidClient.GetTransactionsWithOptions(c.AccessToken, plaid.GetTransactionsOptions{
		StartDate:  start,
		EndDate:    end,
		AccountIDs: []string{cfg.PlaidAccountID},
		Count:      50,
		Offset:     0,
	})
	checkErr(err)

	// Chase bank returns positive values for credit card charges. WTF?
	if cfg.ID == "chase" {
		for i := range res.Transactions {
			res.Transactions[i].Amount *= -1
		}
	}
	return res
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

func writeCSV(fileName string, trans []plaid.Transaction) {
	f, err := os.Create(fileName)
	checkErr(err)

	_, err = f.WriteString("Date,Amount,Description\n")
	checkErr(err)
	for _, t := range trans {
		_, err = f.WriteString(fmt.Sprintf("%s,%.2f,%s\n", t.Date, -t.Amount, t.Name))
		checkErr(err)
	}
	f.Sync()
}

func printTrans(trans []plaid.Transaction) {
	for _, t := range trans {
		fmt.Printf("%s,%.2f,%s\n", t.Date, -t.Amount, t.Name)
	}
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}
