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

	cfg "register/pkg/config"
	"register/pkg/driver"
	"register/pkg/handler"

	"github.com/spf13/cobra"
)

// storeCmd represents the store command
var storeCmd = &cobra.Command{
	Use:   "store",
	Short: "Just a placeholder. Doesn't do anything yet.",
	Long:  `Just a placeholder. Doesn't do anything yet.`,
	Run: func(cmd *cobra.Command, args []string) {
		store()
	},
}

func init() {
	config = cfg.ReadConfig()
	rootCmd.AddCommand(storeCmd)
}

func store() {
	conn, err := driver.ConnectSQL(&driver.ConnectParams{
		DBType: driver.DBType(config.DBType),
		Host:   config.DBHost,
		Port:   config.DBPort,
		DBName: config.DBName,
		User:   config.DBUsername,
		Pass:   config.DBPassword,
	})
	if err != nil {
		panic(err)
	}
	qHandler := handler.NewQueryHandler(conn)

	cols := qHandler.GetColumns()
	for i := 0; i < 10; i++ {
		c := cols[i]
		fmt.Printf("%s\n", c.Name)
	}
}
