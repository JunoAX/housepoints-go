package database

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/JunoAX/housepoints-go/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

// FamilyDBManager manages connections to family-specific databases
type FamilyDBManager struct {
	platformDB *PlatformDB
	pools      sync.Map // map[familyID]string -> *pgxpool.Pool
	mu         sync.RWMutex
}

// NewFamilyDBManager creates a new family database manager
func NewFamilyDBManager(platformDB *PlatformDB) *FamilyDBManager {
	return &FamilyDBManager{
		platformDB: platformDB,
	}
}

// GetFamilyDB retrieves or creates a connection pool for a family database
func (m *FamilyDBManager) GetFamilyDB(ctx context.Context, family *models.Family) (*pgxpool.Pool, error) {
	// Check if pool already exists
	if pool, ok := m.pools.Load(family.ID.String()); ok {
		return pool.(*pgxpool.Pool), nil
	}

	// Create new connection pool
	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check after acquiring lock
	if pool, ok := m.pools.Load(family.ID.String()); ok {
		return pool.(*pgxpool.Pool), nil
	}

	// Build connection string for family database
	// For now, using same user as platform DB (postgres)
	// In production, you'd decrypt db_password_encrypted
	connString := fmt.Sprintf(
		"postgres://postgres:%s@%s:%d/%s?sslmode=disable",
		"HP_Sec2025_O0mZVY90R1Yg8L", // TODO: Get from secure config
		family.DBHost,
		family.DBPort,
		family.DBName,
	)

	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse family DB config for %s: %w", family.Slug, err)
	}

	// Connection pool settings for family databases
	config.MaxConns = 25
	config.MinConns = 2
	config.MaxConnLifetime = time.Hour
	config.MaxConnIdleTime = 30 * time.Minute
	config.HealthCheckPeriod = 1 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create family DB pool for %s: %w", family.Slug, err)
	}

	// Test connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping family DB for %s: %w", family.Slug, err)
	}

	// Store in cache
	m.pools.Store(family.ID.String(), pool)

	return pool, nil
}

// GetFamilyDBBySlug is a convenience method that looks up family and gets DB
func (m *FamilyDBManager) GetFamilyDBBySlug(ctx context.Context, slug string) (*pgxpool.Pool, *models.Family, error) {
	// Look up family from platform database
	family, err := m.platformDB.GetFamilyBySlug(ctx, slug)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get family by slug: %w", err)
	}

	// Get or create connection pool
	pool, err := m.GetFamilyDB(ctx, family)
	if err != nil {
		return nil, nil, err
	}

	// Update last activity
	go func() {
		ctx := context.Background()
		_ = m.platformDB.UpdateFamilyLastActivity(ctx, family.ID.String())
	}()

	return pool, family, nil
}

// Close closes all family database connections
func (m *FamilyDBManager) Close() {
	m.pools.Range(func(key, value interface{}) bool {
		if pool, ok := value.(*pgxpool.Pool); ok {
			pool.Close()
		}
		m.pools.Delete(key)
		return true
	})
}

// PoolStats returns statistics about connection pools
func (m *FamilyDBManager) PoolStats() map[string]interface{} {
	stats := make(map[string]interface{})
	count := 0

	m.pools.Range(func(key, value interface{}) bool {
		count++
		if pool, ok := value.(*pgxpool.Pool); ok {
			poolStats := pool.Stat()
			stats[key.(string)] = map[string]interface{}{
				"acquired_conns":   poolStats.AcquiredConns(),
				"idle_conns":       poolStats.IdleConns(),
				"total_conns":      poolStats.TotalConns(),
				"max_conns":        poolStats.MaxConns(),
			}
		}
		return true
	})

	stats["total_pools"] = count
	return stats
}
