package db

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ProcessingTaskInput struct {
	ID        string
	MediaID   string
	Step      string
	Status    string
	InputKey  string
	OutputKey string
}

type ProcessingTaskRow struct {
	ID         string
	MediaID    string
	Step       string
	Status     string
	RetryCount int
	InputKey   string
	OutputKey  string
}

func InsertProcessingTask(ctx context.Context, pool *pgxpool.Pool, t ProcessingTaskInput) (bool, error) {
	cmd, err := pool.Exec(ctx,
		"INSERT INTO processing_task (id, media_id, step, status, input_key, output_key) VALUES ($1, $2, $3, $4, $5, $6) ON CONFLICT (media_id, step) DO NOTHING",
		t.ID, t.MediaID, t.Step, t.Status, t.InputKey, t.OutputKey,
	)
	if err != nil {
		return false, err
	}
	return cmd.RowsAffected() == 1, nil
}

func GetTask(ctx context.Context, pool *pgxpool.Pool, taskID string) (*ProcessingTaskRow, error) {
	row := pool.QueryRow(ctx,
		"SELECT id, media_id, step, status, retry_count, input_key, output_key FROM processing_task WHERE id = $1",
		taskID,
	)
	var t ProcessingTaskRow
	if err := row.Scan(&t.ID, &t.MediaID, &t.Step, &t.Status, &t.RetryCount, &t.InputKey, &t.OutputKey); err != nil {
		return nil, err
	}
	return &t, nil
}

func ClaimTask(ctx context.Context, pool *pgxpool.Pool, taskID string, workerID string, leaseSeconds int) (bool, error) {
	cmd, err := pool.Exec(ctx,
		"UPDATE processing_task SET status = 'RUNNING', lock_by = $2, lock_until = NOW() + ($3 * INTERVAL '1 second'), updated_at = NOW() WHERE id = $1 AND status IN ('PENDING','RETRY') AND (lock_until IS NULL OR lock_until < NOW())",
		taskID, workerID, leaseSeconds,
	)
	if err != nil {
		return false, err
	}
	return cmd.RowsAffected() == 1, nil
}

func MarkTaskSucceeded(ctx context.Context, pool *pgxpool.Pool, taskID string) error {
	_, err := pool.Exec(ctx,
		"UPDATE processing_task SET status = 'SUCCEEDED', lock_by = NULL, lock_until = NULL, updated_at = NOW() WHERE id = $1",
		taskID,
	)
	return err
}

func MarkTaskFailed(ctx context.Context, pool *pgxpool.Pool, taskID string, errMsg string) error {
	_, err := pool.Exec(ctx,
		"UPDATE processing_task SET status = 'FAILED', last_error = $2, lock_by = NULL, lock_until = NULL, updated_at = NOW() WHERE id = $1",
		taskID, errMsg,
	)
	return err
}

func MarkTaskRetry(ctx context.Context, pool *pgxpool.Pool, taskID string, errMsg string, backoff time.Duration) error {
	seconds := int(backoff.Seconds())
	if seconds <= 0 {
		seconds = 30
	}
	_, err := pool.Exec(ctx,
		"UPDATE processing_task SET status = 'RETRY', retry_count = retry_count + 1, last_error = $2, lock_by = NULL, lock_until = NOW() + ($3 * INTERVAL '1 second'), updated_at = NOW() WHERE id = $1",
		taskID, errMsg, seconds,
	)
	return err
}
