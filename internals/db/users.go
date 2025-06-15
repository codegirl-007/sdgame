package db

import (
	"context"
	"database/sql"
	"time"

	_ "github.com/tursodatabase/libsql-client-go/libsql"
)

type User struct {
	ID          string
	GitHubID    int64
	GitHubLogin string
	AvatarURL   string
	Email       string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type UserClient struct {
	db *sql.DB
}

func NewUserClient(db *sql.DB) *UserClient {
	return &UserClient{db: db}
}

func (uc *UserClient) UpsertUser(ctx context.Context, user User) error {
	_, err := uc.db.ExecContext(ctx, `
		INSERT INTO users (id, github_id, github_login, avatar_url, email)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(github_id) DO UPDATE SET
			github_login = excluded.github_login,
			avatar_url = excluded.avatar_url,
			email = excluded.email,
			updated_at = CURRENT_TIMESTAMP
	`, user.ID, user.GitHubID, user.GitHubLogin, user.AvatarURL, user.Email)
	return err
}

func (uc *UserClient) GetUserByGitHubID(ctx context.Context, githubID int64) (*User, error) {
	row := uc.db.QueryRowContext(ctx, `
		SELECT id, github_id, github_login, avatar_url, email, created_at, updated_at
		FROM users WHERE github_id = ?
	`, githubID)

	var user User
	err := row.Scan(
		&user.ID, &user.GitHubID, &user.GitHubLogin,
		&user.AvatarURL, &user.Email, &user.CreatedAt, &user.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &user, err
}
