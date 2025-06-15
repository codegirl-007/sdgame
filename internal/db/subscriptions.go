package db

import (
	"context"
	"database/sql"
	"time"

	_ "github.com/tursodatabase/libsql-client-go/libsql"
)

type Subscription struct {
	ID        string
	UserID    string
	StripeID  string
	Status    string
	StartedAt time.Time
	EndsAt    sql.NullTime
}

type SubscriptionClient struct {
	db *sql.DB
}

func NewSubscriptionClient(db *sql.DB) *SubscriptionClient {
	return &SubscriptionClient{db: db}
}

// Insert or update subscription
func (c *SubscriptionClient) Upsert(ctx context.Context, s *Subscription) error {
	_, err := c.db.ExecContext(ctx, `
		INSERT INTO subscriptions (id, user_id, stripe_id, status, started_at, ends_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			status = excluded.status,
			started_at = excluded.started_at,
			ends_at = excluded.ends_at
	`, s.ID, s.UserID, s.StripeID, s.Status, s.StartedAt, s.EndsAt)
	return err
}

// Get by user ID
func (c *SubscriptionClient) GetByUserID(ctx context.Context, userID string) (*Subscription, error) {
	row := c.db.QueryRowContext(ctx, `
		SELECT id, user_id, stripe_id, status, started_at, ends_at
		FROM subscriptions
		WHERE user_id = ?
	`, userID)

	var s Subscription
	err := row.Scan(&s.ID, &s.UserID, &s.StripeID, &s.Status, &s.StartedAt, &s.EndsAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// Delete by user ID
func (c *SubscriptionClient) DeleteByUserID(ctx context.Context, userID string) error {
	_, err := c.db.ExecContext(ctx, `
		DELETE FROM subscriptions
		WHERE user_id = ?
	`, userID)
	return err
}
