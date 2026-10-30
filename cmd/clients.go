package cmd

import (
	"register/pkg/banking"

	"github.com/plaid/plaid-go/v15/plaid"
)

func GetBankingClient() *banking.Client {
	plaidEnvironment := plaid.Development
	if config.PlaidEnvironment == "production" {
		plaidEnvironment = plaid.Production
	}
	return banking.NewClient(config.UserID, config.Banks, config.IncomeSources, config.PlaidClientID, config.PlaidEnvSecrets[config.PlaidEnvironment], plaidEnvironment, func(_ string) {})
}
