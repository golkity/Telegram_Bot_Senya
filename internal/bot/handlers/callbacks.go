package handlers

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"telegram_bot/internal/modules/submission"
	"telegram_bot/internal/modules/user"
	"telegram_bot/pkg/metrics"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var adminCommands = map[string]bool{
	"manage_user":         true,
	"delete_user":         true,
	"confirm_delete":      true,
	"user_stats":          true,
	"set_role":            true,
	"assign_course_menu":  true,
	"assign_curator_menu": true,
	"gen_excel":           true,
}

func (h *Handler) HandleCallback(ctx context.Context, callback *tgbotapi.CallbackQuery) {
	defer h.AnswerCallback(callback.ID, "")

	data := callback.Data
	parts := strings.Split(data, ":")
	if len(parts) == 0 {
		return
	}
	cmd := parts[0]

	metrics.UpdatesTotal.WithLabelValues("callback_" + cmd).Inc()

	initiator, err := h.userSvc.GetUserInfo(ctx, callback.From.ID)
	if err != nil {
		metrics.ErrorsTotal.WithLabelValues("bot_callback").Inc()
		return
	}

	if adminCommands[cmd] && initiator.Role != user.RoleAdmin {
		h.AnswerCallback(callback.ID, "⛔ Нет прав администратора")
		return
	}

	switch cmd {
	case "manage_user":
		if len(parts) < 2 {
			return
		}
		userID, _ := strconv.ParseInt(parts[1], 10, 64)
		h.showUserActions(ctx, callback.Message.Chat.ID, userID)

	case "delete_user":
		if len(parts) < 2 {
			return
		}
		userID, _ := strconv.ParseInt(parts[1], 10, 64)
		h.confirmDeleteUser(ctx, callback.Message.Chat.ID, userID)

	case "confirm_delete":
		if len(parts) < 2 {
			return
		}
		userID, _ := strconv.ParseInt(parts[1], 10, 64)
		h.executeDeleteUser(ctx, callback.Message.Chat.ID, userID)

	case "user_stats":
		if len(parts) < 2 {
			return
		}
		userID, _ := strconv.ParseInt(parts[1], 10, 64)
		h.showUserStats(ctx, callback.Message.Chat.ID, userID)

	case "set_role":
		if len(parts) < 3 {
			return
		}
		targetID, _ := strconv.ParseInt(parts[1], 10, 64)
		newRole := parts[2]
		h.executeSetRole(ctx, callback.Message.Chat.ID, targetID, newRole)

	case "assign_course_menu":
		if len(parts) < 2 {
			return
		}
		targetID, _ := strconv.ParseInt(parts[1], 10, 64)
		h.showCourseAssignment(ctx, callback.Message.Chat.ID, targetID)

	case "gen_rep":
		if len(parts) < 3 {
			return
		}
		reportType := parts[1]
		courseID := parts[2]
		h.executeGenerateCuratorReport(ctx, callback.Message.Chat.ID, callback.From.ID, reportType, courseID)

	case "assign_curator_menu":
		if len(parts) < 2 {
			return
		}
		targetID, _ := strconv.ParseInt(parts[1], 10, 64)
		h.showCuratorAssignment(ctx, callback.Message.Chat.ID, targetID)

	case "view_works":
		if len(parts) < 2 {
			return
		}
		studentID, _ := strconv.ParseInt(parts[1], 10, 64)
		h.showStudentTaskMenu(ctx, callback.Message.Chat.ID, studentID)

	case "view_task":
		if len(parts) < 4 {
			return
		}
		studentID, _ := strconv.ParseInt(parts[1], 10, 64)
		subType := parts[2]
		taskNum := parts[3]
		h.showStudentSubmissionsForTask(ctx, callback.Message.Chat.ID, studentID, subType, taskNum)

	case "sub_details":
		if len(parts) < 2 {
			return
		}
		subID, _ := strconv.ParseInt(parts[1], 10, 64)
		h.showSubmissionDetails(ctx, callback.Message.Chat.ID, subID)

	case "dl_file":
		if len(parts) < 3 {
			return
		}
		subID, _ := strconv.ParseInt(parts[1], 10, 64)
		fileIndex, _ := strconv.Atoi(parts[2])
		h.sendSubmissionFile(ctx, callback.Message.Chat.ID, subID, fileIndex)

	case "dev_attach":
		if len(parts) < 2 {
			return
		}
		curatorID, _ := strconv.ParseInt(parts[1], 10, 64)
		h.attachDeveloper(ctx, callback.From.ID, curatorID, callback.Message.Chat.ID)

	case "gen_excel":
		if len(parts) < 3 {
			return
		}
		curatorID, _ := strconv.ParseInt(parts[1], 10, 64)
		courseID := parts[2]
		h.generateCuratorExcel(ctx, callback.Message.Chat.ID, curatorID, courseID)
	}
}

func (h *Handler) showUserActions(ctx context.Context, chatID int64, targetID int64) {
	u, err := h.userSvc.GetUserInfo(ctx, targetID)
	if err != nil {
		metrics.ErrorsTotal.WithLabelValues("bot_callback").Inc()
		h.SendMessage(chatID, "Пользователь не найден", nil)
		return
	}

	text := fmt.Sprintf("👤 %s %s (@%s)\nРоль: %s\nID: %d", u.FirstName, u.LastName, u.Username, u.Role, u.ID)

	rows := [][]tgbotapi.InlineKeyboardButton{
		{tgbotapi.NewInlineKeyboardButtonData("✏️ Изменить роль", fmt.Sprintf("role_menu:%d", targetID))},
		{tgbotapi.NewInlineKeyboardButtonData("📊 Статистика", fmt.Sprintf("user_stats:%d", targetID))},
		{tgbotapi.NewInlineKeyboardButtonData("🗑️ Удалить", fmt.Sprintf("delete_user:%d", targetID))},
	}

	if u.Role == user.RoleStudent {
		rows = append(rows, []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData("👨‍🏫 Назначить куратора", fmt.Sprintf("assign_curator_menu:%d", targetID)),
		})
	}

	h.SendMessage(chatID, text, tgbotapi.NewInlineKeyboardMarkup(rows...))
}

func (h *Handler) confirmDeleteUser(_ context.Context, chatID int64, targetID int64) {
	text := fmt.Sprintf("⚠️ Вы уверены, что хотите удалить пользователя %d? Это действие необратимо.", targetID)
	rows := [][]tgbotapi.InlineKeyboardButton{
		{
			tgbotapi.NewInlineKeyboardButtonData("✅ Да, удалить", fmt.Sprintf("confirm_delete:%d", targetID)),
			tgbotapi.NewInlineKeyboardButtonData("❌ Отмена", fmt.Sprintf("manage_user:%d", targetID)),
		},
	}
	h.SendMessage(chatID, text, tgbotapi.NewInlineKeyboardMarkup(rows...))
}

func (h *Handler) executeDeleteUser(ctx context.Context, chatID int64, targetID int64) {
	err := h.userSvc.DeleteUser(ctx, targetID)
	if err != nil {
		metrics.ErrorsTotal.WithLabelValues("bot_callback").Inc()
		h.SendMessage(chatID, "❌ Ошибка при удалении пользователя", nil)
		return
	}
	h.SendMessage(chatID, "✅ Пользователь и все его данные удалены.", nil)
}

func (h *Handler) showUserStats(ctx context.Context, chatID int64, targetID int64) {
	stats, err := h.userSvc.GetDailyStats(ctx, targetID)
	if err != nil {
		metrics.ErrorsTotal.WithLabelValues("bot_callback").Inc()
		h.SendMessage(chatID, "Нет данных", nil)
		return
	}
	text := fmt.Sprintf("📊 Статистика пользователя %d:\nВсего файлов: %d\nДЗ сегодня: %s", targetID, stats.TotalFilesToday, stats.HomeworkStatus)
	h.SendMessage(chatID, text, nil)
}

func (h *Handler) executeSetRole(ctx context.Context, chatID int64, targetID int64, newRole string) {
	err := h.userSvc.SetRole(ctx, targetID, user.Role(newRole))
	if err != nil {
		metrics.ErrorsTotal.WithLabelValues("bot_callback").Inc()
		h.SendMessage(chatID, "Ошибка изменения роли", nil)
		return
	}
	h.SendMessage(chatID, fmt.Sprintf("✅ Роль пользователя %d изменена на %s", targetID, newRole), nil)
}

func (h *Handler) showCourseAssignment(_ context.Context, chatID int64, _ int64) {
	h.SendMessage(chatID, "Функция выбора курса в разработке (нужен CourseService)", nil)
}

func (h *Handler) showCuratorAssignment(ctx context.Context, chatID int64, targetID int64) {
	curators, err := h.userSvc.GetAllCurators(ctx)
	if err != nil {
		metrics.ErrorsTotal.WithLabelValues("bot_callback").Inc()
	}
	var rows [][]tgbotapi.InlineKeyboardButton
	for _, c := range curators {
		data := fmt.Sprintf("do_assign_cur:%d:%d", targetID, c.ID)
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(c.FirstName, data),
		))
	}
	h.SendMessage(chatID, "Выберите куратора:", tgbotapi.NewInlineKeyboardMarkup(rows...))
}

func (h *Handler) showStudentTaskMenu(ctx context.Context, chatID int64, studentID int64) {
	submissions, err := h.submissionSvc.GetAllSubmissions(ctx, studentID)
	if err != nil || len(submissions) == 0 {
		h.SendMessage(chatID, "📭 У студента нет загруженных работ.", nil)
		return
	}

	tasks := make(map[string]bool)
	for _, s := range submissions {
		key := fmt.Sprintf("%s:%s", s.Type, s.TaskNumber)
		tasks[key] = true
	}

	var rows [][]tgbotapi.InlineKeyboardButton
	for k := range tasks {
		parts := strings.Split(k, ":")
		if len(parts) < 2 {
			continue
		}
		subType, taskNum := parts[0], parts[1]

		btnText := fmt.Sprintf("%s %s", subType, taskNum)
		data := fmt.Sprintf("view_task:%d:%s:%s", studentID, subType, taskNum)

		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(btnText, data),
		))
	}

	h.SendMessage(chatID, "📂 Выберите задание для просмотра:", tgbotapi.NewInlineKeyboardMarkup(rows...))
}

func (h *Handler) showStudentSubmissionsForTask(ctx context.Context, chatID int64, studentID int64, subType, taskNum string) {
	submissions, _ := h.submissionSvc.GetSubmissionsByTask(ctx, studentID, submission.Type(subType), taskNum)

	if len(submissions) == 0 {
		h.SendMessage(chatID, "Нет файлов в этом задании", nil)
		return
	}

	var rows [][]tgbotapi.InlineKeyboardButton
	for _, s := range submissions {
		btnText := fmt.Sprintf("📅 %s", s.SubmittedAt.Format("02.01 15:04"))
		data := fmt.Sprintf("sub_details:%d", s.ID)
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(btnText, data),
		))
	}
	h.SendMessage(chatID, fmt.Sprintf("📋 Сдачи по заданию %s:", taskNum), tgbotapi.NewInlineKeyboardMarkup(rows...))
}

func (h *Handler) showSubmissionDetails(ctx context.Context, chatID int64, subID int64) {
	sub, err := h.submissionSvc.GetSubmissionByID(ctx, subID)
	if err != nil {
		metrics.ErrorsTotal.WithLabelValues("bot_callback").Inc()
		h.SendMessage(chatID, "Ошибка загрузки", nil)
		return
	}

	text := fmt.Sprintf("📄 Сдача #%d\nКоммент: %s\nФайлов: %d", sub.ID, sub.Comment, len(sub.FilePaths))

	var rows [][]tgbotapi.InlineKeyboardButton
	for i := range sub.FilePaths {
		name := "Файл " + strconv.Itoa(i+1)
		if i < len(sub.OriginalNames) {
			name = sub.OriginalNames[i]
		}

		data := fmt.Sprintf("dl_file:%d:%d", sub.ID, i)
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⬇️ "+name, data),
		))
	}
	h.SendMessage(chatID, text, tgbotapi.NewInlineKeyboardMarkup(rows...))
}

func (h *Handler) sendSubmissionFile(ctx context.Context, chatID int64, subID int64, fileIndex int) {
	sub, err := h.submissionSvc.GetSubmissionByID(ctx, subID)
	if err != nil || fileIndex < 0 || fileIndex >= len(sub.FilePaths) {
		metrics.ErrorsTotal.WithLabelValues("bot_callback").Inc()
		h.SendMessage(chatID, "❌ Файл не найден", nil)
		return
	}

	url, err := h.submissionSvc.GetFileLink(ctx, sub.FilePaths[fileIndex])
	if err != nil {
		metrics.ErrorsTotal.WithLabelValues("bot_callback").Inc()
		h.SendMessage(chatID, "❌ Файл недоступен", nil)
		return
	}

	msg := fmt.Sprintf("🔗 Ссылка на файл (действует 1 час):\n%s", url)
	h.SendMessage(chatID, msg, nil)
}

func (h *Handler) attachDeveloper(ctx context.Context, devID int64, curatorID int64, chatID int64) {
	err := h.userSvc.AssignCurator(ctx, devID, curatorID, "Годовой")
	if err != nil {
		metrics.ErrorsTotal.WithLabelValues("bot_callback").Inc()
		h.SendMessage(chatID, "Ошибка прикрепления (возможно, не указан курс)", nil)
		return
	}
	h.SendMessage(chatID, "✅ Вы успешно прикреплены к куратору.", nil)
}

func (h *Handler) generateCuratorExcel(ctx context.Context, chatID int64, curatorID int64, courseID string) {
	h.SendMessage(chatID, "⏳ Запуск генерации сводного Excel отчета...", nil)
	err := h.reportSvc.RequestCuratorExcelReport(ctx, curatorID, courseID)

	if err != nil {
		h.log.Error("failed to queue curator excel report",
			"curator_id", curatorID,
			"course_id", courseID,
			"error", err,
		)
		metrics.ErrorsTotal.WithLabelValues("bot_callback").Inc()
		h.SendMessage(chatID, "❌ Не удалось поставить задачу в очередь. Попробуйте позже.", nil)
		return
	}
}

func (h *Handler) executeGenerateCuratorReport(ctx context.Context, chatID int64, curatorID int64, reportType, courseID string) {
	h.SendMessage(chatID, fmt.Sprintf("⏳ Запуск генерации отчета по курсу «%s»...", courseID), nil)

	err := h.reportSvc.RequestReport(ctx, curatorID, courseID, reportType)
	if err != nil {
		h.log.Error("failed to queue report", "error", err)
		metrics.ErrorsTotal.WithLabelValues("curator_handler").Inc()
		h.SendMessage(chatID, "❌ Очередь отчетов занята, попробуйте позже.", nil)
	}
}
