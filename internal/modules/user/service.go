package user

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"telegram_bot/internal/modules/report"
	"time"
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

	for _, student := range students {
		if s.notifier != nil {
			if err := s.notifier.Notify(student.ID, text); err != nil {
				s.log.Warn("failed to notify student", "id", student.ID, "error", err)
			}
		}
	}
	return nil
}

func (s *Service) SendGlobalReminders(ctx context.Context) error {
	users, err := s.repo.GetAll(ctx)
	if err != nil {
		return err
	}

	for _, u := range users {
		if u.Role != RoleStudent {
			continue
		}
		stat, err := s.repo.GetDailyStat(ctx, u.ID, time.Now())
		if err == nil && stat.TotalFilesToday == 0 {
			if s.notifier != nil {
				_ = s.notifier.Notify(u.ID, "🔔 Не забудьте сдать работы сегодня!")
			}
		}
	}
	return nil
}

func (s *Service) GetCuratorsStats(ctx context.Context) (string, error) {
	curators, err := s.repo.GetByRole(ctx, RoleCurator)
	if err != nil {
		return "", err
	}

	result := "📊 Статистика кураторов:\n"
	for _, c := range curators {
		students, _ := s.repo.GetByCuratorID(ctx, c.ID)
		result += fmt.Sprintf("- %s %s: %d студентов\n", c.FirstName, c.LastName, len(students))
	}
	return result, nil
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
		}
	}
	return nil
}

func (s *Service) GetStats(ctx context.Context, courseID string) ([]report.UserStat, error) {
	users, err := s.repo.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	var stats []report.UserStat
	for _, u := range users {
		if u.Role != RoleStudent {
			continue
		}

		dailyStat, _ := s.repo.GetDailyStat(ctx, u.ID, time.Now())

		stat := report.UserStat{
			UserID:        u.ID,
			Name:          u.FirstName + " " + u.LastName,
			Role:          string(u.Role),
			RegisteredAt:  u.RegisteredAt,
			HomeworkCount: 0,
			FilesCount:    0,
		}
		if dailyStat != nil {
			stat.FilesCount = dailyStat.TotalFilesToday
		}
		stats = append(stats, stat)
	}
	return stats, nil
}
