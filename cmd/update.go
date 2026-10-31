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
	"bufio"
	"fmt"
	"math"
	"os"
	"register/api/providers/sheets_provider"
	"register/api/services/sheets_service"
	"register/api/services/update_service"
	"register/pkg/banking"
	"register/pkg/models"
	"register/pkg/repository"
	"register/pkg/repository/mysql"
	"slices"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Reads bank transactions and updates the financial register spreadsheet",
	Long: `Register reads bank and credit card transactions from Wells Fargo, Fidelity, Chase,
and Citi, both the Register and Budget tabs from your Google Sheets financial spreadsheet,
removes duplicates and updates the Register tab with new transactions subtracting those
amounts from the appropriate budget category columns.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return update()
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)

	updateCmd.Flags().BoolVar(&options.DryRun, "dry-run", false, "Print-only run: no DB or spreadsheet updates performed")
	updateCmd.Flags().BoolVarP(&options.UseCSVFiles, "csv", "c", false, "Read CSV files; default=false")
}

func update() error {
	var (
		reader       = bufio.NewReader(os.Stdin)
		transactions []*models.Transaction
	)

	conn, err := repository.Connect(&config.DB)
	if err != nil {
		return err
	}
	queryRepo := mysql.NewMySQLQueryRepo(conn)

	sheetsProvider, err := sheets_provider.New(options.SpreadsheetID, config)
	if err != nil {
		return err
	}

	bankClient := GetBankingClient()
	registerService := sheets_service.NewRegisterService(sheetsProvider)
	registerService.IncomeSources = config.IncomeSources
	budgetService := sheets_service.NewBudgetService(sheetsProvider)
	updateService := update_service.NewUpdateService(registerService, budgetService, bankClient, queryRepo, config, options)

	// define the listener for printing progress strings
	updateService.Progress = func(msg string) { fmt.Println(msg) }
	bankClient.Progress = func(msg string) { fmt.Println(msg) }

	columns, err := queryRepo.GetColumns()
	if err != nil {
		return err
	}
	registerSheet, err := registerService.NewRegisterSheet(config, columns)
	if err != nil {
		return err
	}

	fmt.Println("Loading Transactions...")
	transactions, err = updateService.LoadTransactions(registerSheet)
	if err != nil {
		return err
	}

	fmt.Println("Preparing Transactions...")
	var dateCorrections []*update_service.DateCorrection
	transactions, dateCorrections, err = updateService.PrepareTransactions(transactions, registerSheet.Entries)
	if err != nil {
		return err
	}

	if transactions == nil {
		fmt.Println("No new transactions")
		if options.DryRun {
			fmt.Println("Dry run was on — no spreadsheet or database updates were made.")
		}
		return nil
	}

	printTransactions(transactions, dateCorrections)

	if !options.DryRun {
		transactions, err = gatherTransactionInfo(reader, bankClient, queryRepo, transactions)
		if err != nil {
			return err
		}

		fmt.Println("Applying transactions:")
		err = updateService.ApplyTransactions(registerSheet.TabName, transactions, dateCorrections)
		return err
	}

	fmt.Println("Dry run was on — no spreadsheet or database updates were made.")
	return nil
}

// gatherTransactionInfo looks through the transactions to see if there are any merchants
// that are missing an entry in the merchants table. If so, it calls functions to
// print out category columns, prompts the user for a desired merchant name, a category,
// whether to search by exact name or by substring and updates the transactions array.
func gatherTransactionInfo(reader *bufio.Reader, bankClient *banking.Client, queryRepo repository.QueryRepo, transactions []*models.Transaction) ([]*models.Transaction, error) {
	if needUserInfoForTransactions(transactions) {
		fmt.Println("Info needed...")

		// print the pretty columns so the user has column IDs to enter for each category
		if err := printColumns(queryRepo); err != nil {
			return nil, err
		}

		// get all the necessary input from the user
		var err error
		transactions, err = getUserInput(reader, bankClient, queryRepo, transactions)
		if err != nil {
			return nil, err
		}
	}

	// get notes for any checks or Amazon purchases (They occur often. It's a problem I have)
	transactions = getTransactionNotes(reader, transactions)

	return transactions, nil
}

// needTransactionName loops through all transactions calling the needsName function
// to determine if it is necessary to prompt the user for input on the transaction
func needUserInfoForTransactions(trans []*models.Transaction) bool {
	return slices.ContainsFunc(trans, needsNameForRegister)
}

// getUserInput is the user interactive piece. It obtains a desired merchant name for
// the Register spreadsheet for the banks name,
func getUserInput(reader *bufio.Reader, bankClient *banking.Client, queryRepo repository.QueryRepo, trans []*models.Transaction) ([]*models.Transaction, error) {
	var err error
	updated := true

	for updated {
		// get all the input needed from the user, prompting for each item
		updated, trans, err = readFromUser(reader, queryRepo, trans)
		if err != nil {
			return nil, err
		}

		// GetLookupData queries the DB and creates an array of models.DataRow where
		// every element contains an aggregate of data from both tables.
		lookupData, err := queryRepo.GetLookupData()
		if err != nil {
			return nil, err
		}

		// lookupData creates a match array for FormatMerchantNames
		matcher, err := banking.NewMerchantMatcher(lookupData)
		if err != nil {
			return nil, err
		}

		// build the matcher for the income sources
		income, err := bankClient.IncomeMatcher()
		if err != nil {
			return nil, err
		}

		// get a columns list from the DB
		columns, err := queryRepo.GetColumns()
		if err != nil {
			return nil, err
		}

		// this applies merchant rules to each transaction
		trans = banking.FormatMerchantNames(trans, matcher, income, columns)
	}
	return trans, nil
}

// needsName reports whether the merchant matcher left this transaction
// without a name of its own: no name at all, or the raw bank name still
// standing in as a placeholder because no rule matched (in which case it
// has no column either). Checks are excluded: they take the CHECK path
// (note prompt) instead.
func needsNameForRegister(t *models.Transaction) bool {
	if strings.Contains(t.BankName, "CHECK #") {
		return false
	}
	return t.Name == "" || (t.Name == t.BankName && t.ColumnIndex == 0)
}

// printColumns echos a list of budget categories to the terminal formatted
// in three columns for a shorter length. Each category has an id number
// associated with it which the user enters.
func printColumns(db repository.QueryRepo) error {
	columns, err := db.GetColumns()
	if err != nil {
		return err
	}
	filtered := categoryColumns(columns)

	// this will allow us to print 3 columns on the screen
	numRows := int(math.Floor(float64(len(filtered)) / 3))
	remItems := len(filtered) % 3

	i := 0
	for r := 1; r <= numRows; r++ {
		fmt.Printf("%2d %-30s %2d %-30s %2d %-30s\n",
			filtered[i].ID, filtered[i].Name,
			filtered[i+1].ID, filtered[i+1].Name,
			filtered[i+2].ID, filtered[i+2].Name,
		)
		i += 3
	}
	for j := 0; j < remItems; i++ {
		fmt.Printf("%2d %-30s \n", filtered[i].ID, filtered[i].Name)
		j++
	}
	return nil
}

// categoryColumns obtains a list of categories from the DB
func categoryColumns(columns []models.Column) []models.Column {
	var filtered []models.Column
	for _, col := range columns {
		if !col.IsCategory() {
			continue
		}
		filtered = append(filtered, col)
	}
	return filtered
}

// getTransactionNotes asks the user for a note to enter into the Register spreadsheet
// for the transaction if it is either a check for an Amazon purchase which would
// describe what the user bought.
func getTransactionNotes(reader *bufio.Reader, trans []*models.Transaction) []*models.Transaction {
	for i, t := range trans {
		if t.Name == models.CheckRegisterName || t.Name == "Amazon" || t.Name == "Amazon Marketplace" {
			fmt.Printf("Source: %s, Name: %s, Date: %s, Amt: $%0.2f\n", t.Source, t.Name, t.Date, t.Amount)
			trans[i].Note = readString(reader, "    Note: ")
		}
	}
	return trans
}

// readFromUser loops over every transaction and if it doesn't have a user-specified name
// for the Register spreadsheet it will prompt for the user for input
func readFromUser(reader *bufio.Reader, queryRepo repository.QueryRepo, trans []*models.Transaction) (bool, []*models.Transaction, error) {
	var err error

	for i, t := range trans {
		if needsNameForRegister(t) {
			fmt.Printf("Source: %s, Date: %s, Amt: $%0.2f\n", t.BankName, t.Date, t.Amount)

			// get the desired bank name
			trans[i].Name = readString(reader, "            Name: ")
			var columnID int
			for {
				// get the column ID for the desired category for this merchant
				columnID, err = readInt(reader, "       Column ID: ")
				if err != nil {
					fmt.Println(err.Error())
					continue
				}
				break
			}

			// ask for a note for this transaction
			trans[i].Note = readString(reader, "           Note: ")

			// ask the user for the desired matching type, either exact or substring which is the default
			matchType := "substring"
			for {
				v := strings.ToLower(strings.TrimSpace(readString(reader, "     Match type (e=exact, s=substring) [s]: ")))
				if v == "" || v == "s" {
					matchType = "substring"
					break
				}
				if v == "e" {
					matchType = "exact"
					break
				}
			}

			// the match pattern: the full bank name, or a shorter
			// substring that catches every variant of this merchant.
			// the raw bank name is stored separately on the rule.
			pattern := t.BankName
			if matchType == "substring" {
				if sub := readString(reader, fmt.Sprintf("         Substring [%s]: ", t.BankName)); sub != "" {
					pattern = sub
				}
			}

			// a new substring rule outranks everything already in the
			// table, so its variants can't be shadowed by an older rule
			priority := 0
			if matchType == "substring" {
				maxPriority, err := queryRepo.GetMaxMerchantPriority()
				if err != nil {
					return false, nil, err
				}
				priority = maxPriority + 10
			}

			// the menu prints column IDs, but the transaction wants the
			// column's position — translate instead of sharing one number
			columns, err := queryRepo.GetColumns()
			if err != nil {
				return false, nil, err
			}
			if index, ok := models.NewColumnSet(columns).IndexForID(columnID); ok {
				trans[i].ColumnIndex = index
			}

			// create the new merchant entry in the merchant table
			if err := queryRepo.CreateMerchant(&models.Merchant{
				Name:      trans[i].Name,
				BankName:  t.BankName,
				ColumnID:  columnID,
				MatchType: matchType,
				Pattern:   pattern,
				Priority:  priority,
			}); err != nil {
				return false, nil, err
			}
			fmt.Printf("    Rule saved (priority %d).\n", priority)
			return true, trans, nil
		}
	}
	return false, trans, nil
}

// readString receives a prompt and reads what the user enters and returns the value trimming the carriage return
func readString(reader *bufio.Reader, prompt string) string {
	fmt.Print(prompt)
	v, _ := reader.ReadString('\n')
	return strings.TrimSuffix(v, "\n")
}

// readInt receives a prompt and reads the number the user enters and returns an integer value
func readInt(reader *bufio.Reader, prompt string) (int, error) {
	fmt.Print(prompt)
	v, _ := reader.ReadString('\n')
	v = strings.TrimSuffix(v, "\n")
	i, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("could not convert %s to an int: %w", v, err)
	}
	return i, nil
}

func printTransactions(trans []*models.Transaction, dateCorrections []*update_service.DateCorrection) {
	fmt.Println("")
	if len(dateCorrections) > 0 {
		fmt.Printf("Updated Transactions:\n")
		fmt.Printf("%-57s %-30s %-12s %-8s %-8s %-8s %-60s\n", "Transaction ID", "Key", "Source", "Old Date", "New Date", "Amount", "Name")
		fmt.Printf("%-57s %-30s %-12s %-8s %-8s %-8s %-60s\n", dashes(57), dashes(30), dashes(12), dashes(8), dashes(8), dashes(8), dashes(60))
		for _, c := range dateCorrections {
			e := c.Entry
			amt := getAmount(e.Source, e.Deposit, e.Withdrawal, e.CreditPurchase)
			fmt.Printf("%-57s %-30s %-12s %-8s %-8s %8.2f %-60s\n", e.TransactionID, e.Key, e.Source, e.Date, c.NewDate, amt, e.Name)
		}
		fmt.Println("")

	}

	fmt.Printf("New Transactions:\n")
	fmt.Printf("%-57s %-30s %-12s %-8s %-8s %-60s\n", "Transaction ID", "Key", "Source", "Date", "Amount", "Name")
	fmt.Printf("%-57s %-30s %-12s %-8s %-8s %-60s\n", dashes(57), dashes(30), dashes(12), dashes(8), dashes(8), dashes(60))
	for _, t := range trans {
		amt := getAmount(t.Source, t.Deposit, t.Withdrawal, t.CreditPurchase)
		fmt.Printf("%-57s %-30s %-12s %-8s %8.2f %-60s\n", t.TransactionID, t.Key, t.Source, t.Date, amt, t.Name)
	}
	fmt.Println("")
}

func getAmount(source string, deposit, withdrawal, creditPurchase float64) float64 {
	amt := 0.0
	if source == "WellsFargo" {
		if deposit != 0 {
			amt = deposit
		} else {
			amt = withdrawal
		}
	} else {
		amt = creditPurchase
	}
	return amt
}

func dashes(count int) string {
	return strings.Repeat("-", count)
}
