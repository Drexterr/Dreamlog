package repositories

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
)

type WaitlistRepository struct{ db *pgxpool.Pool }

func NewWaitlistRepository(db *pgxpool.Pool) *WaitlistRepository { return &WaitlistRepository{db: db} }

func (r *WaitlistRepository) Join(ctx context.Context, email string) error {
	_, err := r.db.Exec(ctx, `INSERT INTO launch_waitlist(email) VALUES ($1) ON CONFLICT (email) DO NOTHING`, email)
	return err
}
