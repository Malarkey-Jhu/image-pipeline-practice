package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

func InsertMedia(ctx context.Context, pool *pgxpool.Pool, id string, status string, originalKey string) error {
	_, err := pool.Exec(ctx,
		"INSERT INTO media (id, status, original_key) VALUES ($1, $2, $3)",
		id, status, originalKey,
	)
	return err
}

func UpdateMediaStatus(ctx context.Context, pool *pgxpool.Pool, id string, status string) error {
	_, err := pool.Exec(ctx,
		"UPDATE media SET status = $2, updated_at = NOW() WHERE id = $1",
		id, status,
	)
	return err
}

type MediaRow struct {
	ID          string
	Status      string
	OriginalKey string
	FinalKey    *string
}

func GetMedia(ctx context.Context, pool *pgxpool.Pool, id string) (*MediaRow, error) {
	row := pool.QueryRow(ctx,
		"SELECT id, status, original_key, final_key FROM media WHERE id = $1",
		id,
	)
	m := &MediaRow{}
	if err := row.Scan(&m.ID, &m.Status, &m.OriginalKey, &m.FinalKey); err != nil {
		return nil, err
	}
	return m, nil
}
