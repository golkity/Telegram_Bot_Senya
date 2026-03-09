package submission

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
	Save(ctx context.Context, sub Submission) error
	GetAllByUserID(ctx context.Context, userID int64) ([]Submission, error)
	GetByTask(ctx context.Context, userID int64, subType Type, taskNum string) ([]Submission, error)
	GetByID(ctx context.Context, id int64) (*Submission, error)
	UpdatePathsAndCurator(ctx context.Context, subID int64, newCuratorID int64, newPaths []string) error
}

type repo struct {
	db *postgres.Client
}

func NewRepo(db *postgres.Client) Repository {
	return &repo{db: db}
}

func (r *repo) Save(ctx context.Context, s Submission) error {
	start := time.Now()

	q := `
       INSERT INTO submissions (
          user_id, 
          curator_id,
          submission_type, 
          task_number, 
          file_paths, 
          original_names, 
          comment, 
          submission_date
       )
       VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
    `

	_, err := r.db.Pool.Exec(ctx, q,
		s.UserID,
		s.CuratorID,
		s.Type,
		s.TaskNumber,
		s.FilePaths,
		s.OriginalNames,
		s.Comment,
	)

	metrics.DBQueryDuration.WithLabelValues("submission_save").Observe(time.Since(start).Seconds())

	if err != nil {
		metrics.DBQueriesTotal.WithLabelValues("submission_save", "error").Inc()
		return fmt.Errorf("failed to save submission to db: %w", err)
	}

	metrics.DBQueriesTotal.WithLabelValues("submission_save", "success").Inc()
	return nil
}

func (r *repo) GetAllByUserID(ctx context.Context, userID int64) ([]Submission, error) {
	start := time.Now()

	q := `
       SELECT id, user_id, curator_id, submission_type, task_number, comment, submission_date, file_paths, original_names
       FROM submissions 
       WHERE user_id = $1 
       ORDER BY submission_date DESC
    `
	rows, err := r.db.Pool.Query(ctx, q, userID)

	metrics.DBQueryDuration.WithLabelValues("submission_get_all").Observe(time.Since(start).Seconds())

	if err != nil {
		metrics.DBQueriesTotal.WithLabelValues("submission_get_all", "error").Inc()
		return nil, err
	}
	defer rows.Close()

	subs := make([]Submission, 0, 10)
	for rows.Next() {
		var s Submission
		var curID *int64

		if err := rows.Scan(
			&s.ID, &s.UserID, &curID, &s.Type, &s.TaskNumber, &s.Comment, &s.SubmittedAt, &s.FilePaths, &s.OriginalNames,
		); err != nil {
			metrics.DBQueriesTotal.WithLabelValues("submission_get_all", "error").Inc()
			return nil, err
		}
		if curID != nil {
			s.CuratorID = *curID
		}
		subs = append(subs, s)
	}
	if err := rows.Err(); err != nil {
		metrics.DBQueriesTotal.WithLabelValues("submission_get_all", "error").Inc()
		return nil, err
	}

	metrics.DBQueriesTotal.WithLabelValues("submission_get_all", "success").Inc()
	return subs, nil
}

func (r *repo) GetByTask(ctx context.Context, userID int64, subType Type, taskNum string) ([]Submission, error) {
	start := time.Now()

	q := `
       SELECT id, user_id, submission_type, task_number, submitted_at 
       FROM submissions 
       WHERE user_id = $1 AND submission_type = $2 AND task_number = $3
       ORDER BY submission_date DESC
    `
	rows, err := r.db.Pool.Query(ctx, q, userID, subType, taskNum)

	metrics.DBQueryDuration.WithLabelValues("submission_get_by_task").Observe(time.Since(start).Seconds())

	if err != nil {
		metrics.DBQueriesTotal.WithLabelValues("submission_get_by_task", "error").Inc()
		return nil, err
	}
	defer rows.Close()

	subs := make([]Submission, 0, 10)
	for rows.Next() {
		var s Submission

		if err := rows.Scan(&s.ID, &s.UserID, &s.Type, &s.TaskNumber, &s.SubmittedAt); err != nil {
			metrics.DBQueriesTotal.WithLabelValues("submission_get_by_task", "error").Inc()
			return nil, err
		}
		subs = append(subs, s)
	}
	if err := rows.Err(); err != nil {
		metrics.DBQueriesTotal.WithLabelValues("submission_get_by_task", "error").Inc()
		return nil, err
	}

	metrics.DBQueriesTotal.WithLabelValues("submission_get_by_task", "success").Inc()
	return subs, nil
}

func (r *repo) GetByID(ctx context.Context, id int64) (*Submission, error) {
	start := time.Now()

	q := `
       SELECT id, user_id, curator_id, submission_type, task_number, file_paths, original_names, comment, submission_date 
       FROM submissions 
       WHERE id = $1
    `
	var s Submission
	var curID *int64
	err := r.db.Pool.QueryRow(ctx, q, id).Scan(
		&s.ID, &s.UserID, &curID, &s.Type, &s.TaskNumber,
		&s.FilePaths, &s.OriginalNames,
		&s.Comment, &s.SubmittedAt,
	)

	metrics.DBQueryDuration.WithLabelValues("submission_get_by_id").Observe(time.Since(start).Seconds())

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			metrics.DBQueriesTotal.WithLabelValues("submission_get_by_id", "not_found").Inc()
			return nil, fmt.Errorf("submission not found")
		}
		metrics.DBQueriesTotal.WithLabelValues("submission_get_by_id", "error").Inc()
		return nil, err
	}
	if curID != nil {
		s.CuratorID = *curID
	}

	metrics.DBQueriesTotal.WithLabelValues("submission_get_by_id", "success").Inc()
	return &s, nil
}

func (r *repo) UpdatePathsAndCurator(ctx context.Context, subID int64, newCuratorID int64, newPaths []string) error {
	start := time.Now()

	q := `
       UPDATE submissions 
       SET curator_id = $1, file_paths = $2 
       WHERE id = $3
    `
	_, err := r.db.Pool.Exec(ctx, q, newCuratorID, newPaths, subID)

	metrics.DBQueryDuration.WithLabelValues("submission_update_paths").Observe(time.Since(start).Seconds())

	if err != nil {
		metrics.DBQueriesTotal.WithLabelValues("submission_update_paths", "error").Inc()
		return err
	}

	metrics.DBQueriesTotal.WithLabelValues("submission_update_paths", "success").Inc()
	return err
}
