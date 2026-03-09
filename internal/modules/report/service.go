package report

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"telegram_bot/pkg/metrics"
)

type Service struct {
	log      *slog.Logger
	taskChan chan Task
}

func NewService(log *slog.Logger, taskChan chan Task) *Service {
	return &Service{
		log:      log,
		taskChan: taskChan,
	}
}

func (s *Service) IsReportDay(weekday time.Weekday) bool {
	return weekday == time.Sunday || weekday == time.Monday
}

func (s *Service) SaveWeeklyReport(ctx context.Context, userID int64, text string) error {
	s.log.Info("saving weekly report", "user_id", userID, "length", len(text))
	return nil
}

func (s *Service) RequestReport(ctx context.Context, userID int64, reportType string) error {
	select {
	case s.taskChan <- Task{AdminChatID: userID, ReportType: reportType}:
		s.log.Info("report task enqueued", "user_id", userID, "type", reportType)
	default:
		metrics.ErrorsTotal.WithLabelValues("report_queue_full").Inc()
		return fmt.Errorf("report queue is full")
	}
	return nil
}

func (s *Service) RequestCuratorExcelReport(ctx context.Context, curatorID int64, courseID string) error {
	select {
	case s.taskChan <- Task{AdminChatID: curatorID, CourseID: courseID, ReportType: "excel_summary"}:
		s.log.Info("curator excel report enqueued", "curator_id", curatorID)
	default:
		metrics.ErrorsTotal.WithLabelValues("report_queue_full").Inc()
		return fmt.Errorf("report queue is full")
	}
	return nil
}
