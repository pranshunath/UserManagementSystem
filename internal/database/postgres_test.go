package database

import (
	"context"
	"testing"
	"time"
)

func TestPostgresConnection(t *testing.T) {
	cfg := Config{
		Host:     "localhost",
		Port:     "5432",
		User:     "postgres",
		Password: "postgrespassword",
		DBName:   "usermanagement",
		SSLMode:  "disable",
	}

	db, err := NewPostgresDB(cfg)
	if err != nil {
		t.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Verify ping
	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("Ping failed: %v", err)
	}

	// Verify tables exist
	var count int
	query := `SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public' AND table_name IN ('users', 'departments', 'roles', 'user_roles')`
	if err := db.QueryRowContext(ctx, query).Scan(&count); err != nil {
		t.Fatalf("Failed to query schema: %v", err)
	}

	if count != 4 {
		t.Fatalf("Expected 4 tables, got %d", count)
	}

	t.Logf(" Successfully connected to PostgreSQL! Found %d required tables.", count)
}
