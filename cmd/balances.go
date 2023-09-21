package cmd

import (
	"fmt"
	cfg "register/pkg/config"

	"github.com/spf13/cobra"
	"golang.org/x/net/context"
)

var balancesCmd = &cobra.Command{
	Use:   "balances",
	Short: "Get all bank balances",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		getBalances()
	},
}

var ctx = context.Background()

func init() {
	config, _ = cfg.ReadConfig(ConfigFile)
	rootCmd.AddCommand(balancesCmd)

	client = getBankingClient()
}

func getBalances() {
	balances := client.BankClient.GetBalances(options.BankIDs)

	for _, balance := range balances {
		if balance.Error != nil {
			fmt.Println(balance.Error.Error())
		} else {
			fmt.Printf("%s: $%.2f\n", balance.BankName, balance.Amount)
		}
	}
}
