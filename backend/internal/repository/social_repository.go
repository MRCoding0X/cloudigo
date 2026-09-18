package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"cloudigo/backend/internal/model"
)

type SocialRepository struct {
	pool *pgxpool.Pool
}

func NewSocialRepository(pool *pgxpool.Pool) *SocialRepository {
	return &SocialRepository{pool: pool}
}

func (r *SocialRepository) Get(ctx context.Context) (*model.SocialLinks, error) {
	row := r.pool.QueryRow(ctx, `SELECT facebook, twitter, instagram, github FROM social_links WHERE id = 1`)
	var s model.SocialLinks
	if err := row.Scan(&s.Facebook, &s.Twitter, &s.Instagram, &s.GitHub); err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *SocialRepository) Update(ctx context.Context, s *model.SocialLinks) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE social_links SET facebook = $1, twitter = $2, instagram = $3, github = $4, updated_at = now()
		WHERE id = 1`,
		s.Facebook, s.Twitter, s.Instagram, s.GitHub,
	)
	return err
}
