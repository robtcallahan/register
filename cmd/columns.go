package cmd

import (
	"fmt"
	"strings"

	"register/api/providers/sheets_provider"
	"register/api/services/sheets_service"
	"register/pkg/driver"
	"register/pkg/handler"
	"register/pkg/models"

	"github.com/spf13/cobra"
)


var columnsCmd = &cobra.Command{
	Use:   "columns",
	Short: "Manage register category columns in the DB and the sheet",
	Long: `List, add, rename, and delete register category columns. The columns
table is the source of truth; add and delete keep the Register tab in step
(insert/remove the sheet column, fill the running-total formulas).`,
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
	addColor  string
	addAfter  string
	deleteYes bool
)

func init() {
	rootCmd.AddCommand(columnsCmd)
	columnsCmd.AddCommand(columnsListCmd)
	columnsCmd.AddCommand(columnsAddCmd)
	columnsCmd.AddCommand(columnsRenameCmd)
	columnsCmd.AddCommand(columnsDeleteCmd)
	columnsCmd.AddCommand(columnsCheckCmd)

	columnsAddCmd.Flags().StringVar(&addColor, "color", "", "column color: green, yellow, or blue (required)")
	columnsAddCmd.Flags().StringVar(&addAfter, "after", "", "insert after this column (default: at the end)")
	columnsDeleteCmd.Flags().BoolVar(&deleteYes, "yes", false, "skip the confirmation prompt")
	_ = columnsAddCmd.MarkFlagRequired("color")
}

func columnsQuery() (*handler.Query, error) {
	conn, err := driver.ConnectSQL(&driver.ConnectParams{
		DBType: driver.DBType(config.DB.Type),
		Host:   config.DB.Host,
		Port:   config.DB.Port,
		DBName: config.DB.Name,
		User:   config.DB.Username,
		Pass:   config.DB.Password,
	})
	if err != nil {
		return nil, err
	}
	return handler.NewQueryHandler(conn), nil
}

func columnsSheetService(q *handler.Query) (*sheets_service.SheetsService, error) {
	columns, err := q.GetColumns()
	if err != nil {
		return nil, err
	}
	provider, err := sheets_provider.New(options.SpreadsheetID, config)
	if err != nil {
		return nil, err
	}
	ss := sheets_service.New(provider)
	if err := ss.NewRegisterSheet(config, columns); err != nil {
		return nil, err
	}
	return ss, nil
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
	case "green", "yellow", "blue":
	default:
		return fmt.Errorf("--color must be green, yellow, or blue (black is reserved for the payoff columns)")
	}
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("column name must not be empty")
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

	ss, err := columnsSheetService(q)
	if err != nil {
		return err
	}
	if err := ss.InsertRegisterColumn(newIndex, name); err != nil {
		return fmt.Errorf("%q is in the DB at %s (%d) but the sheet update failed: %w",
			name, set.LetterFor(newIndex), newIndex, err)
	}

	fmt.Printf("Added %q at column %s (index %d)\n", name, set.LetterFor(newIndex), newIndex)
	return nil
}

func renameColumn(oldName, newName string) error {
	if strings.TrimSpace(newName) == "" {
		return fmt.Errorf("new name must not be empty")
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

	ss, err := columnsSheetService(q)
	if err != nil {
		return err
	}
	if err := ss.RenameRegisterColumnHeader(index, newName); err != nil {
		return fmt.Errorf("%q renamed in the DB but the sheet header update failed: %w", oldName, err)
	}

	fmt.Printf("Renamed %q to %q (column %s)\n", oldName, newName, set.LetterFor(index))
	return nil
}

func deleteColumn(name string) error {
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

	merchants, err := q.GetMerchantsByColumn(col.ID)
	if err != nil {
		return err
	}
	if len(merchants) > 0 {
		fmt.Printf("These merchant rules point at %q and will be deleted with it:\n", name)
		for _, m := range merchants {
			fmt.Printf("  [%d] %-35s -> %s\n", m.ID, m.BankName, m.Name)
		}
		if !confirm(fmt.Sprintf("Delete column %q and %d merchant rule(s)? [y/N] ", name, len(merchants))) {
			fmt.Println("Cancelled.")
			return nil
		}
	} else if !deleteYes {
		if !confirm(fmt.Sprintf("Delete column %q (%s, index %d)? Its sheet column and accumulated amounts go too. [y/N] ",
			name, set.LetterFor(index), index)) {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	if err := q.DeleteColumn(col.ID); err != nil {
		return err
	}

	ss, err := columnsSheetService(q)
	if err != nil {
		return err
	}
	if err := ss.DeleteRegisterColumn(index); err != nil {
		return fmt.Errorf("%q is deleted from the DB but the sheet update failed: %w", name, err)
	}

	fmt.Printf("Deleted %q (column %s, index %d)\n", name, set.LetterFor(index), index)
	return nil
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
	ss, err := columnsSheetService(q)
	if err != nil {
		return err
	}
	problems, err := ss.CheckRegisterColumns(columns)
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

func confirm(prompt string) bool {
	return strings.EqualFold(strings.TrimSpace(readString(prompt)), "y")
}
