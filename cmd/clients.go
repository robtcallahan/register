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
	return banking.NewClient(&banking.ClientOptions{
		UserID:           config.UserID,
		Banks:            config.Banks,
		IncomeSources:    config.IncomeSources,
		Debug:            options.Debug,
		PlaidClientID:    config.PlaidClientID,
		PlaidSecret:      config.PlaidEnvSecrets[config.PlaidEnvironment],
		PlaidEnvironment: plaidEnvironment,
	})
}
