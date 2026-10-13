package cmd

import (
	"fmt"

	"register/pkg/repository"

	"github.com/pressly/goose/v3"
	"github.com/spf13/cobra"
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Run database migrations (up, down, status, redo)",
	Long: `Run goose database migrations against the configured MySQL database.

Subcommands:
  up              Apply all pending migrations
  down            Roll back the most recent migration
  down-to VERSION Roll back to a specific version
  redo            Roll back and re-apply the most recent migration
  status          Show applied vs pending migrations`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runMigration(args[0], args[1:])
	},
}

func init() {
	rootCmd.AddCommand(migrateCmd)
}

func runMigration(action string, rest []string) error {
	// Connect via the existing repository helper
	db, err := repository.Connect(&config.DB)
	if err != nil {
		return fmt.Errorf("unable to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("unable to get sql.DB: %w", err)
	}
	defer sqlDB.Close()

	if err := goose.SetDialect("mysql"); err != nil {
		return fmt.Errorf("unable to set goose dialect: %w", err)
	}

	migrationsDir := "migrations"

	switch action {
	case "up":
		return goose.Up(sqlDB, migrationsDir)
	case "down":
		return goose.Down(sqlDB, migrationsDir)
	case "down-to":
		if len(rest) == 0 {
			return fmt.Errorf("down-to requires a version argument")
		}
		var version int64
		if _, err := fmt.Sscanf(rest[0], "%d", &version); err != nil {
			return fmt.Errorf("invalid version %q: %w", rest[0], err)
		}
		return goose.DownTo(sqlDB, migrationsDir, version)
	case "redo":
		return goose.Redo(sqlDB, migrationsDir)
	case "status":
		return goose.Status(sqlDB, migrationsDir)
	default:
		return fmt.Errorf("unknown migrate action %q (want up, down, down-to, redo, status)", action)
	}
}
