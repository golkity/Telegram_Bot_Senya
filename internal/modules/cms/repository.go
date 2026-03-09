package cms

import (
	"context"
	"errors"
	"fmt"
	"time"

	"telegram_bot/pkg/metrics"
	"telegram_bot/pkg/postgres"

	"github.com/jackc/pgx/v5"
)

type Repository interface {
	GetByKey(ctx context.Context, key string) (string, error)
	Update(ctx context.Context, key string, value string) error
	GetAll(ctx context.Context) ([]Content, error)
	UpdateSetting(ctx context.Context, key string, content string) error
}

type repo struct {
	db *postgres.Client
}

func NewRepo(db *postgres.Client) Repository {
	return &repo{db: db}
}

func (r *repo) GetByKey(ctx context.Context, key string) (string, error) {
	start := time.Now()

	q := `SELECT content FROM cms_contents WHERE key = $1`
	var content string
	err := r.db.Pool.QueryRow(ctx, q, key).Scan(&content)

	metrics.DBQueryDuration.WithLabelValues("cms_get_by_key").Observe(time.Since(start).Seconds())

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			metrics.DBQueriesTotal.WithLabelValues("cms_get_by_key", "not_found").Inc()
			return "", fmt.Errorf("content not found")
		}
		metrics.DBQueriesTotal.WithLabelValues("cms_get_by_key", "error").Inc()
		return "", err
	}

	metrics.DBQueriesTotal.WithLabelValues("cms_get_by_key", "success").Inc()
	return content, nil
}

func (r *repo) Update(ctx context.Context, key string, value string) error {
	start := time.Now()

	q := `
       INSERT INTO cms_contents (key, content, description) 
       VALUES ($1, $2, 'Auto-created')
       ON CONFLICT (key) DO UPDATE SET content = $2
    `
	_, err := r.db.Pool.Exec(ctx, q, key, value)

	metrics.DBQueryDuration.WithLabelValues("cms_update").Observe(time.Since(start).Seconds())

	if err != nil {
		metrics.DBQueriesTotal.WithLabelValues("cms_update", "error").Inc()
		return err
	}

	metrics.DBQueriesTotal.WithLabelValues("cms_update", "success").Inc()
	return nil
}

func (r *repo) GetAll(ctx context.Context) ([]Content, error) {
	start := time.Now()

	q := `SELECT key, content, description FROM cms_contents ORDER BY key`
	rows, err := r.db.Pool.Query(ctx, q)

	metrics.DBQueryDuration.WithLabelValues("cms_get_all").Observe(time.Since(start).Seconds())

	if err != nil {
		metrics.DBQueriesTotal.WithLabelValues("cms_get_all", "error").Inc()
		return nil, err
	}
	defer rows.Close()

	contents := make([]Content, 0, 50)
	for rows.Next() {
		var c Content
		if err := rows.Scan(&c.Key, &c.Value, &c.Description); err != nil {
			metrics.DBQueriesTotal.WithLabelValues("cms_get_all", "error").Inc()
			return nil, err
		}
		contents = append(contents, c)
	}
	if err := rows.Err(); err != nil {
		metrics.DBQueriesTotal.WithLabelValues("cms_get_all", "error").Inc()
		return nil, err
	}

	metrics.DBQueriesTotal.WithLabelValues("cms_get_all", "success").Inc()
	return contents, nil
}

func (r *repo) UpdateSetting(ctx context.Context, key string, content string) error {
	start := time.Now()

	q := `
       INSERT INTO cms_contents (key, content)
       VALUES ($1, $2)
       ON CONFLICT (key) DO UPDATE 
       SET content = EXCLUDED.content
    `
	_, err := r.db.Pool.Exec(ctx, q, key, content)

	metrics.DBQueryDuration.WithLabelValues("cms_update_setting").Observe(time.Since(start).Seconds())

	if err != nil {
		metrics.DBQueriesTotal.WithLabelValues("cms_update_setting", "error").Inc()
		return fmt.Errorf("failed to upsert cms setting %s: %w", key, err)
	}

	metrics.DBQueriesTotal.WithLabelValues("cms_update_setting", "success").Inc()
	return nil
}
