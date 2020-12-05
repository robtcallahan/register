package banking

import (
	"github.com/plaid/plaid-go/plaid"
	"register/pkg/config"
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
	Verbose       bool
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
	Verbose     bool
}
