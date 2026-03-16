package user

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"telegram_bot/internal/modules/report"
	"telegram_bot/pkg/metrics"
)

type Notifier interface {
	Notify(userID int64, text string) error
}

type Service struct {
	repo     Repository
	notifier Notifier
	log      *slog.Logger
}

func NewService(repo Repository, notifier Notifier, log *slog.Logger) *Service {
	return &Service{
		repo:     repo,
		notifier: notifier,
		log:      log,
	}
}

func (s *Service) RegisterOrUpdate(ctx context.Context, telegramUser User) error {
	if telegramUser.Role == "" {
		telegramUser.Role = RoleStudent
	}

	existing, err := s.repo.GetByID(ctx, telegramUser.ID)
	if err != nil && !errors.Is(err, ErrUserNotFound) {
		s.log.Error("failed to check user existence", "error", err)
		return err
	}

	if existing == nil {
		s.log.Info("registering new user", "id", telegramUser.ID, "username", telegramUser.Username)
		return s.repo.Create(ctx, telegramUser)
	}

	return nil
}

func (s *Service) SetUserCourse(ctx context.Context, userID int64, courseID string) error {
	return s.repo.UpdateUserCourse(ctx, userID, courseID)
}

func (s *Service) GetCourseByName(ctx context.Context, name string) (*Course, error) {
	return s.repo.GetCourseByName(ctx, name)
}

func (s *Service) GetUserInfo(ctx context.Context, id int64) (*User, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) GetAllUsers(ctx context.Context) ([]User, error) {
	return s.repo.GetAll(ctx)
}

func (s *Service) SetRole(ctx context.Context, id int64, role Role) error {
	return s.repo.UpdateRole(ctx, id, role)
}

func (s *Service) AssignCurator(ctx context.Context, studentID int64, curatorID int64, courseID string) error {
	curator, err := s.repo.GetByID(ctx, curatorID)
	if err != nil {
		return err
	}
	if curator.Role != RoleCurator && curator.Role != RoleAdmin && curator.Role != RoleDeveloper {
		return errors.New("target user is not a curator")
	}

	return s.repo.UpdateCurator(ctx, studentID, &curatorID, &courseID)
}

func (s *Service) GetStudentsByCurator(ctx context.Context, curatorID int64) ([]User, error) {
	return s.repo.GetByCuratorID(ctx, curatorID)
}

func (s *Service) GetAllCurators(ctx context.Context) ([]User, error) {
	return s.repo.GetByRole(ctx, RoleCurator)
}

func (s *Service) GetAllCourses(ctx context.Context) ([]string, error) {
	return s.repo.GetAllCourses(ctx)
}

func (s *Service) GetCuratorsByCourse(ctx context.Context, courseID string) ([]User, error) {
	return s.repo.GetCuratorsByCourse(ctx, courseID)
}

func (s *Service) GetDailyStats(ctx context.Context, userID int64) (*DailyStat, error) {
	return s.repo.GetDailyStat(ctx, userID, time.Now())
}

func (s *Service) ToggleAdminNotifications(ctx context.Context, userID int64) bool {
	u, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return false
	}
	newState := !u.AdminNotifications
	_ = s.repo.UpdateAdminNotifications(ctx, userID, newState)
	return newState
}

func (s *Service) BroadcastToStudents(ctx context.Context, curatorID int64, text string) error {
	students, err := s.repo.GetByCuratorID(ctx, curatorID)
	if err != nil {
		return err
	}

	if len(students) == 0 {
		return nil
	}

	go func(studentList []User) {
		s.log.Info("starting background broadcast", "curator_id", curatorID, "students_count", len(studentList))

		successCount := 0
		for _, student := range studentList {
			if s.notifier != nil {
				if err := s.notifier.Notify(student.ID, text); err != nil {
					s.log.Warn("failed to notify student", "id", student.ID, "error", err)
					metrics.ErrorsTotal.WithLabelValues("broadcast_student_fail").Inc()
				} else {
					successCount++
				}
			}
			time.Sleep(50 * time.Millisecond)
		}

		s.log.Info("background broadcast finished", "curator_id", curatorID, "successful_sends", successCount)
	}(students)

	return nil
}

func (s *Service) SendGlobalReminders(ctx context.Context) error {
	users, err := s.repo.GetAll(ctx)
	if err != nil {
		return err
	}

	go func(userList []User) {
		s.log.Info("starting global reminders broadcast")
		sent := 0

		for _, u := range userList {
			if u.Role != RoleStudent {
				continue
			}
			stat, err := s.repo.GetDailyStat(context.Background(), u.ID, time.Now())
			if err == nil && stat.TotalFilesToday == 0 {
				if s.notifier != nil {
					if err := s.notifier.Notify(u.ID, "🔔 Не забудьте сдать работы сегодня!"); err == nil {
						sent++
					} else {
						metrics.ErrorsTotal.WithLabelValues("global_reminder_fail").Inc()
					}
				}
				time.Sleep(50 * time.Millisecond)
			}
		}
		s.log.Info("global reminders finished", "sent_count", sent)
	}(users)

	return nil
}

func (s *Service) GetCuratorsStats(ctx context.Context) (string, error) {
	curators, err := s.repo.GetByRole(ctx, RoleCurator)
	if err != nil {
		return "", err
	}

	if len(curators) == 0 {
		return "📭 В системе пока нет кураторов.", nil
	}

	var sb strings.Builder
	sb.WriteString("👨‍🏫 <b>Общая статистика кураторов:</b>\n")
	sb.WriteString("━━━━━━━━━━━━━━━━━━\n\n")

	totalStudents := 0

	for _, c := range curators {
		students, _ := s.repo.GetByCuratorID(ctx, c.ID)
		count := len(students)
		totalStudents += count

		status := "🟢 В норме"
		if count >= 30 {
			status = "🔴 Перегружен"
		} else if count >= 15 {
			status = "🟡 Загружен"
		}

		sb.WriteString(fmt.Sprintf("🔹 <b>%s %s</b>\n", c.FirstName, c.LastName))
		sb.WriteString(fmt.Sprintf("  ├ 👥 Студентов: <b>%d</b>\n", count))
		sb.WriteString(fmt.Sprintf("  └ 🚦 Статус: %s\n\n", status))
	}

	sb.WriteString("━━━━━━━━━━━━━━━━━━\n")
	sb.WriteString(fmt.Sprintf("📊 <b>Итого студентов у кураторов:</b> %d", totalStudents))

	return sb.String(), nil
}

func (s *Service) DeleteUser(ctx context.Context, userID int64) error {
	return s.repo.Delete(ctx, userID)
}

func (s *Service) TransferStudents(ctx context.Context, sourceID, targetID int64) error {
	sourceUser, err := s.repo.GetByID(ctx, sourceID)
	if err != nil {
		return fmt.Errorf("source curator not found (id: %d): %w", sourceID, err)
	}

	targetUser, err := s.repo.GetByID(ctx, targetID)
	if err != nil {
		return fmt.Errorf("target curator not found (id: %d): %w", targetID, err)
	}

	if targetUser.Role != RoleCurator && targetUser.Role != RoleAdmin {
		return fmt.Errorf("target user (id: %d) is not a curator", targetID)
	}

	err = s.repo.TransferStudents(ctx, sourceID, targetID)
	if err != nil {
		s.log.Error("failed to transfer students", "source", sourceID, "target", targetID, "error", err)
		return err
	}

	if s.notifier != nil {
		msg := fmt.Sprintf("📢 <b>Системное уведомление</b>\n\nВам были переданы ученики от куратора: %s %s (@%s).",
			sourceUser.FirstName, sourceUser.LastName, sourceUser.Username)

		if err := s.notifier.Notify(targetID, msg); err != nil {
			s.log.Warn("failed to notify target curator about transfer", "target_id", targetID, "error", err)
			metrics.ErrorsTotal.WithLabelValues("transfer_notify_fail").Inc()
		}
	}
	return nil
}

func (s *Service) GetCourseStatisticsText(ctx context.Context) (string, error) {
	stats, err := s.repo.GetStudentsCountByCourse(ctx)
	if err != nil {
		return "", err
	}

	if len(stats) == 0 {
		return "🤷‍♂️ Пока нет ни одного ученика, прикрепленного к курсам.", nil
	}

	var sb strings.Builder
	sb.WriteString("📚 <b>Аналитика распределения по курсам</b>\n")
	sb.WriteString("━━━━━━━━━━━━━━━━━━\n\n")

	total := 0
	for course, count := range stats {
		sb.WriteString(fmt.Sprintf("📌 <b>Курс:</b> «%s»\n", course))
		sb.WriteString(fmt.Sprintf("  └ 👥 Учеников: <b>%d</b>\n\n", count))
		total += count
	}

	sb.WriteString("━━━━━━━━━━━━━━━━━━\n")
	sb.WriteString(fmt.Sprintf("📊 <b>Всего активных учеников:</b> %d", total))

	return sb.String(), nil
}

func (s *Service) GetCuratorStatisticsText(ctx context.Context) (string, error) {
	stats, err := s.repo.GetStudentsCountByCurator(ctx)
	if err != nil {
		return "", err
	}

	if len(stats) == 0 {
		return "🤷‍♂️ Пока нет ни одного ученика, прикрепленного к кураторам.", nil
	}

	var sb strings.Builder
	sb.WriteString("👨‍🏫 <b>Распределение учеников (Сводка)</b>\n")
	sb.WriteString("━━━━━━━━━━━━━━━━━━\n\n")

	total := 0
	for curatorName, count := range stats {
		sb.WriteString(fmt.Sprintf("👤 <b>Куратор:</b> %s\n", curatorName))
		sb.WriteString(fmt.Sprintf("  └ 🎓 Учеников: <b>%d</b>\n\n", count))
		total += count
	}

	sb.WriteString("━━━━━━━━━━━━━━━━━━\n")
	sb.WriteString(fmt.Sprintf("📊 <b>Всего распределено:</b> %d чел.", total))

	return sb.String(), nil
}

func (s *Service) GetStats(ctx context.Context, courseID string) ([]report.UserStat, error) {
	return s.repo.GetUsersStatsReport(ctx, courseID)
}

func (s *Service) GetSubmissionsReport(ctx context.Context) ([]report.SubmissionStat, error) {
	return s.repo.GetSubmissionsReport(ctx)
}

func (s *Service) StreamStrictSubmissionsReport(ctx context.Context, rowCallback func(username, role string, detail report.SubmissionDetail) error) (*report.StrictReportData, error) {
	return s.repo.StreamStrictSubmissionsReport(ctx, rowCallback)
}

func (s *Service) GetWeeklyReportsReport(ctx context.Context, courseID string, curatorID int64) ([]report.WeeklyReportData, error) {
	return s.repo.GetWeeklyReportsReport(ctx, courseID, curatorID)
}

func (s *Service) GetDailyAdminStatsText(ctx context.Context) (string, error) {
	return s.repo.GetDailyAdminStatsText(ctx)
}

func (s *Service) GetStudentSheetsReport(ctx context.Context) ([]report.StudentSheetRecord, error) {
	return s.repo.GetStudentSheetsReport(ctx)
}

func (s *Service) GetCuratorStatsReport(ctx context.Context, curatorID int64, courseID string) ([]report.UserStat, error) {
	return s.repo.GetCuratorStatsReport(ctx, curatorID, courseID)
}

func (s *Service) GetCourseIDByName(ctx context.Context, name string) (string, error) {
	return s.repo.GetCourseIDByName(ctx, name)
}
