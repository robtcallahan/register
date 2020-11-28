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

	cfg "register/pkg/config"

	"github.com/spf13/cobra"
	// homedir "github.com/mitchellh/go-homedir"
	// "github.com/spf13/viper"
)

var (
	config *cfg.Config
	// SpreadsheetID ...
	SpreadsheetID string
	// StartRow ...
	StartRow int64
	// EndRow ...
	EndRow int64
	// Debug ...
	Debug bool
	// Test ...
	Test bool

	err error
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "register",
	Short: "Reads bank transactions and updates the financial register spreadsheet",
	Long: `Register reads bank and credit card transactions from Wells Fargo, Fidlity, Chase,
and Citi, both the Register and Budget tabs from your Google Sheets financial spreadsheet,
removes duplicates and updates the Register tab with new transactions subtracting those
amounts from the appropriate budget category columns.`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			cmd.Help()
			os.Exit(0)
		}
	},
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
	// cobra.OnInitialize(initConfig)
	cobra.OnInitialize()

	rootCmd.Flags().Int64VarP(&StartRow, "start", "s", config.RegisterStartRow, "The first row to start reading in the spreadsheet")
	rootCmd.Flags().Int64VarP(&EndRow, "end", "e", config.RegisterEndRow, "The last row to read in the spreadsheet")
	rootCmd.Flags().StringVarP(&SpreadsheetID, "id", "i", config.SpreadsheetID, "The Google spreadsheet id")

	rootCmd.Flags().BoolVarP(&Test, "test", "t", false, "Test mode; no updates performed")
	rootCmd.Flags().BoolVarP(&Debug, "debug", "d", false, "Debug mode")
}

// initConfig reads in config file and ENV variables if set.
// func initConfig() {
// 	if cfgFile != "" {
// 		// Use config file from the flag.
// 		viper.SetConfigFile(cfgFile)
// 	} else {
// 		// Find home directory.
// 		home, err := homedir.Dir()
// 		if err != nil {
// 			fmt.Println(err)
// 			os.Exit(1)
// 		}

// 		// Search config in home directory with name ".register" (without extension).
// 		viper.AddConfigPath(home)
// 		viper.SetConfigName(".register")
// 	}

// 	viper.AutomaticEnv() // read in environment variables that match

// 	// If a config file is found, read it in.
// 	if err := viper.ReadInConfig(); err == nil {
// 		fmt.Println("Using config file:", viper.ConfigFileUsed())
// 	}
// }
