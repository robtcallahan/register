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
	"register/api/services/sheets_service"

	"register/pkg/driver"
	"register/pkg/handler"

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

func monthly() {
	sheetsService, err := sheets_service.New(options.SpreadsheetID, options.Verbose)
	checkError(err)

	conn, err := driver.ConnectSQL(&driver.ConnectParams{
		DBType: driver.DBType(config.DBType),
		Host:   config.DBHost,
		Port:   config.DBPort,
		DBName: config.DBName,
		User:   config.DBUsername,
		Pass:   config.DBPassword,
	})
	checkError(err)
	qHandler := handler.NewQueryHandler(conn)
	cols := qHandler.GetColumns()

	dir := "/Users/rcallahan/workspace/go/src/register/api/services/sheets_service/json/"

	fmt.Printf("Reading Register...\n")
	err = sheetsService.NewRegisterSheet(config.MonthlyStartRow, config.MonthlyEndRow)
	checkError(err)
	err = sheetsService.ReadRegisterSheet()
	checkError(err)

	fmt.Println("Aggregating...")
	catAgg, payeeAgg := sheetsService.Aggregate(sheetsService.RegisterSheet, cols)

	sheets_service.WriteJSONFile(dir + "columns.json", cols)
	sheets_service.WriteJSONFile(dir + "register.json", sheetsService.RegisterSheet.Register)
	sheets_service.WriteJSONFile(dir + "cat_agg.json", catAgg)
	sheets_service.WriteJSONFile(dir + "payee_agg.json", payeeAgg)

	fmt.Println("Updating...")
	sheetsService.UpdateMonthlyCategories("MonthlyCategories", catAgg, cols)
	sheetsService.UpdateMonthlyPayees("MonthlyPayees", payeeAgg)
}
