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
	cfg "register/pkg/config"
	"register/pkg/database"

	"github.com/spf13/cobra"
)

// storeCmd represents the store command
var storeCmd = &cobra.Command{
	Use:   "store",
	Short: "A brief description of your command",
	Long:  `A longer description `,
	Run: func(cmd *cobra.Command, args []string) {
		store(cmd, args)
	},
}

var (
	print bool
)

func init() {
	config = cfg.ReadConfig()
	rootCmd.AddCommand(storeCmd)

	storeCmd.Flags().BoolVarP(&print, "print", "p", false, "Print the data")
	storeCmd.Flags().BoolVarP(&Debug, "debug", "d", false, "Debug mode")
}

func store(cmd *cobra.Command, args []string) {
	db := database.New(database.ConfigParams{
		Debug:      Debug,
		DBName:     config.DBName,
		DBUsername: config.DBUsername,
		DBPassword: config.DBPassword,
	})

	if print {
		db.PrintData()
	}
}
