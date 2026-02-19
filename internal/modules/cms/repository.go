package cms

import (
	"context"
	"errors"
	"fmt"
	"telegram_bot/pkg/postgres"

	"github.com/jackc/pgx/v5"
)

type Repository interface {
	GetByKey(ctx context.Context, key string) (string, error)
	Update(ctx context.Context, key string, value string) error
	GetAll(ctx context.Context) ([]Content, error)
}

type repo struct {
	db *postgres.Client
}

func NewRepo(db *postgres.Client) Repository {
	return &repo{db: db}
}

func (r *repo) GetByKey(ctx context.Context, key string) (string, error) {
	q := `SELECT content FROM cms_contents WHERE key = $1`
	var content string
	err := r.db.Pool.QueryRow(ctx, q, key).Scan(&content)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", fmt.Errorf("content not found")
		}
		return "", err
	}
	return content, nil
}

func (r *repo) Update(ctx context.Context, key string, value string) error {
	q := `
		INSERT INTO cms_contents (key, content, description) 
		VALUES ($1, $2, 'Auto-created')
		ON CONFLICT (key) DO UPDATE SET content = $2
	`
	_, err := r.db.Pool.Exec(ctx, q, key, value)
	return err
}

func (r *repo) GetAll(ctx context.Context) ([]Content, error) {
	q := `SELECT key, content, description FROM cms_contents ORDER BY key`
	rows, err := r.db.Pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var contents []Content
	for rows.Next() {
		var c Content
		if err := rows.Scan(&c.Key, &c.Value, &c.Description); err != nil {
			return nil, err
		}
		contents = append(contents, c)
	}
	return contents, nil
}
