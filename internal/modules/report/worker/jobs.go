package worker

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"telegram_bot/internal/modules/report"
	"telegram_bot/pkg/metrics"
)

type StrictReportStream interface {
	WriteRow(username, role string, detail report.SubmissionDetail) error
	Finish(headerData *report.StrictReportData) ([]byte, error)
}

type ReportGenerator interface {
	GenerateStatsReport(data []report.UserStat) ([]byte, error)
	NewStrictSubmissionsStream() (StrictReportStream, error)
	GenerateWeeklyReportsExcel(data []report.WeeklyReportData) ([]byte, error)
	GenerateStudentSheetsExcel(data []report.StudentSheetRecord) ([]byte, error)
}

type DataProvider interface {
	GetStats(ctx context.Context, courseID string) ([]report.UserStat, error)
	StreamStrictSubmissionsReport(ctx context.Context, courseID string, curatorID int64, rowCallback func(username,
		role string, detail report.SubmissionDetail) error) (*report.StrictReportData, error)
	GetCuratorStatsReport(ctx context.Context, curatorID int64, courseID string) ([]report.UserStat, error)
	GetWeeklyReportsReport(ctx context.Context, courseID string, curatorID int64) ([]report.WeeklyReportData, error)
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
	start := time.Now()

	var fileBytes []byte
	var fileName string
	var err error

	switch task.ReportType {

	case "admin_detailed_excel":
		data, errGet := provider.GetStats(ctx, "")
		if errGet != nil {
			log.Error("failed to fetch stats", "error", errGet)
			metrics.ErrorsTotal.WithLabelValues("report_worker").Inc()
			_ = sender.SendFile(task.AdminChatID, nil, "", "❌ Ошибка сбора данных :(")
			return
		}
		fileBytes, err = gen.GenerateStatsReport(data)
		fileName = fmt.Sprintf("users_report_%s.xlsx", time.Now().Format("2006-01-02_15-04"))

	case "excel_submissions", "curator_excel_submissions":
		var curatorID int64 = 0
		if task.ReportType == "curator_excel_submissions" {
			curatorID = task.AdminChatID
		}

		stream, errGen := gen.NewStrictSubmissionsStream()
		if errGen != nil {
			log.Error("failed to init strict submissions stream", "error", errGen)
			metrics.ErrorsTotal.WithLabelValues("report_worker").Inc()
			_ = sender.SendFile(task.AdminChatID, nil, "", "❌ Ошибка инициализации генератора :(")
			return
		}

		// Передаем курс и куратора в провайдер данных
		headerData, errGet := provider.StreamStrictSubmissionsReport(ctx, task.CourseID, curatorID, stream.WriteRow)
		if errGet != nil {
			log.Error("failed to stream strict submissions", "error", errGet)
			metrics.ErrorsTotal.WithLabelValues("report_worker").Inc()
			_ = sender.SendFile(task.AdminChatID, nil, "", "❌ Ошибка сбора данных о сдачах :(")
			return
		}

		fileBytes, err = stream.Finish(headerData)
		fileName = fmt.Sprintf("submissions_%s_%s.xlsx", task.CourseID, time.Now().Format("2006-01-02_15-04"))

	case "admin_weekly", "curator_weekly":
		var curatorID int64 = 0
		if task.ReportType == "curator_weekly" {
			curatorID = task.AdminChatID
		}

		data, errGet := provider.GetWeeklyReportsReport(ctx, task.CourseID, curatorID)
		if errGet != nil {
			log.Error("failed to fetch weekly reports", "error", errGet)
			metrics.ErrorsTotal.WithLabelValues("report_worker").Inc()
			_ = sender.SendFile(task.AdminChatID, nil, "", "❌ Ошибка сбора еженедельных отчетов")
			return
		}

		if len(data) == 0 {
			msg := "📭 За эту неделю отчетов пока нет."
			if task.CourseID != "" {
				msg = fmt.Sprintf("📭 Нет отчетов на курсе «%s».", task.CourseID)
			}
			_ = sender.SendFile(task.AdminChatID, nil, "", msg)
			return
		}

		fileBytes, err = gen.GenerateWeeklyReportsExcel(data)

		prefix := "admin_weekly"
		if task.ReportType == "curator_weekly" {
			prefix = "curator_weekly"
		}
		fileName = fmt.Sprintf("%s_%s_%s.xlsx", prefix, task.CourseID, time.Now().Format("2006-01-02_15-04"))

	case "admin_student_sheets":
		data, errGet := provider.GetStudentSheetsReport(ctx)
		if errGet != nil {
			log.Error("failed to fetch student sheets", "error", errGet)
			metrics.ErrorsTotal.WithLabelValues("report_worker").Inc()
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
		data, errGet := provider.GetCuratorStatsReport(ctx, task.AdminChatID, task.CourseID)
		if errGet != nil {
			log.Error("failed to fetch stats", "error", errGet)
			metrics.ErrorsTotal.WithLabelValues("report_worker").Inc()
			_ = sender.SendFile(task.AdminChatID, nil, "", "❌ Ошибка сбора данных куратора :(")
			return
		}

		if len(data) == 0 {
			msg := "📭 У вас пока нет учеников для формирования отчета."
			if task.CourseID != "" {
				msg = fmt.Sprintf("📭 У вас нет учеников на курсе «%s».", task.CourseID)
			}
			_ = sender.SendFile(task.AdminChatID, nil, "", msg)
			return
		}

		fileBytes, err = gen.GenerateStatsReport(data)

		fileName = fmt.Sprintf("curator_%s_%s.xlsx", task.CourseID, time.Now().Format("2006-01-02_15-04"))
	}

	if err != nil {
		log.Error("failed to generate excel", "error", err)
		metrics.ErrorsTotal.WithLabelValues("report_worker").Inc()
		_ = sender.SendFile(task.AdminChatID, nil, "", "❌ Ошибка генерации отчета :(")
		return
	}

	err = sender.SendFile(task.AdminChatID, fileBytes, fileName, "Ваш отчет готов 📊")
	if err != nil {
		log.Error("failed to send file", "error", err)
		metrics.ErrorsTotal.WithLabelValues("report_worker_telegram").Inc() // <-- ДОБАВЛЕНО
	}

	metrics.ReportDuration.WithLabelValues(task.ReportType).Observe(time.Since(start).Seconds())
	metrics.ReportsGeneratedTotal.WithLabelValues(task.ReportType).Inc()
}
