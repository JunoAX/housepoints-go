package database

import (
	"context"
	"fmt"
	"time"

	"github.com/JunoAX/housepoints-go/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PlatformDB handles connections to the platform routing database
type PlatformDB struct {
	pool *pgxpool.Pool
}

// NewPlatformDB creates a new platform database connection
func NewPlatformDB(ctx context.Context, connString string) (*PlatformDB, error) {
	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse platform DB config: %w", err)
	}

	// Connection pool settings
	config.MaxConns = 10
	config.MinConns = 2
	config.MaxConnLifetime = time.Hour
	config.MaxConnIdleTime = 30 * time.Minute
	config.HealthCheckPeriod = 1 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create platform DB pool: %w", err)
	}

	// Test connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping platform DB: %w", err)
	}

	return &PlatformDB{pool: pool}, nil
}

// GetFamilyBySlug retrieves family information by slug
func (db *PlatformDB) GetFamilyBySlug(ctx context.Context, slug string) (*models.Family, error) {
	query := `
		SELECT id, slug, name, db_host, db_port, db_name, db_user, db_password_encrypted,
		       plan, status, created_at, updated_at
		FROM families
		WHERE slug = $1 AND deleted_at IS NULL AND status = 'active'
	`

	var family models.Family
	var dbUser, dbPassword *string

	err := db.pool.QueryRow(ctx, query, slug).Scan(
		&family.ID,
		&family.Slug,
		&family.Name,
		&family.DBHost,
		&family.DBPort,
		&family.DBName,
		&dbUser,
		&dbPassword,
		&family.Plan,
		&family.Status,
		&family.CreatedAt,
		&family.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get family by slug %s: %w", slug, err)
	}

	// Handle nullable fields
	if dbUser != nil {
		family.DBUser = *dbUser
	}
	if dbPassword != nil {
		family.DBPasswordEncrypted = *dbPassword
	}

	return &family, nil
}

// UpdateFamilyLastActivity updates the last_activity_at timestamp
func (db *PlatformDB) UpdateFamilyLastActivity(ctx context.Context, familyID string) error {
	query := `UPDATE families SET last_activity_at = NOW() WHERE id = $1`
	_, err := db.pool.Exec(ctx, query, familyID)
	return err
}

// Close closes the platform database connection pool
func (db *PlatformDB) Close() {
	db.pool.Close()
}

// Health checks if the platform database is healthy
func (db *PlatformDB) Health(ctx context.Context) error {
	return db.pool.Ping(ctx)
}
