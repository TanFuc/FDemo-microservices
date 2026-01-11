package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"microservices/campaign/internal/domain"
)

type CampaignRepository struct {
	pool *pgxpool.Pool
}

func NewCampaignRepository(pool *pgxpool.Pool) *CampaignRepository {
	return &CampaignRepository{pool: pool}
}

func (r *CampaignRepository) Create(ctx context.Context, campaign *domain.Campaign) error {
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
	if err != nil {
		return fmt.Errorf("failed to create campaign: %w", err)
	}
	return nil
}

func (r *CampaignRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Campaign, error) {
	query := `
		SELECT id, name, start_time, end_time, status, created_at, updated_at
		FROM campaigns
		WHERE id = $1
	`
	campaign := &domain.Campaign{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&campaign.ID,
		&campaign.Name,
		&campaign.StartTime,
		&campaign.EndTime,
		&campaign.Status,
		&campaign.CreatedAt,
		&campaign.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get campaign: %w", err)
	}
	return campaign, nil
}

func (r *CampaignRepository) Update(ctx context.Context, campaign *domain.Campaign) error {
	query := `
		UPDATE campaigns
		SET name = $2, start_time = $3, end_time = $4, status = $5
		WHERE id = $1
	`
	_, err := r.pool.Exec(ctx, query,
		campaign.ID,
		campaign.Name,
		campaign.StartTime,
		campaign.EndTime,
		campaign.Status,
	)
	if err != nil {
		return fmt.Errorf("failed to update campaign: %w", err)
	}
	return nil
}

func (r *CampaignRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM campaigns WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete campaign: %w", err)
	}
	return nil
}

func (r *CampaignRepository) ListActive(ctx context.Context) ([]*domain.Campaign, error) {
	query := `
		SELECT id, name, start_time, end_time, status, created_at, updated_at
		FROM campaigns
		WHERE status = 'ACTIVE' AND NOW() BETWEEN start_time AND end_time
		ORDER BY start_time DESC
	`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list active campaigns: %w", err)
	}
	defer rows.Close()

	var campaigns []*domain.Campaign
	for rows.Next() {
		campaign := &domain.Campaign{}
		if err := rows.Scan(
			&campaign.ID,
			&campaign.Name,
			&campaign.StartTime,
			&campaign.EndTime,
			&campaign.Status,
			&campaign.CreatedAt,
			&campaign.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan campaign: %w", err)
		}
		campaigns = append(campaigns, campaign)
	}
	return campaigns, nil
}

var _ domain.CampaignRepository = (*CampaignRepository)(nil)
