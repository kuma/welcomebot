package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

// Client provides database operations.
type Client interface {
	Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row
	Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	Close() error
	Ping(ctx context.Context) error
}

// Config contains database connection configuration.
type Config struct {
	Host     string
	Port     string
	User     string
	Password string
	Database string
	SSLMode  string
}

// DefaultConfig returns default database configuration.
func DefaultConfig() Config {
	return Config{
		Host:    "localhost",
		Port:    "5432",
		SSLMode: "disable",
	}
}

// postgresClient implements Client using PostgreSQL.
type postgresClient struct {
	db *sql.DB
}

// New creates a new database client with the given configuration.
func New(cfg Config) (Client, error) {
	connStr := buildConnectionString(cfg)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return &postgresClient{db: db}, nil
}

// Query executes a query that returns rows.
func (c *postgresClient) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	rows, err := c.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	return rows, nil
}

// QueryRow executes a query that returns at most one row.
func (c *postgresClient) QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return c.db.QueryRowContext(ctx, query, args...)
}

// Exec executes a query without returning any rows.
func (c *postgresClient) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	result, err := c.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("exec: %w", err)
	}
	return result, nil
}

// Close closes the database connection.
func (c *postgresClient) Close() error {
	if err := c.db.Close(); err != nil {
		return fmt.Errorf("close database: %w", err)
	}
	return nil
}

// Ping verifies the database connection is alive.
func (c *postgresClient) Ping(ctx context.Context) error {
	if err := c.db.PingContext(ctx); err != nil {
		return fmt.Errorf("ping: %w", err)
	}
	return nil
}

// buildConnectionString creates a PostgreSQL connection string.
func buildConnectionString(cfg Config) string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host,
		cfg.Port,
		cfg.User,
		cfg.Password,
		cfg.Database,
		cfg.SSLMode,
	)
}

