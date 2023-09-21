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

	"register/api/providers/sheets_provider"
	"register/api/services/sheets_service"
	cfg "register/pkg/config"

	"github.com/spf13/cobra"
)

var copyCmd = &cobra.Command{
	Use:   "copy",
	Short: "Copies the last 2 rows of the Register spreadsheet -c <num> times",
	Long: `Copies the last 2 rows of the Register spreadsheet then number of times
specified using the -c <num> or --copy <num> options. `,
	Run: func(cmd *cobra.Command, args []string) {
		copyRows()
	},
}

func init() {
	rootCmd.AddCommand(copyCmd)

	//flag.IntVarP(&options.NumCopies, "number", "n", 10, "he number of times to copy the last 2 rows; default=10")
	//flag.Parse()

	copyCmd.Flags().IntVarP(&options.NumCopies, "number", "n", 10, "he number of times to copy the last 2 rows; default=10")
}

func copyRows() {
	var err error

	config, err = cfg.ReadConfig(ConfigFile)
	checkError(err)

	sheetsProvider, err := sheets_provider.New(options.SpreadsheetID, config)
	checkError(err)
	sheetsService := sheets_service.New(sheetsProvider)
	checkError(err)
	err = sheetsService.NewRegisterSheet(config)
	checkError(err)

	fmt.Printf("Reading Register...\n")
	_, err = sheetsService.ReadRegisterSheet()
	checkError(err)

	fmt.Printf("Copying rows %d times...\n", options.NumCopies)
	err = sheetsService.CopyRows(options.NumCopies)
	checkError(err)
}
