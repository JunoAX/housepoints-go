package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/JunoAX/housepoints-go/internal/middleware"
	"github.com/JunoAX/housepoints-go/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrFamilyNotFound = errors.New("family not found")
var ErrSlugTaken = errors.New("slug already taken")

type FamilyRepository struct {
	db *pgxpool.Pool
}

func NewFamilyRepository(db *pgxpool.Pool) *FamilyRepository {
	return &FamilyRepository{db: db}
}

// GetFamilyBySlug retrieves a family by its slug
func (r *FamilyRepository) GetFamilyBySlug(ctx context.Context, slug string) (*middleware.FamilyInfo, error) {
	query := `
		SELECT id, slug, name, active, plan
		FROM families
		WHERE slug = $1 AND deleted_at IS NULL
	`

	var family middleware.FamilyInfo
	err := r.db.QueryRow(ctx, query, slug).Scan(
		&family.ID,
		&family.Slug,
		&family.Name,
		&family.Active,
		&family.Plan,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrFamilyNotFound
		}
		return nil, err
	}

	return &family, nil
}

// GetFamilyByID retrieves full family details by ID
func (r *FamilyRepository) GetFamilyByID(ctx context.Context, id uuid.UUID) (*models.Family, error) {
	query := `
		SELECT id, slug, name, plan, active, created_at, updated_at, deleted_at
		FROM families
		WHERE id = $1
	`

	var family models.Family
	err := r.db.QueryRow(ctx, query, id).Scan(
		&family.ID,
		&family.Slug,
		&family.Name,
		&family.Plan,
		&family.Active,
		&family.CreatedAt,
		&family.UpdatedAt,
		&family.DeletedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrFamilyNotFound
		}
		return nil, err
	}

	return &family, nil
}

// CreateFamily creates a new family with the given slug
func (r *FamilyRepository) CreateFamily(ctx context.Context, family *models.Family) error {
	// Check if slug is already taken
	exists, err := r.SlugExists(ctx, family.Slug)
	if err != nil {
		return err
	}
	if exists {
		return ErrSlugTaken
	}

	query := `
		INSERT INTO families (id, slug, name, plan, active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		RETURNING id, created_at, updated_at
	`

	if family.ID == uuid.Nil {
		family.ID = uuid.New()
	}

	err = r.db.QueryRow(ctx, query,
		family.ID,
		family.Slug,
		family.Name,
		family.Plan,
		family.Active,
	).Scan(&family.ID, &family.CreatedAt, &family.UpdatedAt)

	return err
}

// SlugExists checks if a slug is already in use
func (r *FamilyRepository) SlugExists(ctx context.Context, slug string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM families WHERE slug = $1 AND deleted_at IS NULL)`
	var exists bool
	err := r.db.QueryRow(ctx, query, slug).Scan(&exists)
	return exists, err
}

// UpdateFamily updates family details
func (r *FamilyRepository) UpdateFamily(ctx context.Context, family *models.Family) error {
	query := `
		UPDATE families
		SET name = $1, plan = $2, active = $3, updated_at = NOW()
		WHERE id = $4 AND deleted_at IS NULL
	`

	result, err := r.db.Exec(ctx, query, family.Name, family.Plan, family.Active, family.ID)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrFamilyNotFound
	}

	return nil
}

// GetFamilySettings retrieves family settings
func (r *FamilyRepository) GetFamilySettings(ctx context.Context, familyID uuid.UUID) (*models.FamilySettings, error) {
	query := `
		SELECT id, family_id, timezone, currency, week_start_day, theme_color, custom_domain, created_at, updated_at
		FROM family_settings
		WHERE family_id = $1
	`

	var settings models.FamilySettings
	err := r.db.QueryRow(ctx, query, familyID).Scan(
		&settings.ID,
		&settings.FamilyID,
		&settings.Timezone,
		&settings.Currency,
		&settings.WeekStartDay,
		&settings.ThemeColor,
		&settings.CustomDomain,
		&settings.CreatedAt,
		&settings.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrFamilyNotFound
		}
		return nil, err
	}

	return &settings, nil
}
