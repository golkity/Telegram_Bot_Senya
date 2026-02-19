package user

import (
	"context"
	"errors"
	"fmt"
	"telegram_bot/pkg/postgres"
	"time"

	"github.com/jackc/pgx/v5"
)

type Repository interface {
	Create(ctx context.Context, user User) error
	GetByID(ctx context.Context, id int64) (*User, error)
	GetAll(ctx context.Context) ([]User, error)

	UpdateRole(ctx context.Context, id int64, role Role) error
	UpdateCurator(ctx context.Context, studentID int64, curatorID *int64, courseID *string) error
	UpdateUserCourse(ctx context.Context, userID int64, courseID string) error
	UpdateAdminNotifications(ctx context.Context, id int64, enabled bool) error

	GetByRole(ctx context.Context, role Role) ([]User, error)

	GetCuratorsByCourse(ctx context.Context, courseID string) ([]User, error)
	GetAllCourses(ctx context.Context) ([]string, error)
	GetCourseIDByName(ctx context.Context, name string) (string, error)
	GetCourseByName(ctx context.Context, name string) (*Course, error)

	GetByCuratorID(ctx context.Context, curatorID int64) ([]User, error)
	GetDailyStat(ctx context.Context, userID int64, date time.Time) (*DailyStat, error)

	Delete(ctx context.Context, userID int64) error
	TransferStudents(ctx context.Context, sourceCuratorID, targetCuratorID int64) error
}

type repo struct {
	db *postgres.Client
}

func NewRepo(db *postgres.Client) Repository {
	return &repo{db: db}
}

func (r *repo) Create(ctx context.Context, u User) error {
	tx, err := r.db.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	qUser := `
       INSERT INTO users (user_id, username, first_name, last_name, registration_date)
       VALUES ($1, $2, $3, $4, NOW())
       ON CONFLICT (user_id) DO UPDATE 
       SET username = EXCLUDED.username, 
           first_name = EXCLUDED.first_name, 
           last_name = EXCLUDED.last_name
    `
	_, err = tx.Exec(ctx, qUser, u.ID, u.Username, u.FirstName, u.LastName)
	if err != nil {
		return fmt.Errorf("db create user error: %w", err)
	}

	qRoles := `
       INSERT INTO user_roles (user_id, role)
       VALUES ($1, $2)
       ON CONFLICT (user_id) DO NOTHING
    `
	_, err = tx.Exec(ctx, qRoles, u.ID, u.Role)
	if err != nil {
		return fmt.Errorf("db create user_roles error: %w", err)
	}

	return tx.Commit(ctx)
}

func (r *repo) GetByID(ctx context.Context, id int64) (*User, error) {
	q := `
       SELECT u.user_id, u.username, u.first_name, u.last_name, 
              ur.role, ur.course_id, ur.curator_id, u.registration_date, ur.admin_notifications 
       FROM users u
       LEFT JOIN user_roles ur ON u.user_id = ur.user_id
       WHERE u.user_id = $1
    `
	var u User
	err := r.db.Pool.QueryRow(ctx, q, id).Scan(
		&u.ID, &u.Username, &u.FirstName, &u.LastName,
		&u.Role, &u.CourseID, &u.CuratorID, &u.RegisteredAt, &u.AdminNotifications,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("db get user error: %w", err)
	}
	return &u, nil
}

func (r *repo) GetAll(ctx context.Context) ([]User, error) {
	q := `
       SELECT u.user_id, u.username, u.first_name, u.last_name, 
              ur.role, ur.course_id, ur.curator_id, u.registration_date, ur.admin_notifications 
       FROM users u
       LEFT JOIN user_roles ur ON u.user_id = ur.user_id
    `
	rows, err := r.db.Pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(
			&u.ID, &u.Username, &u.FirstName, &u.LastName,
			&u.Role, &u.CourseID, &u.CuratorID, &u.RegisteredAt, &u.AdminNotifications,
		); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

func (r *repo) UpdateRole(ctx context.Context, id int64, role Role) error {
	q := `INSERT INTO user_roles (user_id, role) VALUES ($1, $2) 
          ON CONFLICT (user_id) DO UPDATE SET role = EXCLUDED.role`
	_, err := r.db.Pool.Exec(ctx, q, id, role)
	return err
}

func (r *repo) UpdateCurator(ctx context.Context, studentID int64, curatorID *int64, courseID *string) error {
	q := `UPDATE user_roles SET curator_id = $1, course_id = $2 WHERE user_id = $3`
	_, err := r.db.Pool.Exec(ctx, q, curatorID, courseID, studentID)
	return err
}

func (r *repo) UpdateUserCourse(ctx context.Context, userID int64, courseID string) error {
	q := `UPDATE user_roles SET course_id = $1 WHERE user_id = $2`
	_, err := r.db.Pool.Exec(ctx, q, courseID, userID)
	return err
}

func (r *repo) UpdateAdminNotifications(ctx context.Context, id int64, enabled bool) error {
	q := `UPDATE user_roles SET admin_notifications = $1 WHERE user_id = $2`
	_, err := r.db.Pool.Exec(ctx, q, enabled, id)
	return err
}

func (r *repo) GetByRole(ctx context.Context, role Role) ([]User, error) {
	q := `
       SELECT u.user_id, u.username, u.first_name, u.last_name, ur.role, ur.course_id, ur.curator_id, u.registration_date, ur.admin_notifications
       FROM users u
       JOIN user_roles ur ON u.user_id = ur.user_id
       WHERE ur.role = $1
    `
	rows, err := r.db.Pool.Query(ctx, q, role)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(
			&u.ID, &u.Username, &u.FirstName, &u.LastName, &u.Role,
			&u.CourseID, &u.CuratorID, &u.RegisteredAt, &u.AdminNotifications,
		); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

func (r *repo) GetCuratorsByCourse(ctx context.Context, courseID string) ([]User, error) {
	q := `
       SELECT u.user_id, COALESCE(u.username, ''), u.first_name, COALESCE(u.last_name, ''), 
              ur.role, ur.course_id, ur.curator_id, u.registration_date, ur.admin_notifications
       FROM users u
       JOIN user_roles ur ON u.user_id = ur.user_id
       WHERE ur.role = 'curator' AND ur.course_id = $1
    `
	rows, err := r.db.Pool.Query(ctx, q, courseID)
	if err != nil {
		return nil, fmt.Errorf("db get curators by course error: %w", err)
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(
			&u.ID, &u.Username, &u.FirstName, &u.LastName, &u.Role,
			&u.CourseID, &u.CuratorID, &u.RegisteredAt, &u.AdminNotifications,
		); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

func (r *repo) GetAllCourses(ctx context.Context) ([]string, error) {
	q := `SELECT course_name FROM courses ORDER BY id ASC`
	rows, err := r.db.Pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var courses []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		courses = append(courses, name)
	}
	return courses, nil
}

func (r *repo) GetCourseIDByName(ctx context.Context, name string) (string, error) {
	q := `SELECT course_id FROM courses WHERE course_name = $1`
	var id string
	err := r.db.Pool.QueryRow(ctx, q, name).Scan(&id)
	if err != nil {
		return "", err
	}
	return id, nil
}

func (r *repo) GetCourseByName(ctx context.Context, name string) (*Course, error) {
	q := `SELECT course_id, course_name FROM courses WHERE course_name = $1`
	var c Course
	err := r.db.Pool.QueryRow(ctx, q, name).Scan(&c.ID, &c.Name)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCourseNotFound
		}
		return nil, err
	}
	return &c, nil
}

func (r *repo) GetByCuratorID(ctx context.Context, curatorID int64) ([]User, error) {
	q := `
       SELECT u.user_id, u.username, u.first_name, u.last_name, ur.role, ur.course_id, ur.curator_id, u.registration_date, ur.admin_notifications
       FROM users u
       JOIN user_roles ur ON u.user_id = ur.user_id
       WHERE ur.curator_id = $1
    `
	rows, err := r.db.Pool.Query(ctx, q, curatorID)
	if err != nil {
		return nil, fmt.Errorf("db get students by curator error: %w", err)
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(
			&u.ID, &u.Username, &u.FirstName, &u.LastName, &u.Role,
			&u.CourseID, &u.CuratorID, &u.RegisteredAt, &u.AdminNotifications,
		); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

func (r *repo) GetDailyStat(ctx context.Context, userID int64, date time.Time) (*DailyStat, error) {
	q := `
       SELECT 
          COUNT(*) FILTER (WHERE submission_type = 'homework') as hw_count,
          COUNT(*) FILTER (WHERE submission_type = 'notes') as notes_count,
          COUNT(*) as total_files
       FROM submissions 
       WHERE user_id = $1 AND DATE(submission_date) = DATE($2)
    `
	var hwCount, notesCount, totalFiles int
	err := r.db.Pool.QueryRow(ctx, q, userID, date).Scan(&hwCount, &notesCount, &totalFiles)
	if err != nil {
		return nil, err
	}

	hwStatus := "Не выполнено"
	if hwCount > 0 {
		hwStatus = "Выполнено"
	}

	notesStatus := "Не выполнено"
	if notesCount > 0 {
		notesStatus = "Выполнено"
	}

	return &DailyStat{
		HomeworkStatus:  hwStatus,
		NotesStatus:     notesStatus,
		TotalFilesToday: totalFiles,
	}, nil
}

func (r *repo) Delete(ctx context.Context, userID int64) error {
	tx, err := r.db.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `DELETE FROM user_roles WHERE user_id = $1`, userID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `DELETE FROM users WHERE user_id = $1`, userID)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *repo) TransferStudents(ctx context.Context, sourceCuratorID, targetCuratorID int64) error {
	query := `UPDATE user_roles SET curator_id = $1 WHERE curator_id = $2`
	_, err := r.db.Pool.Exec(ctx, query, targetCuratorID, sourceCuratorID)
	return err
}
