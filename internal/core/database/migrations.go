package database

import (
	"context"
	"embed"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Migration represents a database migration.
type Migration struct {
	ID      string
	Name    string
	Content string
}

// Migrator handles database migrations.
type Migrator struct {
	db Client
}

// NewMigrator creates a new migrator.
func NewMigrator(db Client) *Migrator {
	return &Migrator{db: db}
}

// RunMigrations executes all pending migrations.
func (m *Migrator) RunMigrations(ctx context.Context) error {
	if err := m.createMigrationsTable(ctx); err != nil {
		return fmt.Errorf("create migrations table: %w", err)
	}

	migrations, err := m.loadMigrations()
	if err != nil {
		return fmt.Errorf("load migrations: %w", err)
	}

	applied, err := m.getAppliedMigrations(ctx)
	if err != nil {
		return fmt.Errorf("get applied migrations: %w", err)
	}

	pending := m.getPendingMigrations(migrations, applied)

	for _, migration := range pending {
		if err := m.applyMigration(ctx, migration); err != nil {
			return fmt.Errorf("apply migration %s: %w", migration.ID, err)
		}
	}

	return nil
}

// createMigrationsTable creates the migrations tracking table.
func (m *Migrator) createMigrationsTable(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			id VARCHAR(100) PRIMARY KEY,
			name VARCHAR(200) NOT NULL,
			applied_at TIMESTAMP DEFAULT NOW()
		)
	`

	_, err := m.db.Exec(ctx, query)
	return err
}

// loadMigrations loads all migration files.
func (m *Migrator) loadMigrations() ([]Migration, error) {
	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return nil, fmt.Errorf("read migrations dir: %w", err)
	}

	migrations := []Migration{}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		content, err := migrationsFS.ReadFile(filepath.Join("migrations", entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("read migration %s: %w", entry.Name(), err)
		}

		migration := Migration{
			ID:      extractMigrationID(entry.Name()),
			Name:    extractMigrationName(entry.Name()),
			Content: string(content),
		}

		migrations = append(migrations, migration)
	}

	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].ID < migrations[j].ID
	})

	return migrations, nil
}

// getAppliedMigrations gets list of already applied migrations.
func (m *Migrator) getAppliedMigrations(ctx context.Context) (map[string]bool, error) {
	query := "SELECT id FROM schema_migrations"

	rows, err := m.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applied := make(map[string]bool)

	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		applied[id] = true
	}

	return applied, nil
}

// getPendingMigrations filters out already applied migrations.
func (m *Migrator) getPendingMigrations(all []Migration, applied map[string]bool) []Migration {
	pending := []Migration{}

	for _, migration := range all {
		if !applied[migration.ID] {
			pending = append(pending, migration)
		}
	}

	return pending
}

// applyMigration applies a single migration.
func (m *Migrator) applyMigration(ctx context.Context, migration Migration) error {
	_, err := m.db.Exec(ctx, migration.Content)
	if err != nil {
		return fmt.Errorf("execute migration: %w", err)
	}

	query := "INSERT INTO schema_migrations (id, name, applied_at) VALUES ($1, $2, $3)"
	_, err = m.db.Exec(ctx, query, migration.ID, migration.Name, time.Now())
	if err != nil {
		return fmt.Errorf("record migration: %w", err)
	}

	return nil
}

// extractMigrationID extracts ID from filename (e.g., "001_name.sql" -> "001")
func extractMigrationID(filename string) string {
	base := strings.TrimSuffix(filename, ".sql")
	parts := strings.SplitN(base, "_", 2)
	if len(parts) > 0 {
		return parts[0]
	}
	return base
}

// extractMigrationName extracts name from filename.
func extractMigrationName(filename string) string {
	base := strings.TrimSuffix(filename, ".sql")
	parts := strings.SplitN(base, "_", 2)
	if len(parts) > 1 {
		return parts[1]
	}
	return base
}

