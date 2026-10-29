package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"

	"register/api/providers/sheets_provider"
	"register/api/services/sheets_service"
	"register/pkg/models"
	"register/pkg/repository"
	"register/pkg/repository/mysql"

	"github.com/spf13/cobra"
)

var columnsCmd = &cobra.Command{
	Use:   "columns",
	Short: "Manage register category columns in the DB and the sheet",
	Long: `List, add, rename, and delete register category columns. The columns
table is the source of truth; add and delete keep the Register tab in step
(insert/remove the sheet column, fill the running-total formulas) and keep
the Budget tab in step too (rename moves the budget row, add creates one,
delete removes it, move repositions it inside its color group).`,
}

var columnsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List register columns from the DB",
	RunE: func(cmd *cobra.Command, args []string) error {
		return listColumns()
	},
}

var columnsAddCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Add a category column after another column",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return addColumn(args[0])
	},
}

var columnsRenameCmd = &cobra.Command{
	Use:   "rename <old-name> <new-name>",
	Short: "Rename a register column (DB + sheet header)",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return renameColumn(args[0], args[1])
	},
}

var columnsMoveCmd = &cobra.Command{
	Use:   "move <name>",
	Short: "Move a register column to after another column",
	Long: `Reposition a category column (DB, Register tab, and its Budget row
move together). The destination must stay inside the column's color
group — crossing a group boundary is refused, so the Budget tab's
group totals keep summing the same categories.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return moveColumn(args[0], moveAfter)
	},
}

var columnsCheckCmd = &cobra.Command{
	Use:   "check",
	Short: "Report drift between the columns table and the Register sheet (read-only)",
	RunE: func(cmd *cobra.Command, args []string) error {
		return checkColumns()
	},
}

var columnsDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a register column, its sheet column, and its merchant rules",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return deleteColumn(args[0])
	},
}

var (
	addColor    string
	addAfter    string
	addNoBudget bool
	deleteYes   bool
	moveAfter   string
)

func init() {
	rootCmd.AddCommand(columnsCmd)
	columnsCmd.AddCommand(columnsListCmd)
	columnsCmd.AddCommand(columnsAddCmd)
	columnsCmd.AddCommand(columnsRenameCmd)
	columnsCmd.AddCommand(columnsDeleteCmd)
	columnsCmd.AddCommand(columnsMoveCmd)
	columnsCmd.AddCommand(columnsCheckCmd)

	columnsAddCmd.Flags().StringVar(&addColor, "color", "", "column color: green, yellow, or blue (required)")
	columnsAddCmd.Flags().StringVar(&addAfter, "after", "", "insert after this column (default: at the end)")
	columnsAddCmd.Flags().BoolVar(&addNoBudget, "no-budget", false, "do not create a Budget tab row for the new category")
	columnsDeleteCmd.Flags().BoolVar(&deleteYes, "yes", false, "skip the confirmation prompt")
	columnsMoveCmd.Flags().StringVar(&moveAfter, "after", "", "move after this column (default: to the end)")
	_ = columnsAddCmd.MarkFlagRequired("color")
}

func columnsQuery() (repository.QueryRepo, error) {
	conn, err := repository.Connect(&config.DB)
	if err != nil {
		return nil, err
	}
	return mysql.NewMySQLQueryRepo(conn), nil
}

func listColumns() error {
	q, err := columnsQuery()
	if err != nil {
		return err
	}
	columns, err := q.GetColumns()
	if err != nil {
		return err
	}
	set := models.NewColumnSet(columns)
	fmt.Printf("%4s  %-4s  %-30s %-8s %-9s %s\n", "IDX", "COL", "NAME", "COLOR", "CATEGORY", "ID")
	for _, col := range columns {
		category := "no"
		if col.IsCategory() {
			category = "yes"
		}
		fmt.Printf("%4d  %-4s  %-30s %-8s %-9s %d\n",
			col.ColumnIndex, set.LetterFor(col.ColumnIndex), col.Name, col.Color, category, col.ID)
	}
	return nil
}

func addColumn(name string) error {
	switch addColor {
	case models.ColorGreen, models.ColorYellow, models.ColorBlue:
	default:
		return errors.New("--color must be green, yellow, or blue (grey is reserved for the payoff columns)")
	}
	if strings.TrimSpace(name) == "" {
		return errors.New("column name must not be empty")
	}

	q, err := columnsQuery()
	if err != nil {
		return err
	}
	columns, err := q.GetColumns()
	if err != nil {
		return err
	}
	set := models.NewColumnSet(columns)
	if _, ok := set.IndexFor(name); ok {
		return fmt.Errorf("a column named %q already exists", name)
	}

	newIndex := 0
	if end, ok := set.End(); ok {
		newIndex = end.ColumnIndex + 1
	}
	if addAfter != "" {
		afterIndex, ok := set.IndexFor(addAfter)
		if !ok {
			return fmt.Errorf("no column named %q", addAfter)
		}
		newIndex = afterIndex + 1
	}
	if newIndex < models.FirstCategoryIndex {
		return fmt.Errorf("cannot insert at index %d: indexes 1-%d are the fixed register block",
			newIndex, models.FirstCategoryIndex-1)
	}

	col := &models.Column{Name: name, Color: addColor, ColumnIndex: newIndex}
	if err := q.AddColumn(col); err != nil {
		return err
	}

	provider, err := sheets_provider.New(options.SpreadsheetID, config)
	if err != nil {
		return err
	}
	rs := sheets_service.NewRegisterService(provider)
	registerSheet, err := rs.NewRegisterSheet(config, nil)
	if err != nil {
		return err
	}
	bs := sheets_service.NewBudgetService(provider)
	budgetSheet, err := bs.NewBudgetSheet(*config)
	if err != nil {
		return err
	}

	if err := rs.InsertRegisterColumn(*registerSheet, newIndex, name); err != nil {
		return fmt.Errorf("%q is in the DB at %s (%d) but the sheet update failed: %w",
			name, set.LetterFor(newIndex), newIndex, err)
	}
	if !addNoBudget {
		if err := bs.InsertBudgetCategory(*budgetSheet, name, addColor); err != nil {
			return fmt.Errorf("%q added at %s (%d) but the Budget row update failed: %w",
				name, set.LetterFor(newIndex), newIndex, err)
		}
	}

	fmt.Printf("Added %q at column %s (index %d)\n", name, set.LetterFor(newIndex), newIndex)
	return nil
}

func renameColumn(oldName, newName string) error {
	if strings.TrimSpace(newName) == "" {
		return errors.New("new name must not be empty")
	}

	q, err := columnsQuery()
	if err != nil {
		return err
	}
	columns, err := q.GetColumns()
	if err != nil {
		return err
	}
	set := models.NewColumnSet(columns)
	index, ok := set.IndexFor(oldName)
	if !ok {
		return fmt.Errorf("no column named %q", oldName)
	}
	if _, exists := set.IndexFor(newName); exists && newName != oldName {
		return fmt.Errorf("a column named %q already exists", newName)
	}
	col, _ := set.ByIndex(index)

	if err := q.RenameColumn(col.ID, newName); err != nil {
		return err
	}

	provider, err := sheets_provider.New(options.SpreadsheetID, config)
	if err != nil {
		return err
	}
	rs := sheets_service.NewRegisterService(provider)
	registerSheet, err := rs.NewRegisterSheet(config, nil)
	if err != nil {
		return err
	}
	bs := sheets_service.NewBudgetService(provider)
	budgetSheet, err := bs.NewBudgetSheet(*config)
	if err != nil {
		return err
	}

	if err := rs.RenameRegisterColumnHeader(*registerSheet, index, newName); err != nil {
		return fmt.Errorf("%q renamed in the DB but the sheet header update failed: %w", oldName, err)
	}
	budgetRenamed, err := bs.RenameBudgetCategory(*budgetSheet, oldName, newName)
	if err != nil {
		return fmt.Errorf("%q renamed but the Budget row update failed: %w", oldName, err)
	}

	fmt.Printf("Renamed %q to %q (column %s)\n", oldName, newName, set.LetterFor(index))
	if budgetRenamed {
		fmt.Println("Renamed the Budget row too.")
	}
	return nil
}

func deleteColumn(name string) error {
	var reader = bufio.NewReader(os.Stdin)

	q, err := columnsQuery()
	if err != nil {
		return err
	}
	columns, err := q.GetColumns()
	if err != nil {
		return err
	}
	set := models.NewColumnSet(columns)
	index, ok := set.IndexFor(name)
	if !ok {
		return fmt.Errorf("no column named %q", name)
	}
	if index < models.FirstCategoryIndex {
		return fmt.Errorf("cannot delete %q: indexes 1-%d are the fixed register block",
			name, models.FirstCategoryIndex-1)
	}
	col, _ := set.ByIndex(index)

	provider, err := sheets_provider.New(options.SpreadsheetID, config)
	if err != nil {
		return err
	}
	rs := sheets_service.NewRegisterService(provider)
	registerSheet, err := rs.NewRegisterSheet(config, nil)
	if err != nil {
		return err
	}
	bs := sheets_service.NewBudgetService(provider)
	budgetSheet, err := bs.NewBudgetSheet(*config)
	if err != nil {
		return err
	}

	budgetEntry, hasBudget, err := bs.BudgetEntryFor(*budgetSheet, name)
	if err != nil {
		return err
	}
	if hasBudget {
		fmt.Printf("%q has a Budget row (Weekly $%.2f, Monthly $%.2f) that will be deleted with it.\n",
			name, budgetEntry.Weekly, budgetEntry.Monthly)
	}

	merchants, err := q.GetMerchantsByColumn(col.ID)
	if err != nil {
		return err
	}
	if len(merchants) > 0 {
		fmt.Printf("These merchant rules point at %q and will be deleted with it:\n", name)
		for _, m := range merchants {
			fmt.Printf("  [%d] %-35s -> %s\n", m.ID, m.BankName, m.Name)
		}
		if !confirm(reader, fmt.Sprintf("Delete column %q and %d merchant rule(s)? [y/N] ", name, len(merchants))) {
			fmt.Println("Cancelled.")
			return nil
		}
	} else if !deleteYes {
		if !confirm(reader, fmt.Sprintf("Delete column %q (%s, index %d)? Its sheet column and accumulated amounts go too. [y/N] ",
			name, set.LetterFor(index), index)) {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	if err := q.DeleteColumn(col.ID); err != nil {
		return err
	}

	if err := rs.DeleteRegisterColumn(*registerSheet, index); err != nil {
		return fmt.Errorf("%q is deleted from the DB but the sheet update failed: %w", name, err)
	}
	if hasBudget {
		if _, err := bs.DeleteBudgetCategory(*budgetSheet, name); err != nil {
			return fmt.Errorf("%q is deleted but its Budget row removal failed: %w", name, err)
		}
	}

	fmt.Printf("Deleted %q (column %s, index %d)\n", name, set.LetterFor(index), index)
	return nil
}

// moveColumn moves the named register column to a new position (after
// the column named by the --after flag, or to the end), keeping three
// stores in sync: the category's Budget row moves to mirror the new
// register order, the DB column_index values shift via MoveColumn,
// and the Register tab's dimension moves via MoveRegisterColumn.
// The Budget move happens first, so its failure leaves everything
// untouched; a sheet failure after the DB move is reported as such.
func moveColumn(name, moveAfter string) error {
	q, err := columnsQuery()
	if err != nil {
		return err
	}
	columns, err := q.GetColumns()
	if err != nil {
		return err
	}
	set := models.NewColumnSet(columns)
	from, ok := set.IndexFor(name)
	if !ok {
		return fmt.Errorf("no column named %q", name)
	}
	after := 0
	if moveAfter != "" {
		idx, ok := set.IndexFor(moveAfter)
		if !ok {
			return fmt.Errorf("no column named %q", moveAfter)
		}
		after = idx
	}
	to, changed, err := sheets_service.PlanColumnMove(columns, from, after)
	if err != nil {
		return err
	}
	col, _ := set.ByIndex(from)
	if !changed {
		fmt.Printf("%q is already at column %s (index %d)\n", name, set.LetterFor(from), from)
		return nil
	}

	provider, err := sheets_provider.New(options.SpreadsheetID, config)
	if err != nil {
		return err
	}
	rs := sheets_service.NewRegisterService(provider)
	registerSheet, err := rs.NewRegisterSheet(config, nil)
	if err != nil {
		return err
	}
	bs := sheets_service.NewBudgetService(provider)
	budgetSheet, err := bs.NewBudgetSheet(*config)
	if err != nil {
		return err
	}

	budgetRowsMap, err := bs.BudgetRows(*budgetSheet)
	if err != nil {
		return err
	}

	// Budget mirror: the category's row stays behind the nearest
	// same-group category that precedes it in the NEW register order
	// (or ahead of the nearest one that follows). Register-only
	// columns (Credit Cards, AppleCard, Taxes) have no row to move,
	// and moving one never rearranges anyone's budget.
	budgetNote := ""
	if _, hasBudget := budgetRowsMap[name]; hasBudget {
		final := sheets_service.ApplyColumnMove(columns, from, to)
		before, following := findBudgetNeighbor(final, name, col.Color, budgetRowsMap)
		switch {
		case before != "":
			if err := bs.MoveBudgetCategoryAfter(*budgetSheet, name, before); err != nil {
				return fmt.Errorf("budget row move for %q failed (nothing else changed yet): %w", name, err)
			}
			budgetNote = " Budget row moved with it."
		case following != "":
			if err := bs.MoveBudgetCategoryBefore(*budgetSheet, name, following); err != nil {
				return fmt.Errorf("budget row move for %q failed (nothing else changed yet): %w", name, err)
			}
			budgetNote = " Budget row moved with it."
		}
	}

	if err := q.MoveColumn(col.ID, to); err != nil {
		return err
	}
	// MoveDimension counts the destination in pre-move coordinates:
	// "after column N" is destination N, and "to the end" is the old
	// last index when moving left-to-right past it.
	destAfter := after
	if after == 0 {
		destAfter = len(columns)
	}
	if err := rs.MoveRegisterColumn(*registerSheet, from, destAfter); err != nil {
		return fmt.Errorf("%q moved to %s (%d) in the DB but the sheet move failed: %w",
			name, set.LetterFor(to), to, err)
	}

	fmt.Printf("Moved %q to column %s (index %d).%s\n", name, set.LetterFor(to), to, budgetNote)
	return nil
}

// findBudgetNeighbor scans the new register order for the nearest
// same-color category with a Budget row on either side of the moved
// column. before is the nearest one ahead of it, following the nearest
// one after it; either may be empty when no such neighbor exists.
func findBudgetNeighbor(final []models.Column, name, color string, budgetRowsMap map[string]int) (before, following string) {
	seen := false
	for _, c := range final {
		if c.Name == name {
			seen = true
			continue
		}
		if c.Color != color {
			continue
		}
		if _, hasBudget := budgetRowsMap[c.Name]; !hasBudget {
			continue
		}
		if !seen {
			before = c.Name
		} else if following == "" {
			following = c.Name
		}
	}
	return before, following
}

func checkColumns() error {
	q, err := columnsQuery()
	if err != nil {
		return err
	}
	columns, err := q.GetColumns()
	if err != nil {
		return err
	}

	provider, err := sheets_provider.New(options.SpreadsheetID, config)
	if err != nil {
		return err
	}
	rs := sheets_service.NewRegisterService(provider)
	registerSheet, err := rs.NewRegisterSheet(config, nil)
	if err != nil {
		return err
	}

	problems, err := rs.CheckRegisterColumns(*registerSheet, columns)
	if err != nil {
		return err
	}
	if len(problems) == 0 {
		fmt.Printf("OK: %d columns, DB and sheet agree\n", len(columns))
		return nil
	}
	fmt.Println("Column drift detected:")
	for _, p := range problems {
		fmt.Println("  - " + p)
	}
	return fmt.Errorf("%d column problem(s) found", len(problems))
}

func confirm(reader *bufio.Reader, prompt string) bool {
	return strings.EqualFold(strings.TrimSpace(readString(reader, prompt)), "y")
}
