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
	stdcsv "encoding/csv"
	"fmt"
	"math"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"register/api/providers/sheets_provider"
	"register/api/services/sheets_service"
	"register/api/services/update_service"
	"register/pkg/banking"
	"register/pkg/models"
	"register/pkg/repository"
	"register/pkg/repository/mysql"

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

/**/
func init() {
	rootCmd.AddCommand(updateCmd)

	updateCmd.Flags().BoolVar(&options.DryRun, "dry-run", false, "Print-only run: no DB or spreadsheet updates performed")
	updateCmd.Flags().BoolVarP(&options.UseCSVFiles, "csv", "c", false, "Read CSV files; default=false")
}

func update() error {
	var (
		transactions []*models.Transaction
		err          error
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

	registerService := sheets_service.NewRegisterService(sheetsProvider)
	registerService.IncomeSources = config.IncomeSources
	budgetService := sheets_service.NewBudgetService(sheetsProvider)

	bankClient := GetBankingClient()
	updateService := update_service.NewUpdateService(config, options, bankClient, queryRepo)

	columns, err := queryRepo.GetColumns()
	if err != nil {
		return err
	}
	registerSheet, err := registerService.NewRegisterSheet(config, columns)
	if err != nil {
		return err
	}

	fmt.Println("Reading Register...")
	_, err = registerService.ReadRegisterSheet(registerSheet)
	if err != nil {
		return err
	}

	// Todo: account for pagination
	// Due to pagination and no code to account for it, I'm going back about 3 mounths (12 weeks).
	// startDate := config.StartDate
	// endDate := config.EndDate
	startDate := weeksAgo(12)
	endDate := today()
	bankIDs := []string{"wellsfargo", "chase"}
	transactions, err = updateService.FetchTransactions(bankIDs, startDate, endDate)
	if err != nil {
		return err
	}

	if len(transactions) < 1 {
		fmt.Println("No transactions")
		return nil
	}

	if err := writeRawTransactionsCSV(transactions); err != nil {
		return err
	}

	transactions, err = updateService.NormalizeMerchants(transactions)
	if err != nil {
		return err
	}
	if options.Debug {
		printTransactions(transactions)
	}

	transactions = updateService.DedupeAgainstSheet(transactions, registerSheet.KeysMap)

	fmt.Println("Filtering out credit card payments...")
	transactions = bankClient.FilterCreditCardPayments(transactions)

	fmt.Println("Sorting...")
	transactions = bankClient.SortTransactions(transactions)

	if len(transactions) < 1 {
		fmt.Println("No transactions")
		return nil
	}

	printTransactions(transactions)

	if !options.DryRun {
		transactions, err = gatherTransactionInfo(bankClient, queryRepo, transactions)
		if err != nil {
			return err
		}

		transactions, err = updateService.PersistTransactions(transactions)
		if err != nil {
			return err
		}

		if err := updateSpreadsheet(bankClient, registerService, budgetService, queryRepo, transactions); err != nil {
			return err
		}
		return nil
	}
	return nil
}

// gatherTransactionInfo looks thru the transactions to see if there are any merchants
// that are missing an entry in the merchants table. If so, it calls functions to
// print out category columns, prompts the user for a desired merchant name, a category,
// whether to search by exact name or by substring and updates the transactions array.
func gatherTransactionInfo(bankClient *banking.Client, queryRepo repository.QueryRepo, transactions []*models.Transaction) ([]*models.Transaction, error) {
	if needUserInfoForTransactions(transactions) {
		fmt.Println("Info needed...")

		// print the pretty columns so the user has column IDs to enter for each category
		if err := printColumns(queryRepo); err != nil {
			return nil, err
		}

		// get all the necessary input from the user
		var err error
		transactions, err = getUserInput(bankClient, queryRepo, transactions)
		if err != nil {
			return nil, err
		}
	}

	// get notes for any checks or Amazon purchases (They occur often. It's a problem I have)
	transactions = getTransactionNotes(transactions)

	return transactions, nil
}

// needTransactionName loops thru all transactions calling the needsName function
// to determine if it is necessary to prompt the user for input on the transaction
func needUserInfoForTransactions(trans []*models.Transaction) bool {
	return slices.ContainsFunc(trans, needsNameForRegister)
}

// getUserInput is the user interactive piece. It obtains a desired merchant name for
// the Register spreadsheet for the banks name,
func getUserInput(bankClient *banking.Client, queryRepo repository.QueryRepo, trans []*models.Transaction) ([]*models.Transaction, error) {
	var err error
	updated := true

	for updated {
		// get all the input needed from the user, prompting for each item
		updated, trans, err = readFromUser(queryRepo, trans)
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
		trans = bankClient.FormatMerchantNames(trans, matcher, income, columns)
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
func getTransactionNotes(trans []*models.Transaction) []*models.Transaction {
	for i, t := range trans {
		if t.Name == "CHECK" || t.Name == "Amazon" || t.Name == "Amazon Marketplace" {
			fmt.Printf("Source: %s, Name: %s, Date: %s, Amt: $%0.2f\n", t.Source, t.Name, t.Date, t.Amount)
			trans[i].Note = readString("    Note: ")
		}
	}
	return trans
}

// readFromUser loops over every transaction and if it doesn't have a user-specified name
// for the Register spreadsheet it will prompt for the user for input
func readFromUser(queryRepo repository.QueryRepo, trans []*models.Transaction) (bool, []*models.Transaction, error) {
	var err error

	for i, t := range trans {
		if needsNameForRegister(t) {
			fmt.Printf("Source: %s, Date: %s, Amt: $%0.2f\n", t.BankName, t.Date, t.Amount)

			// get the desired bank name
			trans[i].Name = readString("            Name: ")
			var columnID int
			for {
				// get the column ID for the desired category for this merchant
				columnID, err = readInt("       Column ID: ")
				if err != nil {
					fmt.Println(err.Error())
					continue
				}
				break
			}

			// ask for a note for this transaction
			trans[i].Note = readString("           Note: ")

			// ask the user for the desired matching type, either exact or substring which is the default
			matchType := "substring"
			for {
				v := strings.ToLower(strings.TrimSpace(readString("     Match type (e=exact, s=substring) [s]: ")))
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
				if sub := readString(fmt.Sprintf("         Substring [%s]: ", t.BankName)); sub != "" {
					pattern = sub
				}
			}

			// a new substring rule outranks everything already in the
			// table, so its variants can't be shadowed by an older rule
			priority := 0
			if matchType == "substring" {
				maxPrio, err := queryRepo.GetMaxMerchantPriority()
				if err != nil {
					return false, nil, err
				}
				priority = maxPrio + 10
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
func readString(prompt string) string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print(prompt)
	v, _ := reader.ReadString('\n')
	return strings.TrimSuffix(v, "\n")
}

// readInt receives a prompt and reads the number the user enters and returns an integer value
func readInt(prompt string) (int, error) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print(prompt)
	v, _ := reader.ReadString('\n')
	v = strings.TrimSuffix(v, "\n")
	i, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("could not convert %s to an int: %w", v, err)
	}
	return i, nil
}

// writeRawTransactionsCSV dumps the fetched transactions to transactions.csv
// in the finance directory. It runs before merchant normalization and sheet
// dedupe, so the file holds every transaction pulled from the banks,
// including ones already recorded in the register. encoding/csv quotes the
// fields, so bank names containing commas can't shift the columns.
func writeRawTransactionsCSV(transactions []*models.Transaction) error {
	fmt.Println("Writing raw transactions CSV file...")
	file, err := os.Create(config.FinanceDir + "/transactions.csv")
	if err != nil {
		return fmt.Errorf("error creating file %s: %w", config.FinanceDir+"/transactions.csv", err)
	}
	defer file.Close()

	w := stdcsv.NewWriter(file)
	for _, t := range transactions {
		record := []string{
			t.Source, t.Date, t.BankName,
			strconv.FormatFloat(t.Amount, 'f', 2, 64),
			strconv.FormatFloat(t.Deposit, 'f', 2, 64),
			strconv.FormatFloat(t.Withdrawal, 'f', 2, 64),
			strconv.FormatFloat(t.CreditCard, 'f', 2, 64),
		}
		if err := w.Write(record); err != nil {
			return fmt.Errorf("error writing to file: %w", err)
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return fmt.Errorf("error writing to file: %w", err)
	}
	return nil
}

// updateSpreadsheet writes the transactions into the register sheet and
// refreshes the bank balances. Skipped when there is nothing to write or on
// --dry-run runs.
func updateSpreadsheet(bankClient *banking.Client, registerService *sheets_service.RegisterService,
	budgetService *sheets_service.BudgetService, qHandler repository.QueryRepo, transactions []*models.Transaction) error {
	if len(transactions) == 0 || options.DryRun {
		fmt.Println("No updates needed")
		return nil
	}

	registerSheet, err := registerService.NewRegisterSheet(config, nil)
	if err != nil {
		return err
	}
	budgetSheet, err := budgetService.NewBudgetSheet(config)
	if err != nil {
		return err
	}

	fmt.Printf("Reading Budget...\n")
	if _, err := budgetService.ReadBudgetSheet(budgetSheet); err != nil {
		return err
	}

	fmt.Printf("    (%3s) %-12s %-10s %8s %-30s %s\n", "Num", "Source", "Date", "Amount", "Name", "Note")
	fmt.Printf("    (%3s) %-12s %-10s %8s %-30s %s\n", dashes(3), dashes(12), dashes(10), dashes(8), dashes(30), dashes(15))
	for i, r := range transactions {
		fmt.Printf("    (%3d) %-12s %-10s %8.2f %-30s %s\n", i+1, r.Source, r.Date, -1*r.Amount, r.Name, r.Note)
	}

	// add the needed number of rows for transactions
	fmt.Println("Adding rows...")
	if err := registerService.CopyRows(*registerSheet, len(transactions)); err != nil {
		return err
	}

	fmt.Printf("Transaction updates...\n")

	fmt.Printf("Updating spreadsheet...\n")
	columns, err := qHandler.GetColumns()
	if err != nil {
		return err
	}
	transNameToColName, err := qHandler.GetNameMapToColumn()
	if err != nil {
		return err
	}

	if err := registerService.UpdateRows(*registerSheet, *budgetSheet, columns, transNameToColName, transactions); err != nil {
		return err
	}

	lastRowUpdated := registerSheet.SheetCoords.FirstRowToUpdate + int64(len(transactions)*2) + 1
	if _, err := registerService.WriteCell(*registerSheet, sheets_service.LastUpdateCell, time.Now().Format("01/02/2006")); err != nil {
		return err
	}
	if _, err := registerService.WriteCell(*registerSheet, sheets_service.CheckingDeltaCell, fmt.Sprintf("=SUM(%s-I%d)", sheets_service.CheckingBalanceCell, lastRowUpdated)); err != nil {
		return err
	}

	if !options.UseCSVFiles {
		fmt.Println("Getting accounts balances...")
		balances := bankClient.GetBalances([]string{banking.WellsFargoID})
		printBalances(balances)
		fmt.Println("Updating balances...")
		if err := updateBalances(*registerService, *registerSheet, balances); err != nil {
			return err
		}
	}
	return nil
}

func updateBalances(registerService sheets_service.RegisterService, registerSheet sheets_service.RegisterSheet, balances map[string]banking.Balance) error {
	if balances[banking.WellsFargoID].Error == nil {
		if _, err := registerService.WriteCell(registerSheet, sheets_service.CheckingBalanceCell, balances[banking.WellsFargoID].Amount); err != nil {
			return err
		}
	}
	return nil
}

func printBalances(balances map[string]banking.Balance) {
	b := balances[banking.WellsFargoID]
	if b.Error != nil {
		fmt.Printf("    Wells Fargo: balance unavailable: %v\n", b.Error)
		return
	}
	fmt.Printf("    Wells Fargo: $%8.2f\n", b.Amount)
}

func printTransactions(trans []*models.Transaction) {
	fmt.Printf("    (%3s) [%-28s] %-12s %-10s %-8s %2s %-30s %-30s\n", "Num", "Key", "Source", "Date", "Amount", "CI", "Name", "Bank Name")
	for i, t := range trans {
		amt := 0.0
		if t.Source == "WellsFargo" {
			if t.Deposit != 0 {
				amt = t.Deposit
			} else {
				amt = t.Withdrawal
			}
		} else {
			amt = t.CreditPurchase
		}
		fmt.Printf("    (%3d) [%-28s] %-12s %-10s %8.2f %2d %-30s %-30s\n", i+1, t.Key, t.Source, t.Date, amt, t.ColumnIndex, t.Name, t.BankName)
	}
	fmt.Println("")
}

// dashes is a simple helper to print out "count" number of dashes to be printed under a column header
func dashes(count int) string {
	return strings.Repeat("-", count)
}
