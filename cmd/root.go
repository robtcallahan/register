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

package cmd

import (
	"fmt"
	"os"
	"strings"

	"register/pkg/banking"
	cfg "register/pkg/config"

	"github.com/plaid/plaid-go/v15/plaid"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "register",
	Short: "Reads bank transactions and updates the financial register spreadsheet",
	Long: `Register reads bank and credit card transactions from Wells Fargo, Fidelity, Chase,
and Citi, both the Register and Budget tabs from your Google Sheets financial spreadsheet,
removes duplicates and updates the Register tab with new transactions subtracting those
amounts from the appropriate budget category columns.`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			_ = cmd.Help()
			os.Exit(0)
		}
	},
}

const ConfigFile = "config/config.json"

var options = &cfg.Options{}
var config = &cfg.Config{}

type Client struct {
	BankClient *banking.Client
}

var client = new(Client)

type LinkToken struct {
	LinkToken string `json:"link_token"`
}

func (t *LinkToken) ToString() string {
	return fmt.Sprintf("%s", t.LinkToken)
}

type AccessToken struct {
	AccessToken string `json:"access_token"`
}

func (t *AccessToken) ToString() string {
	return fmt.Sprintf("%s", t.AccessToken)
}

type PublicToken struct {
	PublicToken string `json:"public_token"`
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	var err error

	cobra.OnInitialize()

	config, err = cfg.ReadConfig(ConfigFile)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	var bankIDs string
	//flag.StringVarP(&bankIDs, "bank-ids", "b", "wellsfargo,fidelity,chase", "comma-separated list of bank IDs; default: wellsfargo,fidelity,chase")
	//flag.StringVarP(&options.SpreadsheetID, "ss_id", "s", config.SpreadsheetID, "The Google spreadsheet id")
	//flag.BoolVarP(&options.Debug, "debug", "d", false, "Debug mode")
	//flag.Parse()

	rootCmd.PersistentFlags().StringVarP(&bankIDs, "bank-ids", "b", "wellsfargo,fidelity,chase", "comma-separated list of bank IDs")
	rootCmd.PersistentFlags().StringVarP(&options.SpreadsheetID, "ss_id", "s", config.SpreadsheetID, "The Google spreadsheet id")
	rootCmd.PersistentFlags().BoolVarP(&options.Debug, "debug", "d", false, "Debug mode")

	options.BankIDs = strings.Split(bankIDs, ",")
}

func getBankingClient() *Client {
	if client.BankClient != nil {
		return client
	}

	plaidEnvironment := plaid.Development
	if config.PlaidEnvironment == "production" {
		plaidEnvironment = plaid.Production
	}
	client.BankClient = banking.NewClient(&banking.ClientOptions{
		UserID:           config.UserID,
		Banks:            config.Banks,
		Debug:            options.Debug,
		PlaidClientID:    config.PlaidClientID,
		PlaidSecret:      config.PlaidEnvSecrets[config.PlaidEnvironment],
		PlaidEnvironment: plaidEnvironment,
		PlaidTokensDir:   config.PlaidTokensDir,
	})
	return client
}

func checkError(err error) {
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	return
}
