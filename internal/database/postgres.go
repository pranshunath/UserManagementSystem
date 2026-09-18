package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	// Register pgx driver for database/sql
	_ "github.com/jackc/pgx/v5/stdlib"
)

// Config holds all parameters required to connect to PostgreSQL.
type Config struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// NewPostgresDB establishes a connection pool to the PostgreSQL database.
// It configures connection pool parameters and performs a PingContext to guarantee
// the database is reachable before returning the *sql.DB instance.
func NewPostgresDB(cfg Config) (*sql.DB, error) {
	// Connection string format (Data Source Name - DSN)
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName, cfg.SSLMode,
	)

	// sql.Open validates the connection string format, but does NOT initiate a network connection yet.
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database handle: %w", err)
	}

	// -------------------------------------------------------------
	// Connection Pool Tuning
	// -------------------------------------------------------------
	// 1. SetMaxOpenConns: Maximum number of open connections to the database.
	// Prevents overwhelming the PostgreSQL server with too many concurrent connections.
	db.SetMaxOpenConns(25)

	// 2. SetMaxIdleConns: Maximum number of connections in the idle connection pool.
	// Keeps hot connections ready so subsequent requests don't pay the TCP handshake penalty.
	db.SetMaxIdleConns(10)

	// 3. SetConnMaxLifetime: Maximum amount of time a connection may be reused.
	// Expired connections are closed gracefully before reuse.
	db.SetConnMaxLifetime(15 * time.Minute)

	// 4. SetConnMaxIdleTime: Maximum amount of time a connection may remain idle before being closed.
	db.SetConnMaxIdleTime(5 * time.Minute)

	// Ping the database with a 5-second timeout to verify live connectivity.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}
