package cmd

import (
	"fmt"
	cfg "register/pkg/config"

	"github.com/spf13/cobra"
)

var balancesCmd = &cobra.Command{
	Use:   "balances",
	Short: "Get all bank balances",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		getBalances()
	},
}

func init() {
	if c, err := cfg.ReadConfig(ConfigFile); err == nil {
		config = c
	}
	rootCmd.AddCommand(balancesCmd)
}

func getBalances() {
	bankClient := GetBankingClient()
	balances := bankClient.GetBalances(options.BankIDs)

	for _, balance := range balances {
		if balance.Error != nil {
			fmt.Println(balance.Error.Error())
		} else {
			fmt.Printf("%s: $%.2f\n", balance.BankName, balance.Amount)
		}
	}
}
