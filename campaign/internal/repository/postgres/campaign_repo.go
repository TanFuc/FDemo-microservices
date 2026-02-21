package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"microservices/campaign/internal/model"
)

// CampaignRepository implements the campaign repository interface using PostgreSQL.
type CampaignRepository struct {
	pool *pgxpool.Pool
}

// NewCampaignRepository creates a new campaign repository.
func NewCampaignRepository(pool *pgxpool.Pool) *CampaignRepository {
	return &CampaignRepository{pool: pool}
}

// Create creates a new campaign.
func (r *CampaignRepository) Create(ctx context.Context, campaign *model.Campaign) error {
	query := `
		INSERT INTO campaigns (id, name, start_time, end_time, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.pool.Exec(ctx, query,
		campaign.ID,
		campaign.Name,
		campaign.StartTime,
		campaign.EndTime,
		campaign.Status,
		campaign.CreatedAt,
		campaign.UpdatedAt,
	)
	return err
}

// GetByID retrieves a campaign by ID.
func (r *CampaignRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Campaign, error) {
	query := `
		SELECT id, name, start_time, end_time, status, created_at, updated_at
		FROM campaigns
		WHERE id = $1
	`
	row := r.pool.QueryRow(ctx, query, id)

	var campaign model.Campaign
	err := row.Scan(
		&campaign.ID,
		&campaign.Name,
		&campaign.StartTime,
		&campaign.EndTime,
		&campaign.Status,
		&campaign.CreatedAt,
		&campaign.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &campaign, nil
}

// Update updates a campaign.
func (r *CampaignRepository) Update(ctx context.Context, campaign *model.Campaign) error {
	query := `
		UPDATE campaigns
		SET name = $2, start_time = $3, end_time = $4, status = $5, updated_at = $6
		WHERE id = $1
	`
	_, err := r.pool.Exec(ctx, query,
		campaign.ID,
		campaign.Name,
		campaign.StartTime,
		campaign.EndTime,
		campaign.Status,
		time.Now(),
	)
	return err
}

// Delete deletes a campaign.
func (r *CampaignRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM campaigns WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	return err
}

// ListActive retrieves all active campaigns.
func (r *CampaignRepository) ListActive(ctx context.Context) ([]*model.Campaign, error) {
	query := `
		SELECT id, name, start_time, end_time, status, created_at, updated_at
		FROM campaigns
		WHERE status = 'ACTIVE' AND start_time <= NOW() AND end_time >= NOW()
		ORDER BY created_at DESC
	`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var campaigns []*model.Campaign
	for rows.Next() {
		var campaign model.Campaign
		if err := rows.Scan(
			&campaign.ID,
			&campaign.Name,
			&campaign.StartTime,
			&campaign.EndTime,
			&campaign.Status,
			&campaign.CreatedAt,
			&campaign.UpdatedAt,
		); err != nil {
			return nil, err
		}
		campaigns = append(campaigns, &campaign)
	}

	return campaigns, nil
}

// List retrieves campaigns with pagination.
func (r *CampaignRepository) List(ctx context.Context, offset, limit int) ([]*model.Campaign, int64, error) {
	// Get total count
	var total int64
	countQuery := `SELECT COUNT(*) FROM campaigns`
	if err := r.pool.QueryRow(ctx, countQuery).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Get campaigns
	query := `
		SELECT id, name, start_time, end_time, status, created_at, updated_at
		FROM campaigns
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`
	rows, err := r.pool.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var campaigns []*model.Campaign
	for rows.Next() {
		var campaign model.Campaign
		if err := rows.Scan(
			&campaign.ID,
			&campaign.Name,
			&campaign.StartTime,
			&campaign.EndTime,
			&campaign.Status,
			&campaign.CreatedAt,
			&campaign.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		campaigns = append(campaigns, &campaign)
	}

	return campaigns, total, nil
}
