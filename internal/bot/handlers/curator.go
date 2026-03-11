package handlers

import (
	"context"
	"fmt"
	"strings"

	"telegram_bot/internal/bot/keyboards"
	"telegram_bot/internal/modules/user"
	"telegram_bot/pkg/metrics"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (h *Handler) HandleCuratorMenu(ctx context.Context, msg *tgbotapi.Message) {
	h.deleteUserMessage(msg)
	h.SendMessage(msg.Chat.ID, "👨‍🏫 Панель куратора. Выберите действие:", keyboards.CuratorMenu)
}

func (h *Handler) HandleCuratorMyStudents(ctx context.Context, msg *tgbotapi.Message) {
	h.deleteUserMessage(msg)

	students, err := h.userSvc.GetStudentsByCurator(ctx, msg.From.ID)
	if err != nil {
		h.log.Error("failed to get students", "curator_id", msg.From.ID, "error", err)
		metrics.ErrorsTotal.WithLabelValues("curator_handler").Inc()
		h.SendMessage(msg.Chat.ID, "❌ Ошибка получения списка студентов.", nil)
		return
	}

	if len(students) == 0 {
		h.SendMessage(msg.Chat.ID, "📭 У вас пока нет прикрепленных студентов.", nil)
		return
	}

	var sb strings.Builder
	sb.Grow(len(students) * 60)
	sb.WriteString(fmt.Sprintf("📋 Ваши студенты (%d):\n\n", len(students)))

	for i, s := range students {
		roleIcon := "🎓"
		roleText := ""
		if s.Role == user.RoleDeveloper {
			roleIcon = "👨‍💻"
			roleText = " [Dev]"
		}

		name := s.FirstName
		if s.LastName != "" {
			name += " " + s.LastName
		}

		username := ""
		if s.Username != "" {
			username = fmt.Sprintf("(@%s)", s.Username)
		}

		sb.WriteString(fmt.Sprintf("%d. %s %s %s%s [ID: %d]\n", i+1, roleIcon, name, username, roleText, s.ID))
	}

	h.SendMessage(msg.Chat.ID, sb.String(), nil)
}

func (h *Handler) HandleCuratorReminder(ctx context.Context, msg *tgbotapi.Message) {
	h.deleteUserMessage(msg)
	h.state.SetState(msg.From.ID, StateWaitingForReminderText)
	h.SendMessage(msg.Chat.ID, "📝 Введите текст напоминания для всех ваших студентов:", keyboards.CancelButton)
}

func (h *Handler) HandleViewStudentWorks(ctx context.Context, msg *tgbotapi.Message) {
	h.deleteUserMessage(msg)

	students, err := h.userSvc.GetStudentsByCurator(ctx, msg.From.ID)
	if err != nil {
		metrics.ErrorsTotal.WithLabelValues("curator_handler").Inc()
		h.SendMessage(msg.Chat.ID, "Ошибка получения списка.", nil)
		return
	}

	if len(students) == 0 {
		h.SendMessage(msg.Chat.ID, "У вас нет студентов.", nil)
		return
	}

	rows := make([][]tgbotapi.InlineKeyboardButton, 0, len(students))
	for _, s := range students {
		name := s.FirstName
		if s.LastName != "" {
			name += " " + s.LastName
		}
		btnText := name
		if s.Role == user.RoleDeveloper {
			btnText = "👨‍💻 " + btnText
		} else {
			btnText = "🎓 " + btnText
		}

		data := fmt.Sprintf("view_works:%d", s.ID)

		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(btnText, data),
		))
	}

	markup := tgbotapi.NewInlineKeyboardMarkup(rows...)
	h.SendMessage(msg.Chat.ID, "👤 Выберите ученика для просмотра работ:", markup)
}

func (h *Handler) sendCourseSelectionForReport(ctx context.Context, chatID int64, reportType string, text string) {
	courses, err := h.userSvc.GetAllCourses(ctx)
	if err != nil {
		h.SendMessage(chatID, "❌ Ошибка получения списка курсов.", nil)
		return
	}

	var rows [][]tgbotapi.InlineKeyboardButton
	for _, c := range courses {
		data := fmt.Sprintf("gen_rep:%s:%s", reportType, c)
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(c, data),
		))
	}

	h.SendMessage(chatID, text+"\n📂 Выберите курс:", tgbotapi.NewInlineKeyboardMarkup(rows...))
}

func (h *Handler) HandleCuratorDailyReport(ctx context.Context, msg *tgbotapi.Message) {
	h.deleteUserMessage(msg)
	h.sendCourseSelectionForReport(ctx, msg.Chat.ID, "curator_daily", "📊 Генерация ежедневного отчета.")
}

func (h *Handler) HandleCuratorWeeklyReport(ctx context.Context, msg *tgbotapi.Message) {
	h.deleteUserMessage(msg)
	h.sendCourseSelectionForReport(ctx, msg.Chat.ID, "curator_weekly", "📈 Генерация еженедельного отчета.")
}

func (h *Handler) HandleWordReport(ctx context.Context, msg *tgbotapi.Message) {
	h.deleteUserMessage(msg)
	h.sendCourseSelectionForReport(ctx, msg.Chat.ID, "word_weekly", "📝 Генерация Word-отчета.")
}

func (h *Handler) HandleSubmissionReportExcel(ctx context.Context, msg *tgbotapi.Message) {
	h.deleteUserMessage(msg)
	h.sendCourseSelectionForReport(ctx, msg.Chat.ID, "excel_submissions", "🗂 Генерация детального Excel по сдачам.")
}

func (h *Handler) HandleSummaryExcel(ctx context.Context, msg *tgbotapi.Message) {
	h.deleteUserMessage(msg)
	h.sendCourseSelectionForReport(ctx, msg.Chat.ID, "excel_summary", "📑 Генерация сводной таблицы Excel.")
}
