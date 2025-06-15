package db

import (
	"context"
	"database/sql"
	"errors"
	"time"

	_ "github.com/tursodatabase/libsql-client-go/libsql"
)

type Progress struct {
	ID          string
	UserID      string
	LevelID     string
	Status      string
	CompletedAt sql.NullTime
	AttemptData sql.NullString
	Stars       sql.NullInt64
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type ProgressClient struct {
	db *sql.DB
}

func NewProgressClient(db *sql.DB) *ProgressClient {
	return &ProgressClient{db: db}
}

func (c *ProgressClient) UpsertProgress(ctx context.Context, p *Progress) error {
	_, err := c.db.ExecContext(ctx, `
		INSERT INTO progress (id, user_id, level_id, status, completed_at, attempt_data, stars)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id, level_id) DO UPDATE SET
		  status = excluded.status,
		  completed_at = excluded.completed_at,
		  attempt_data = excluded.attempt_data,
		  stars = excluded.stars,
		  updated_at = CURRENT_TIMESTAMP
	`, p.ID, p.UserID, p.LevelID, p.Status, p.CompletedAt, p.AttemptData, p.Stars)
	return err
}

func (c *ProgressClient) GetProgressByUser(ctx context.Context, userID string) ([]*Progress, error) {
	rows, err := c.db.QueryContext(ctx, `
		SELECT id, user_id, level_id, status, completed_at, attempt_data, stars, created_at, updated_at
		FROM progress
		WHERE user_id = ?
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var progressList []*Progress
	for rows.Next() {
		var p Progress
		err := rows.Scan(&p.ID, &p.UserID, &p.LevelID, &p.Status, &p.CompletedAt, &p.AttemptData, &p.Stars, &p.CreatedAt, &p.UpdatedAt)
		if err != nil {
			return nil, err
		}
		progressList = append(progressList, &p)
	}

	return progressList, nil
}

func (c *ProgressClient) GetProgress(ctx context.Context, userID, levelID string) (*Progress, error) {
	row := c.db.QueryRowContext(ctx, `
		SELECT id, user_id, level_id, status, completed_at, attempt_data, stars, created_at, updated_at
		FROM progress
		WHERE user_id = ? AND level_id = ?
	`, userID, levelID)

	var p Progress
	err := row.Scan(&p.ID, &p.UserID, &p.LevelID, &p.Status, &p.CompletedAt, &p.AttemptData, &p.Stars, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}

func (c *ProgressClient) DeleteProgress(ctx context.Context, userID, levelID string) error {
	_, err := c.db.ExecContext(ctx, `
		DELETE FROM progress
		WHERE user_id = ? AND level_id = ?
	`, userID, levelID)
	return err
}
