package plaid_auth

import (
	"context"
	"fmt"
	"register/pkg/banking"

	"github.com/plaid/plaid-go/v15/plaid"
)

func GetLinkToken(c *banking.Client) (string, error) {
	ctx := context.Background()
	user := plaid.LinkTokenCreateRequestUser{
		ClientUserId: c.UserID,
	}

	request := plaid.NewLinkTokenCreateRequest(
		"Plaid Test",
		"en",
		[]plaid.CountryCode{plaid.COUNTRYCODE_US},
		user,
	)
	request.SetClientId(c.ClientID)
	request.SetSecret(c.Secret)
	request.SetProducts([]plaid.Products{plaid.PRODUCTS_TRANSACTIONS})
	request.SetLinkCustomizationName("default")
	request.SetRedirectUri("https://localhost:9000/public/oauth.html")

	resp, _, err := c.PlaidClient.PlaidApi.LinkTokenCreate(ctx).LinkTokenCreateRequest(*request).Execute()
	if err != nil {
		return "", fmt.Errorf("create Plaid link token: %w", err)
	}

	linkToken := resp.GetLinkToken()
	return linkToken, nil
}

func ExchangePublicToken(ctx context.Context, c *banking.Client, publicToken string) (string, error) {
	exchangePublicTokenReq := plaid.NewItemPublicTokenExchangeRequest(publicToken)
	exchangePublicTokenResp, resp, err := c.PlaidClient.PlaidApi.ItemPublicTokenExchange(ctx).ItemPublicTokenExchangeRequest(
		*exchangePublicTokenReq,
	).Execute()
	if err != nil {
		return "", fmt.Errorf("create Plaid item public token exchange: %w", err)
	}
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("bad response code from ItemPublicTokenExchangeRequest: %d", resp.StatusCode)
	}
	return exchangePublicTokenResp.GetAccessToken(), nil
}
