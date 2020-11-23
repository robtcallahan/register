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
	"register/pkg/sheets"

	"github.com/spf13/cobra"
)

// copyCmd represents the copy command
var copyCmd = &cobra.Command{
	Use:   "copy",
	Short: "A brief description of your command",
	Long:  `A longer description.`,
	Run: func(cmd *cobra.Command, args []string) {
		copy(cmd, args)
	},
}

func init() {
	rootCmd.AddCommand(copyCmd)
	config = cfg.ReadConfig()

	copyCmd.Flags().IntP("copies", "c", 0, "The number of times to copy the last 2 rows")
	copyCmd.Flags().Int64P("start", "s", config.RegisterStartRow, "The last used row in the spreadsheet")
	copyCmd.Flags().Int64P("end", "e", config.RegisterEndRow, "The last used row in the spreadsheet")
	copyCmd.Flags().StringP("id", "i", config.SpreadsheetID, "The Google spreadsheet id")
}

func copy(cmd *cobra.Command, args []string) {
	var err error
	ssID, _ := cmd.Flags().GetString("id")
	copies, _ := cmd.Flags().GetInt("copies")
	startRow, _ := cmd.Flags().GetInt64("start")
	endRow, _ := cmd.Flags().GetInt64("end")

	srv := &sheets.SheetService{
		Service:       sheets.NewService(),
		SpreadsheetID: ssID,
	}
	reg := sheets.NewRegisterSheet(srv, *config, startRow, endRow)

	fmt.Printf("Reading Register...\n")
	reg.ID, err = srv.GetSheetID(config.TabNames["register"])
	checkError(err)
	reg.Read(false)

	fmt.Printf("Copying rows %d times...\n", copies)
	reg.CopyRows(copies)
}
