package main

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/spf13/cobra"

	"github.com/z3spinner/go-stop/db/migrations"
)

func getMigrator() (*migrate.Migrate, error) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, fmt.Errorf("DATABASE_URL not set")
	}
	// lib/pq (used by golang-migrate) rejects sslmode=prefer; replace with require.
	dbURL = strings.ReplaceAll(dbURL, "sslmode=prefer", "sslmode=require")
	d, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return nil, fmt.Errorf("migrations source: %w", err)
	}
	return migrate.NewWithSourceInstance("iofs", d, dbURL)
}

var rootCmd = &cobra.Command{
	Use:   "migratedb",
	Short: "Database migration tool for go-stop",
}

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Apply all pending migrations",
	RunE: func(cmd *cobra.Command, args []string) error {
		m, err := getMigrator()
		if err != nil {
			return err
		}
		if err = m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			return err
		}
		fmt.Println("Migration complete")
		return nil
	},
}

var downCmd = &cobra.Command{
	Use:   "down [N]",
	Short: "Roll back N migrations (default: 1)",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		m, err := getMigrator()
		if err != nil {
			return err
		}
		steps := 1
		if len(args) == 1 {
			steps, err = strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid step count: %w", err)
			}
		}
		return m.Steps(-steps)
	},
}

var forceCmd = &cobra.Command{
	Use:   "force VERSION",
	Short: "Force-set the migration version (fixes dirty state)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		version, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid version: %w", err)
		}
		m, err := getMigrator()
		if err != nil {
			return err
		}
		return m.Force(version)
	},
}

var dropCmd = &cobra.Command{
	Use:   "drop",
	Short: "Drop everything in the database",
	RunE: func(cmd *cobra.Command, args []string) error {
		m, err := getMigrator()
		if err != nil {
			return err
		}
		if err = m.Drop(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			return err
		}
		fmt.Println("Done")
		return nil
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print current migration version",
	RunE: func(cmd *cobra.Command, args []string) error {
		m, err := getMigrator()
		if err != nil {
			return err
		}
		version, dirty, err := m.Version()
		if err != nil {
			return err
		}
		fmt.Printf("version: %d, dirty: %v\n", version, dirty)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(upCmd, downCmd, forceCmd, dropCmd, versionCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
