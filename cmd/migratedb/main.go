package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/cobra"

	"github.com/z3spinner/go-stop/internal/infrastructure/postgres"
	"github.com/z3spinner/go-stop/internal/infrastructure/postgres/sqlc/migrations"
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

// encryptPhonesCmd re-encrypts all phone numbers in the database using the
// current PHONE_ENCRYPTION_KEY. Rows that are already correctly encrypted are
// left unchanged (deterministic encryption means encrypt(decrypt(x)) == x).
// Safe to run multiple times.
var encryptPhonesCmd = &cobra.Command{
	Use:   "encrypt-phones",
	Short: "Encrypt all plaintext phone numbers using PHONE_ENCRYPTION_KEY",
	RunE: func(cmd *cobra.Command, args []string) error {
		if os.Getenv("PHONE_ENCRYPTION_KEY") == "" {
			return fmt.Errorf("PHONE_ENCRYPTION_KEY is not set")
		}
		crypto, err := postgres.NewPhoneCryptoFromEnv()
		if err != nil {
			return err
		}

		dbURL := os.Getenv("DATABASE_URL")
		if dbURL == "" {
			return fmt.Errorf("DATABASE_URL not set")
		}
		dbURL = strings.ReplaceAll(dbURL, "sslmode=prefer", "sslmode=require")
		pool, err := pgxpool.New(context.Background(), dbURL)
		if err != nil {
			return fmt.Errorf("connect: %w", err)
		}
		defer pool.Close()

		total := 0
		for _, t := range []struct{ table, idCol, phoneCol string }{
			{"rides", "id", "phone"},
			{"requests", "id", "phone"},
			{"interests", "id", "searcher_phone"},
			{"subscriptions", "id", "phone"},
		} {
			n, err := reencryptColumn(context.Background(), pool, crypto, t.table, t.idCol, t.phoneCol)
			if err != nil {
				return fmt.Errorf("%s.%s: %w", t.table, t.phoneCol, err)
			}
			fmt.Printf("  %-16s %d rows updated\n", t.table+":", n)
			total += n
		}
		fmt.Printf("Done — %d phone(s) encrypted\n", total)
		return nil
	},
}

// reencryptColumn fetches every phone value in the table and encrypts any
// plaintext rows. Rows already correctly encrypted (encrypt(decrypt(x))==x)
// are skipped. Returns an error if a value looks encrypted but can't be
// decrypted with the current key — this indicates a key mismatch.
func reencryptColumn(ctx context.Context, pool *pgxpool.Pool, crypto *postgres.PhoneCrypto, table, idCol, phoneCol string) (int, error) {
	rows, err := pool.Query(ctx,
		"SELECT "+idCol+", "+phoneCol+" FROM "+table+" WHERE "+phoneCol+" IS NOT NULL")
	if err != nil {
		return 0, err
	}
	type row struct{ id, phone string }
	var all []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.id, &r.phone); err != nil {
			rows.Close()
			return 0, err
		}
		all = append(all, r)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}

	updated := 0
	for _, r := range all {
		plain, isLegacy, err := crypto.DecryptOrDetect(r.phone)
		if err != nil {
			// Value is ciphertext but decryption failed → wrong key
			return updated, fmt.Errorf("row %q: value appears encrypted but cannot be decrypted — wrong PHONE_ENCRYPTION_KEY?", r.id)
		}
		if !isLegacy {
			continue // already correctly encrypted with this key
		}
		enc, err := crypto.Encrypt(plain)
		if err != nil {
			return updated, fmt.Errorf("encrypt %q: %w", r.id, err)
		}
		_, err = pool.Exec(ctx,
			"UPDATE "+table+" SET "+phoneCol+" = $1 WHERE "+idCol+" = $2",
			enc, r.id)
		if err != nil {
			return updated, fmt.Errorf("update %q: %w", r.id, err)
		}
		updated++
	}
	return updated, nil
}

func init() {
	rootCmd.AddCommand(upCmd, downCmd, forceCmd, dropCmd, versionCmd, encryptPhonesCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
