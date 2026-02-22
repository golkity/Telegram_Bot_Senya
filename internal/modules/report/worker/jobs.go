package worker

import (
	"context"
	"fmt"
	"log/slog"
	"telegram_bot/internal/modules/report"
	"time"
)

type ReportGenerator interface {
	GenerateStatsReport(data []report.UserStat) ([]byte, error)
}

type DataProvider interface {
	GetStats(ctx context.Context, courseID string) ([]report.UserStat, error)
}

type FileSender interface {
	SendFile(chatID int64, fileData []byte, fileName string, caption string) error
}

func ProcessReportJob(
	ctx context.Context,
	task report.Task,
	provider DataProvider,
	gen ReportGenerator,
	sender FileSender,
	log *slog.Logger,
) {
	log.Info("worker started processing report", "admin_id", task.AdminChatID, "type", task.ReportType)

	data, err := provider.GetStats(ctx, task.CourseID)
	if err != nil {
		log.Error("failed to fetch stats", "error", err)
		_ = sender.SendFile(task.AdminChatID, nil, "", "Ошибка сбора данных :(")
		return
	}

	fileBytes, err := gen.GenerateStatsReport(data)
	if err != nil {
		log.Error("failed to generate excel", "error", err)
		_ = sender.SendFile(task.AdminChatID, nil, "", "Ошибка генерации отчета :(")
		return
	}

	fileName := fmt.Sprintf("report_%s.xlsx", time.Now().Format("2006-01-02_15-04"))
	err = sender.SendFile(task.AdminChatID, fileBytes, fileName, "Ваш отчет готов <3")
	if err != nil {
		log.Error("failed to send file", "error", err)
	}
}
