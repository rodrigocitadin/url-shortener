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

	// Golang-Migrate Setup
	sourceDriver, _ := iofs.New(migrations.FS, "migrations")
	dbDriver, _ := postgres.WithInstance(db, &postgres.Config{})
	m, err := migrate.NewWithInstance("iofs", sourceDriver, "postgres", dbDriver)
	if err != nil {
		return fmt.Errorf("error creating migration instance: %w", err)
	}
	defer m.Close()

	if dryRun {
		return printNextMigrationSQL(m, action)
	}

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

	var targetVersion uint
	switch action {
	case "up":
		targetVersion = version + 1
	case "down":
		if version == 0 {
			log.Println("Already at version 0. Nothing to downgrade.")
			return nil
		}
		targetVersion = version
	}

	filename, content, err := findMigrationFile(targetVersion, action)
	if err != nil {
		if action == "up" {
			log.Println(" -> No subsequent migration (Up to date).")
			return nil
		}
		return err
	}

	fmt.Printf("\n--- [DRY RUN] Would execute file: %s ---\n", filename)
	fmt.Println(string(content))
	fmt.Println("------------------------------------------------")

	return nil
}

func findMigrationFile(targetVersion uint, action string) (string, []byte, error) {
	files, err := fs.ReadDir(migrations.FS, "migrations")
	if err != nil {
		return "", nil, err
	}

	suffix := fmt.Sprintf(".%s.sql", action)
	prefix := fmt.Sprintf("%06d", targetVersion)

	for _, file := range files {
		if strings.HasPrefix(file.Name(), prefix) && strings.HasSuffix(file.Name(), suffix) {
			content, err := migrations.FS.ReadFile(path.Join("migrations", file.Name()))
			return file.Name(), content, err
		}
	}

	return "", nil, fmt.Errorf("migration file version %d not found", targetVersion)
}
