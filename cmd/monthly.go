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

	"register/pkg/repository"
	"register/pkg/repository/mysql"

	"register/api/services/sheets_service"

	"github.com/spf13/cobra"
)

// monthlyCmd represents the monthly command
var monthlyCmd = &cobra.Command{
	Use:   "monthly",
	Short: "Monthly aggregates monthly budget category expenses and updates the monthly summary tabs",
	Long:  `Monthly aggregates monthly budget category expenses and updates the monthly summary tabs`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return monthly()
	},
}

func init() {
	rootCmd.AddCommand(monthlyCmd)
}

func monthly() error {
	conn, err := repository.Connect(&config.DB)
	if err != nil {
		return err
	}
	qHandler := mysql.NewMySQLQueryRepo(conn)

	sheetsProvider, err := sheets_provider.New(options.SpreadsheetID, config)
	if err != nil {
		return err
	}
	sheetsService := sheets_service.NewRegisterService(sheetsProvider)
	sheetsService.IncomeSources = config.IncomeSources
	cols, err := qHandler.GetColumns()
	if err != nil {
		return err
	}
	registerSheet, err := sheetsService.NewRegisterSheet(config, cols)
	if err != nil {
		return err
	}

	monthlyService := sheets_service.NewMonthlyService(sheetsProvider, config.IncomeSources)

	fmt.Printf("Reading Register...\n")
	_, err = sheetsService.ReadRegisterSheet(registerSheet)
	if err != nil {
		return err
	}

	fmt.Println("Aggregating...")
	catAgg, payeeAgg, err := monthlyService.Aggregate(*registerSheet, cols)
	if err != nil {
		return err
	}

	fmt.Println("Updating...")
	if err := monthlyService.UpdateMonthlyCategories("MonthlyCategories", catAgg, cols); err != nil {
		return fmt.Errorf("monthly categories: %w", err)
	}
	if err := monthlyService.UpdateMonthlyPayees("MonthlyPayees", payeeAgg); err != nil {
		return fmt.Errorf("monthly payees: %w", err)
	}
	if err := monthlyService.UpdateYearlySummary("YearlySummary", catAgg, cols); err != nil {
		return fmt.Errorf("yearly summary: %w", err)
	}
	return nil
}
