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
	GenerateStrictSubmissionsReport(data *report.StrictReportData) ([]byte, error)
	GenerateWeeklyReportsExcel(data []report.WeeklyReportData) ([]byte, error)
	GenerateStudentSheetsExcel(data []report.StudentSheetRecord) ([]byte, error)
}

type DataProvider interface {
	GetStats(ctx context.Context, courseID string) ([]report.UserStat, error)
	GetStrictSubmissionsReport(ctx context.Context) (*report.StrictReportData, error)
	GetWeeklyReportsReport(ctx context.Context) ([]report.WeeklyReportData, error)
	GetStudentSheetsReport(ctx context.Context) ([]report.StudentSheetRecord, error)
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

	var fileBytes []byte
	var fileName string
	var err error

	switch task.ReportType {

	case "admin_detailed_excel":
		data, errGet := provider.GetStats(ctx, "")
		if errGet != nil {
			log.Error("failed to fetch stats", "error", errGet)
			_ = sender.SendFile(task.AdminChatID, nil, "", "❌ Ошибка сбора данных :(")
			return
		}
		fileBytes, err = gen.GenerateStatsReport(data)
		fileName = fmt.Sprintf("users_report_%s.xlsx", time.Now().Format("2006-01-02_15-04"))

	case "excel_submissions":
		data, errGet := provider.GetStrictSubmissionsReport(ctx)
		if errGet != nil {
			log.Error("failed to fetch strict submissions", "error", errGet)
			_ = sender.SendFile(task.AdminChatID, nil, "", "❌ Ошибка сбора данных о сдачах :(")
			return
		}
		fileBytes, err = gen.GenerateStrictSubmissionsReport(data)
		fileName = fmt.Sprintf("submissions_%s.xlsx", time.Now().Format("2006-01-02_15-04"))

	case "admin_weekly":
		data, errGet := provider.GetWeeklyReportsReport(ctx)
		if errGet != nil {
			log.Error("failed to fetch weekly reports", "error", errGet)
			_ = sender.SendFile(task.AdminChatID, nil, "", "❌ Ошибка сбора еженедельных отчетов")
			return
		}

		fileBytes, err = gen.GenerateWeeklyReportsExcel(data)
		fileName = fmt.Sprintf("weekly_reports_%s.xlsx", time.Now().Format("2006-01-02_15-04"))

	case "admin_student_sheets":
		data, errGet := provider.GetStudentSheetsReport(ctx)
		if errGet != nil {
			log.Error("failed to fetch student sheets", "error", errGet)
			_ = sender.SendFile(task.AdminChatID, nil, "", "❌ Ошибка сбора листов по ученикам")
			return
		}

		if len(data) == 0 {
			_ = sender.SendFile(task.AdminChatID, nil, "", "🤷‍♂️ Пока нет ни одного ученика для формирования листов.")
			return
		}

		fileBytes, err = gen.GenerateStudentSheetsExcel(data)
		fileName = fmt.Sprintf("student_sheets_%s.xlsx", time.Now().Format("2006-01-02_15-04"))

	default:
		data, errGet := provider.GetStats(ctx, task.CourseID)
		if errGet != nil {
			log.Error("failed to fetch stats", "error", errGet)
			_ = sender.SendFile(task.AdminChatID, nil, "", "❌ Ошибка сбора данных куратора :(")
			return
		}
		fileBytes, err = gen.GenerateStatsReport(data)
		fileName = fmt.Sprintf("curator_report_%s.xlsx", time.Now().Format("2006-01-02_15-04"))
	}

	if err != nil {
		log.Error("failed to generate excel", "error", err)
		_ = sender.SendFile(task.AdminChatID, nil, "", "❌ Ошибка генерации отчета :(")
		return
	}

	err = sender.SendFile(task.AdminChatID, fileBytes, fileName, "Ваш отчет готов 📊")
	if err != nil {
		log.Error("failed to send file", "error", err)
	}
}
