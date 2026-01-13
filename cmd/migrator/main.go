package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/lib/pq"
	"github.com/rodrigocitadin/url-shortener/migrations"
)

func main() {
	var (
		action    string
		shardsEnv string
		dryRun    bool
	)

	flag.StringVar(&action, "action", "up", "Action: up, down")
	flag.StringVar(&shardsEnv, "shards", "", "Comma-separated DSNs")
	flag.BoolVar(&dryRun, "dry-run", false, "If true, only shows the SQL that would be executed")
	flag.Parse()

	if shardsEnv == "" {
		shardsEnv = os.Getenv("SHARD_DSNS")
	}
	if shardsEnv == "" {
		log.Fatal("Error: No shards configured.")
	}

	dsnList := strings.Split(shardsEnv, ",")

	if dryRun {
		log.Println("DRY RUN MODE ENABLED: No changes will be made to the database.")
	}

	for i, dsn := range dsnList {
		dsn = strings.TrimSpace(dsn)
		log.Printf("--- Processing Shard #%d ---", i)

		if err := runMigration(dsn, action, dryRun); err != nil {
			log.Fatalf("Error on Shard #%d: %v", i, err)
		}
	}
	log.Println("Process finished.")
}

func runMigration(dsn, action string, dryRun bool) error {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("ping db: %w", err)
	}

	sourceDriver, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return fmt.Errorf("error initializing source driver: %w", err)
	}

	dbDriver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("error initializing db driver: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", sourceDriver, "postgres", dbDriver)
	if err != nil {
		return fmt.Errorf("error creating migration instance: %w", err)
	}
	defer m.Close()

	// 1. DRY RUN Logic
	if dryRun {
		return printNextMigrationSQL(m, action)
	}

	// 2. Real Execution Logic
	log.Printf("Executing %s...", strings.ToUpper(action))
	switch action {
	case "up":
		err = m.Up()
	case "down":
		err = m.Down()
	default:
		return fmt.Errorf("invalid action: %s", action)
	}

	if err != nil && err != migrate.ErrNoChange {
		return err
	}
	if err == migrate.ErrNoChange {
		log.Println(" -> Nothing to do (Up to date).")
	} else {
		log.Println(" -> Success!")
	}
	return nil
}

// printNextMigrationSQL determines which file would be executed and prints its content
func printNextMigrationSQL(m *migrate.Migrate, action string) error {
	version, dirty, err := m.Version()

	if err == migrate.ErrNilVersion {
		version = 0
		dirty = false
	} else if err != nil {
		return fmt.Errorf("failed to read version: %w", err)
	}

	if dirty {
		log.Printf("ALERT: Database is in 'Dirty' state (version %d). Manual fix required.", version)
		return nil
	}

	log.Printf("Current Version: %d", version)

	filename, content, err := findMigrationFile(version, action)
	if err != nil {
		if action == "up" {
			log.Println(" -> No subsequent migration found (Up to date).")
			return nil
		}
		return err
	}

	fmt.Printf("\n--- [DRY RUN] Would execute file: %s ---\n", filename)
	fmt.Println(string(content))
	fmt.Println("------------------------------------------------")

	return nil
}

// findMigrationFile scans the migrations folder looking for the target file
// It supports Timestamps by listing all files and finding the next logical one.
func findMigrationFile(currentVersion uint, action string) (string, []byte, error) {
	files, err := fs.ReadDir(migrations.FS, ".")
	if err != nil {
		return "", nil, err
	}

	type migrationFile struct {
		Version  uint64
		Name     string
		Fullpath string
	}

	var validFiles []migrationFile
	suffix := fmt.Sprintf(".%s.sql", action)

	for _, file := range files {
		if strings.HasSuffix(file.Name(), suffix) {
			parts := strings.Split(file.Name(), "_")
			if len(parts) > 0 {
				v, err := strconv.ParseUint(parts[0], 10, 64)
				if err == nil {
					validFiles = append(validFiles, migrationFile{
						Version:  v,
						Name:     file.Name(),
						Fullpath: path.Join(".", file.Name()),
					})
				}
			}
		}
	}

	sort.Slice(validFiles, func(i, j int) bool {
		return validFiles[i].Version < validFiles[j].Version
	})

	for _, mf := range validFiles {
		switch action {
		case "up":
			if mf.Version > uint64(currentVersion) {
				content, err := migrations.FS.ReadFile(mf.Fullpath)
				return mf.Name, content, err
			}
		case "down":
			if mf.Version == uint64(currentVersion) {
				content, err := migrations.FS.ReadFile(mf.Fullpath)
				return mf.Name, content, err
			}
		}
	}

	return "", nil, fmt.Errorf("migration file not found for current version %d with action %s", currentVersion, action)
}
