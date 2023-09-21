/*
Copyright © 2020 NAME HERE <EMAIL ADDRESS>

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
	"register/api/providers/sheets_provider"

	"register/pkg/driver"
	"register/pkg/handler"

	"register/api/services/sheets_service"

	"github.com/spf13/cobra"
)

// monthlyCmd represents the monthly command
var monthlyCmd = &cobra.Command{
	Use:   "monthly",
	Short: "Monthly aggregates monthly budget category expenses and updates the monthly summary tabs",
	Long:  `Monthly aggregates monthly budget category expenses and updates the monthly summary tabs`,
	Run: func(cmd *cobra.Command, args []string) {
		monthly()
	},
}

func init() {
	rootCmd.AddCommand(monthlyCmd)
}

const (
// jsonDir = "/Users/rob/ws/go/src/register/services/sheets_service/json"
// jsonDir = "/Users/rcallahan/workspace/go/src/register/services/sheets_service/json"
)

func monthly() {
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

	sheetsProvider, err := sheets_provider.New(options.SpreadsheetID, config)
	checkError(err)
	sheetsService := sheets_service.New(sheetsProvider)
	checkError(err)
	err = sheetsService.NewRegisterSheet(config)

	fmt.Printf("Reading Register...\n")
	_, err = sheetsService.ReadRegisterSheet()
	checkError(err)

	cols := qHandler.GetColumns()

	fmt.Println("Aggregating...")
	catAgg, payeeAgg := sheetsService.Aggregate(cols)

	//sheets_service.WriteJSONFile(jsonDir+"columns.json", cols)
	//sheets_service.WriteJSONFile(jsonDir+"register.json", sheetsService.RegisterSheet.Register)
	//sheets_service.WriteJSONFile(jsonDir+"cat_agg.json", catAgg)
	//sheets_service.WriteJSONFile(jsonDir+"payee_agg.json", payeeAgg)
	/**/
	fmt.Println("Updating...")
	sheetsService.UpdateMonthlyCategories("MonthlyCategories", catAgg, cols)
	sheetsService.UpdateMonthlyPayees("MonthlyPayees", payeeAgg)
}
