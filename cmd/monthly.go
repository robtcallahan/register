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
	"regexp"

	"register/pkg/driver"
	"register/pkg/handler"
	"register/pkg/sheets"

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
	sheetService := &sheets.SheetService{
		Service:       sheets.NewService(),
		SpreadsheetID: options.SpreadsheetID,
	}

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

	fmt.Printf("Reading Register...\n")
	regSrv := sheets.NewRegisterSheet(sheetService, *config, config.MonthlyStartRow, config.MonthlyEndRow, options.Debug)
	id, err := sheetService.GetSheetID(config.TabNames["register"])
	regSrv.ID = id
	checkError(err)
	register, _, rangeValues := regSrv.Read()

	// map of register entries by month and category
	catAgg := make(map[string]map[string]float64)

	// map of register entries by monty and payee
	payeeAgg := make(map[string]map[string]float64)

	cols := qHandler.GetColumns()

	fmt.Println("Aggregating...")
	for i, r := range register {
		re := regexp.MustCompile(`(\d\d)/\d\d/20`)
		m := re.FindStringSubmatch(r.Date)
		if len(m) > 0 {
			k := m[1] + "/20"
			if _, ok := payeeAgg[k]; !ok {
				payeeAgg[k] = make(map[string]float64)
			}
			payeeAgg[k][r.Name] = payeeAgg[k][r.Name] + r.Deposit - r.Withdrawal - r.CreditCard

			if _, ok := catAgg[k]; !ok {
				catAgg[k] = make(map[string]float64)
			}

			if r.Name == "CrowdStrike Salary" {
				catAgg[k]["CrowdStrike Salary"] += r.Deposit
				continue
			}

			for j := 10; j < len(rangeValues[i*2]); j++ {
				if cols[j].Name == "Credit Cards" || r.Deposit != 0 {
					continue
				}
				f32 := regSrv.GetRegisterField(rangeValues[i*2], cols[j].ColumnIndex)
				catAgg[k][cols[j].Name] = catAgg[k][cols[j].Name] + f32
			}
		}
	}

	fmt.Println("Updating...")
	sheetService.UpdateMonthlyCategories("MonthlyCategories", catAgg, cols)
	sheetService.UpdateMonthlyPayees("MonthlyPayees", payeeAgg)
}
