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
	"errors"
	"fmt"

	"register/pkg/config"

	"github.com/plaid/plaid-go/v15/plaid"
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

type Client struct {
	PlaidClient   *plaid.APIClient
	ClientID      string
	Secret        string
	Environment   plaid.Environment
	UserID        string
	Banks         map[string]config.Bank
	IncomeSources []config.IncomeSource
	Progress      func(_ string)
}

type Balance struct {
	BankName string
	Amount   float64
	Error    error
}

func NewClient(userID string, banks map[string]config.Bank, incomeSources []config.IncomeSource, clientID, secret string, env plaid.Environment, progress func(_ string)) *Client {
	c := &Client{
		ClientID:      clientID,
		Secret:        secret,
		Environment:   env,
		UserID:        userID,
		Banks:         banks,
		IncomeSources: incomeSources,
		Progress:      progress,
	}
	configuration := plaid.NewConfiguration()
	configuration.AddDefaultHeader("PLAID-CLIENT-ID", c.ClientID)
	configuration.AddDefaultHeader("PLAID-SECRET", c.Secret)
	configuration.UseEnvironment(c.Environment)
	client := plaid.NewAPIClient(configuration)
	c.PlaidClient = client
	return c
}

func (c *Client) GetAccounts(ctx context.Context, accessToken string) ([]plaid.AccountBase, error) {
	accountsGetRequest := plaid.NewAccountsGetRequest(accessToken)
	accountsGetResponse, httpResponse, err := c.PlaidClient.PlaidApi.AccountsGet(ctx).AccountsGetRequest(*accountsGetRequest).Execute()

	if err != nil {
		buf := new(bytes.Buffer)
		_, err2 := buf.ReadFrom(httpResponse.Body)
		if err2 != nil {
			return nil, err2
		}
		return nil, fmt.Errorf("%w\n: %s", err, buf.Bytes())
	}
	return accountsGetResponse.GetAccounts(), err
}

func (c *Client) GetBalances(bankIDs []string) map[string]Balance {
	ctx := context.Background()
	balances := make(map[string]Balance, 4)
	balance := Balance{}

	for _, id := range bankIDs {
		bank := c.Banks[id]

		if bank.AccessToken == "" {
			balances[id] = Balance{
				BankName: bank.Name,
				Error:    fmt.Errorf("no Plaid access token configured for %s", bank.Name),
			}
			continue
		}

		amount, err := c.GetBalance(ctx, bank.AccessToken, id)
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

func (c *Client) GetBalance(ctx context.Context, accessToken string, bankID string) (*float64, error) {
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
		return nil, fmt.Errorf("error with %s: %w\n%s", bankID, err, buf.String())
	}
	nullFloat64 := balancesGetResp.Accounts[0].Balances.Current
	if !nullFloat64.IsSet() {
		return nil, errors.New("available balance is not set")
	}
	return nullFloat64.Get(), err
}

func (c *Client) GetBankStatus(bankID string) (*plaid.Institution, error) {
	ctx := context.Background()
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
		return nil, fmt.Errorf("%w\n%s", err, buf.String())
	}
	return &resp.Institution, nil
}
