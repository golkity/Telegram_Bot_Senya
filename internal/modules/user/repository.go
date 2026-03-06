package user

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"telegram_bot/internal/modules/report"
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
	GetStudentsCountByCourse(ctx context.Context) (map[string]int, error)
	GetStudentsCountByCurator(ctx context.Context) (map[string]int, error)
	GetUsersStatsReport(ctx context.Context, courseID string) ([]report.UserStat, error)

	GetSubmissionsReport(ctx context.Context) ([]report.SubmissionStat, error)
	GetStrictSubmissionsReport(ctx context.Context) (*report.StrictReportData, error)
	GetWeeklyReportsReport(ctx context.Context) ([]report.WeeklyReportData, error)
	GetDailyAdminStatsText(ctx context.Context) (string, error)

	GetStudentSheetsReport(ctx context.Context) ([]report.StudentSheetRecord, error)
	GetCuratorStatsReport(ctx context.Context, curatorID int64) ([]report.UserStat, error)
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

	_, err = tx.Exec(ctx, `UPDATE user_roles SET curator_id = NULL WHERE curator_id = $1`, userID)
	if err != nil {
		return fmt.Errorf("failed to unlink students: %w", err)
	}

	_, err = tx.Exec(ctx, `DELETE FROM user_roles WHERE user_id = $1`, userID)
	if err != nil {
		return fmt.Errorf("failed to delete user role: %w", err)
	}

	_, _ = tx.Exec(ctx, `UPDATE submissions SET curator_id = NULL WHERE curator_id = $1`, userID)

	_, err = tx.Exec(ctx, `DELETE FROM users WHERE user_id = $1`, userID)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	return tx.Commit(ctx)
}

func (r *repo) TransferStudents(ctx context.Context, sourceCuratorID, targetCuratorID int64) error {
	query := `UPDATE user_roles SET curator_id = $1 WHERE curator_id = $2`
	_, err := r.db.Pool.Exec(ctx, query, targetCuratorID, sourceCuratorID)
	return err
}

func (r *repo) GetStudentsByCurator(ctx context.Context, curatorID int64) ([]int64, error) {
	q := `SELECT user_id FROM user_roles WHERE curator_id = $1`
	rows, err := r.db.Pool.Query(ctx, q, curatorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err == nil {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

func (r *repo) GetStudentsCountByCourse(ctx context.Context) (map[string]int, error) {
	q := `
		SELECT COALESCE(c.course_name, ur.course_id, 'Неизвестный курс'), COUNT(ur.user_id)
		FROM user_roles ur
		LEFT JOIN courses c ON ur.course_id = c.course_id
		WHERE ur.role = 'student' AND ur.course_id IS NOT NULL
		GROUP BY c.course_name, ur.course_id
		ORDER BY COUNT(ur.user_id) DESC
	`

	rows, err := r.db.Pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("db get students count by course error: %w", err)
	}
	defer rows.Close()

	stats := make(map[string]int)
	for rows.Next() {
		var courseName string
		var count int
		if err := rows.Scan(&courseName, &count); err != nil {
			return nil, err
		}
		stats[courseName] = count
	}

	return stats, nil
}

func (r *repo) GetStudentsCountByCurator(ctx context.Context) (map[string]int, error) {
	q := `
		SELECT 
			COALESCE(curator.first_name || ' ' || curator.last_name, 'Без имени (ID: ' || ur.curator_id || ')'),
			COUNT(ur.user_id)
		FROM user_roles ur
		JOIN users curator ON ur.curator_id = curator.user_id
		WHERE ur.role = 'student' AND ur.curator_id IS NOT NULL
		GROUP BY curator.first_name, curator.last_name, ur.curator_id
		ORDER BY COUNT(ur.user_id) DESC
	`

	rows, err := r.db.Pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("db get students count by curator error: %w", err)
	}
	defer rows.Close()

	stats := make(map[string]int)
	for rows.Next() {
		var curatorName string
		var count int
		if err := rows.Scan(&curatorName, &count); err != nil {
			return nil, err
		}
		stats[strings.TrimSpace(curatorName)] = count
	}

	return stats, nil
}

func (r *repo) GetUsersStatsReport(ctx context.Context, courseID string) ([]report.UserStat, error) {
	q := `
		SELECT 
			u.user_id, 
			TRIM(COALESCE(u.first_name, '') || ' ' || COALESCE(u.last_name, '')) AS name,
			COALESCE(ur.role, 'student') AS role,
			COUNT(s.id) FILTER (WHERE s.submission_type = 'homework') AS hw_count,
			COUNT(s.id) AS total_files,
			u.registration_date
		FROM users u
		LEFT JOIN user_roles ur ON u.user_id = ur.user_id
		LEFT JOIN submissions s ON u.user_id = s.user_id
		WHERE ($1::text = '' OR ur.course_id = $1)
		GROUP BY u.user_id, u.first_name, u.last_name, ur.role, u.registration_date
		ORDER BY ur.role, u.registration_date DESC
	`

	rows, err := r.db.Pool.Query(ctx, q, courseID)
	if err != nil {
		return nil, fmt.Errorf("db get users stats error: %w", err)
	}
	defer rows.Close()

	var stats []report.UserStat
	for rows.Next() {
		var stat report.UserStat
		var name string

		if err := rows.Scan(
			&stat.UserID,
			&name,
			&stat.Role,
			&stat.HomeworkCount,
			&stat.FilesCount,
			&stat.RegisteredAt,
		); err != nil {
			continue
		}

		if name == "" {
			name = "Без имени"
		}
		stat.Name = name

		stats = append(stats, stat)
	}

	return stats, nil
}

func (r *repo) GetSubmissionsReport(ctx context.Context) ([]report.SubmissionStat, error) {
	q := `
		SELECT 
			TO_CHAR(s.submission_date, 'YYYY-MM-DD HH24:MI') as date,
			COALESCE(u.first_name, '') || ' ' || COALESCE(u.last_name, '') as student_name,
			COALESCE(c.first_name, '') || ' ' || COALESCE(c.last_name, 'Без куратора') as curator_name,
			s.submission_type,
			COALESCE(s.status, 'pending')
		FROM submissions s
		JOIN users u ON s.user_id = u.user_id
		LEFT JOIN user_roles ur ON u.user_id = ur.user_id
		LEFT JOIN users c ON ur.curator_id = c.user_id
		ORDER BY s.submission_date DESC
	`

	rows, err := r.db.Pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("db get submissions error: %w", err)
	}
	defer rows.Close()

	var stats []report.SubmissionStat
	for rows.Next() {
		var stat report.SubmissionStat
		if err := rows.Scan(
			&stat.Date, &stat.StudentName, &stat.CuratorName,
			&stat.Type, &stat.Status,
		); err == nil {
			if stat.Type == "homework" {
				stat.Type = "ДЗ"
			} else if stat.Type == "notes" {
				stat.Type = "Конспект"
			}
			stats = append(stats, stat)
		}
	}
	return stats, nil
}

func (r *repo) GetStrictSubmissionsReport(ctx context.Context) (*report.StrictReportData, error) {
	var totalUsers, courseStudents, courseDevs int
	_ = r.db.Pool.QueryRow(ctx, `
		SELECT 
			COUNT(*),
			COUNT(*) FILTER (WHERE role = 'student'),
			COUNT(*) FILTER (WHERE role = 'developer')
		FROM user_roles
	`).Scan(&totalUsers, &courseStudents, &courseDevs)

	q := `
		SELECT 
			COALESCE(u.username, u.first_name, 'Без имени') AS username,
			COALESCE(ur.role, 'student') AS role,
			s.id AS sub_id,
			TO_CHAR(s.submission_date, 'YYYY-MM-DD HH24:MI:SS') AS sub_date,
			s.submission_type,
			COALESCE(s.status, 'pending') AS status,
			COALESCE(sm.subtask_name, s.task_number, 'Не указано') AS task_name,
			COALESCE(cardinality(s.file_paths), 0) AS files_count,
			COALESCE(s.comment, '—') AS comment
		FROM submissions s
		JOIN users u ON s.user_id = u.user_id
		LEFT JOIN user_roles ur ON u.user_id = ur.user_id
		LEFT JOIN subtask_mappings sm ON s.task_number = sm.subtask_code
		ORDER BY username, s.submission_date DESC
	`

	rows, err := r.db.Pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("db get strict submissions error: %w", err)
	}
	defer rows.Close()

	data := &report.StrictReportData{
		ReportTitle:    "ИСТОРИЯ СДАЧ (ГЛОБАЛЬНЫЙ ОТЧЕТ)",
		GenerationDate: time.Now().Format("02.01.2006 15:04"),
		CourseName:     "Все курсы",
		CuratorName:    "Все кураторы",
		Period:         "За всё время",
		WeekType:       "—",
		CourseFilter:   "НЕТ",
		TotalUsers:     totalUsers,
		CourseStudents: courseStudents,
		CourseDevs:     courseDevs,
	}

	userMap := make(map[string]*report.UserSubmissionsData)
	var usernames []string

	activeStudentsMap := make(map[string]bool)
	activeDevsMap := make(map[string]bool)

	for rows.Next() {
		var username, role, subDate, subType, status, taskName, comment string
		var subID int64
		var filesCount int

		if err := rows.Scan(&username, &role, &subID, &subDate, &subType, &status, &taskName, &filesCount, &comment); err != nil {
			continue
		}

		data.TotalSubmissions++
		if subType == "homework" {
			data.HWSubmissions++
			subType = "📚 ДЗ"
		} else if subType == "notes" {
			data.NotesSubmissions++
			subType = "📝 Конспект"
		}

		if role == "student" {
			activeStudentsMap[username] = true
		} else if role == "developer" {
			activeDevsMap[username] = true
		}

		if status == "pending" {
			status = "⏳ На проверке"
		} else if status == "approved" {
			status = "✅ Проверено"
		}

		userData, exists := userMap[username]
		if !exists {
			userData = &report.UserSubmissionsData{Username: username}
			userMap[username] = userData
			usernames = append(usernames, username)
		}

		detail := report.SubmissionDetail{
			DateTime:     subDate,
			Type:         subType,
			TaskName:     taskName,
			FilesCount:   filesCount,
			Comment:      comment,
			Status:       status,
			SubmissionID: subID,
		}

		userData.Submissions = append(userData.Submissions, detail)
		userData.TotalCount++
	}

	data.ActiveStudents = len(activeStudentsMap)
	data.ActiveDevs = len(activeDevsMap)
	data.DaysInPeriod = "За всё время"
	data.ReportTypeStat = "Глобальная выгрузка"

	if data.TotalUsers > 0 {
		data.AvgSubmissions = float64(data.TotalSubmissions) / float64(data.TotalUsers)
	}

	totalHWNotes := float64(data.HWSubmissions + data.NotesSubmissions)
	if totalHWNotes > 0 {
		hwPct := (float64(data.HWSubmissions) / totalHWNotes) * 100
		notesPct := (float64(data.NotesSubmissions) / totalHWNotes) * 100
		data.HWNotesRatio = fmt.Sprintf("%.1f%% / %.1f%%", hwPct, notesPct)
	} else {
		data.HWNotesRatio = "0.0% / 0.0%"
	}

	for _, uname := range usernames {
		uData := userMap[uname]
		for i := range uData.Submissions {
			uData.Submissions[i].Number = i + 1
		}
		data.Users = append(data.Users, *uData)
	}

	return data, nil
}

func (r *repo) GetWeeklyReportsReport(ctx context.Context) ([]report.WeeklyReportData, error) {
	q := `
		SELECT 
			TO_CHAR(wr.created_at, 'YYYY-MM-DD HH24:MI') AS date,
			COALESCE(u.username, u.first_name, 'Без имени') AS student_name,
			COALESCE(c.first_name, 'Без куратора') AS curator_name,
			wr.report_text
		FROM weekly_reports wr
		JOIN users u ON wr.user_id = u.user_id
		LEFT JOIN user_roles ur ON u.user_id = ur.user_id
		LEFT JOIN users c ON ur.curator_id = c.user_id
		ORDER BY wr.created_at DESC
	`

	rows, err := r.db.Pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("db get weekly reports error: %w", err)
	}
	defer rows.Close()

	var reports []report.WeeklyReportData
	for rows.Next() {
		var rep report.WeeklyReportData
		if err := rows.Scan(&rep.Date, &rep.StudentName, &rep.CuratorName, &rep.Text); err == nil {
			reports = append(reports, rep)
		}
	}
	return reports, nil
}

func (r *repo) GetDailyAdminStatsText(ctx context.Context) (string, error) {
	var newUsers, totalHW, totalNotes int

	err := r.db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE registration_date >= NOW() - INTERVAL '24 hours'`).Scan(&newUsers)
	if err != nil {
		return "", err
	}

	err = r.db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM submissions WHERE submission_date >= CURRENT_DATE AND submission_type = 'homework'`).Scan(&totalHW)
	if err != nil {
		return "", err
	}

	err = r.db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM submissions WHERE submission_date >= CURRENT_DATE AND submission_type = 'notes'`).Scan(&totalNotes)
	if err != nil {
		return "", err
	}

	text := fmt.Sprintf(
		"📈 <b>Общий отчет за сегодня:</b>\n\n"+
			"👤 Новых регистраций (за 24ч): <b>%d</b>\n"+
			"📚 Сдано ДЗ (сегодня): <b>%d</b>\n"+
			"📝 Сдано конспектов (сегодня): <b>%d</b>",
		newUsers, totalHW, totalNotes,
	)
	return text, nil
}

func (r *repo) GetStudentSheetsReport(ctx context.Context) ([]report.StudentSheetRecord, error) {
	q := `
		SELECT 
			COALESCE(c.first_name, '') || ' ' || COALESCE(c.last_name, 'Без куратора') AS curator_name,
			COALESCE(u.first_name, '') || ' ' || COALESCE(u.last_name, 'Без имени') AS student_name,
			COALESCE(u.username, '—') AS username,
			COALESCE(cr.course_name, 'Без курса') AS course_name,
			COUNT(s.id) FILTER (WHERE s.submission_type = 'homework') AS hw_count,
			COUNT(s.id) FILTER (WHERE s.submission_type = 'notes') AS notes_count,
			TO_CHAR(u.registration_date, 'YYYY-MM-DD') AS reg_date
		FROM users u
		JOIN user_roles ur ON u.user_id = ur.user_id AND ur.role = 'student'
		LEFT JOIN users c ON ur.curator_id = c.user_id
		LEFT JOIN courses cr ON ur.course_id = cr.course_id
		LEFT JOIN submissions s ON u.user_id = s.user_id
		GROUP BY curator_name, student_name, u.username, cr.course_name, u.registration_date
		ORDER BY curator_name, student_name
	`

	rows, err := r.db.Pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("db get student sheets error: %w", err)
	}
	defer rows.Close()

	var records []report.StudentSheetRecord
	for rows.Next() {
		var rec report.StudentSheetRecord
		if err := rows.Scan(&rec.CuratorName, &rec.StudentName, &rec.Username, &rec.CourseName, &rec.HWCount, &rec.NotesCount, &rec.RegisteredAt); err == nil {
			records = append(records, rec)
		}
	}
	return records, nil
}

func (r *repo) GetCuratorStatsReport(ctx context.Context, curatorID int64) ([]report.UserStat, error) {
	q := `
		SELECT 
			u.user_id, 
			TRIM(COALESCE(u.first_name, '') || ' ' || COALESCE(u.last_name, '')) AS name,
			COALESCE(ur.role, 'student') AS role,
			COUNT(s.id) FILTER (WHERE s.submission_type = 'homework') AS hw_count,
			COUNT(s.id) AS total_files,
			u.registration_date
		FROM users u
		JOIN user_roles ur ON u.user_id = ur.user_id
		LEFT JOIN submissions s ON u.user_id = s.user_id
		WHERE ur.curator_id = $1
		GROUP BY u.user_id, u.first_name, u.last_name, ur.role, u.registration_date
		ORDER BY u.registration_date DESC
	`

	rows, err := r.db.Pool.Query(ctx, q, curatorID)
	if err != nil {
		return nil, fmt.Errorf("db get curator stats error: %w", err)
	}
	defer rows.Close()

	var stats []report.UserStat
	for rows.Next() {
		var stat report.UserStat
		var name string

		if err := rows.Scan(
			&stat.UserID,
			&name,
			&stat.Role,
			&stat.HomeworkCount,
			&stat.FilesCount,
			&stat.RegisteredAt,
		); err != nil {
			continue
		}

		if name == "" {
			name = "Без имени"
		}
		stat.Name = name

		stats = append(stats, stat)
	}

	return stats, nil
}
