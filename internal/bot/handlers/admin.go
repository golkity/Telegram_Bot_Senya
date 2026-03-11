package handlers

import (
	"context"
	"fmt"
	"strconv"

	"telegram_bot/internal/bot/keyboards"
	"telegram_bot/internal/modules/user"
	"telegram_bot/pkg/metrics"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (h *Handler) deleteUserMessage(msg *tgbotapi.Message) {
	if msg == nil {
		return
	}
	go func() {
		_, _ = h.bot.Request(tgbotapi.NewDeleteMessage(msg.Chat.ID, msg.MessageID))
	}()
}

func (h *Handler) checkAdminPermission(ctx context.Context, chatID int64, userID int64) bool {
	u, err := h.userSvc.GetUserInfo(ctx, userID)
	if err != nil || u.Role != user.RoleAdmin {
		h.SendMessage(chatID, "⛔ У вас нет прав администратора.", nil)
		return false
	}
	return true
}

func (h *Handler) HandleAdminPanel(ctx context.Context, msg *tgbotapi.Message) {
	h.deleteUserMessage(msg)
	if !h.checkAdminPermission(ctx, msg.Chat.ID, msg.From.ID) {
		return
	}
	h.SendMessage(msg.Chat.ID, "👨‍💼 Админ-панель", keyboards.AdminMenu)
}

func (h *Handler) HandleUserManagementMenu(ctx context.Context, msg *tgbotapi.Message) {
	h.deleteUserMessage(msg)
	if !h.checkAdminPermission(ctx, msg.Chat.ID, msg.From.ID) {
		return
	}
	h.SendMessage(msg.Chat.ID, "👤 Меню управления пользователями:", keyboards.UserManagementMenu)
}

func (h *Handler) HandleRoleManagement(ctx context.Context, msg *tgbotapi.Message) {
	h.deleteUserMessage(msg)
	if !h.checkAdminPermission(ctx, msg.Chat.ID, msg.From.ID) {
		return
	}
	h.SendMessage(msg.Chat.ID, "⚙️ Управление ролями:", keyboards.RoleManagementMenu)
}

func (h *Handler) HandleCustomization(ctx context.Context, msg *tgbotapi.Message) {
	h.deleteUserMessage(msg)
	if !h.checkAdminPermission(ctx, msg.Chat.ID, msg.From.ID) {
		return
	}
	h.SendMessage(msg.Chat.ID, "🎨 Кастомизация бота:", keyboards.CustomizationMenu)
}

func (h *Handler) HandleUserStatisticsRequest(ctx context.Context, msg *tgbotapi.Message) {
	h.deleteUserMessage(msg)
	if !h.checkAdminPermission(ctx, msg.Chat.ID, msg.From.ID) {
		return
	}
	h.state.SetState(msg.From.ID, StateWaitingForInputUserStats)
	h.SendMessage(msg.Chat.ID, "Введите ID пользователя или Username (без @) для просмотра статистики:", keyboards.BackButton)
}

func (h *Handler) HandleStartDeleteUser(ctx context.Context, msg *tgbotapi.Message) {
	h.deleteUserMessage(msg)
	if !h.checkAdminPermission(ctx, msg.Chat.ID, msg.From.ID) {
		return
	}
	h.state.SetState(msg.From.ID, StateWaitingForDeleteInput)
	h.SendMessage(msg.Chat.ID, "Введите ID пользователя, которого нужно удалить:", keyboards.BackButton)
}

func (h *Handler) HandleDeleteUserInput(ctx context.Context, msg *tgbotapi.Message) {
	h.deleteUserMessage(msg)
	if !h.checkAdminPermission(ctx, msg.Chat.ID, msg.From.ID) {
		return
	}

	input := msg.Text
	targetID, err := strconv.ParseInt(input, 10, 64)
	if err != nil {
		h.SendMessage(msg.Chat.ID, "❌ Некорректный ID. Введите число.", nil)
		return
	}

	u, err := h.userSvc.GetUserInfo(ctx, targetID)
	if err != nil {
		h.SendMessage(msg.Chat.ID, "❌ Пользователь не найден.", nil)
		return
	}

	h.state.SetData(msg.From.ID, "delete_target_id", targetID)
	h.state.SetState(msg.From.ID, StateWaitingForDeleteConfirm)

	text := fmt.Sprintf("⚠️ Вы собираетесь удалить пользователя:\n\n👤 %s %s (@%s)\nID: %d\n\nПодтвердите действие:",
		u.FirstName, u.LastName, u.Username, u.ID)

	h.SendMessage(msg.Chat.ID, text, keyboards.ConfirmDeleteMenu)
}

func (h *Handler) HandleDeleteConfirm(ctx context.Context, msg *tgbotapi.Message) {
	h.deleteUserMessage(msg)
	if !h.checkAdminPermission(ctx, msg.Chat.ID, msg.From.ID) {
		return
	}

	if msg.Text == "❌ Нет, отмена" {
		h.state.ClearState(msg.From.ID)
		h.state.ClearData(msg.From.ID)
		h.SendMessage(msg.Chat.ID, "Удаление отменено.", keyboards.UserManagementMenu)
		return
	}

	if msg.Text == "✅ Да, удалить" {
		rawID := h.state.GetData(msg.From.ID, "delete_target_id")
		var targetID int64
		switch v := rawID.(type) {
		case int64:
			targetID = v
		case float64:
			targetID = int64(v)
		default:
			h.log.Error("failed to parse delete_target_id from state", "rawID_type", fmt.Sprintf("%T", rawID))
			metrics.ErrorsTotal.WithLabelValues("admin_handler").Inc()
			h.SendMessage(msg.Chat.ID, "❌ Ошибка памяти. Попробуйте снова.", keyboards.UserManagementMenu)
			return
		}

		students, _ := h.userSvc.GetStudentsByCurator(ctx, targetID)

		err := h.userSvc.DeleteUser(ctx, targetID)
		if err != nil {
			h.log.Error("failed to delete user", "target_id", targetID, "error", err)
			metrics.ErrorsTotal.WithLabelValues("admin_handler").Inc()
			h.SendMessage(msg.Chat.ID, "❌ Ошибка при удалении. Проверьте логи сервера.", keyboards.UserManagementMenu)
		} else {
			h.SendMessage(msg.Chat.ID, "✅ Пользователь успешно удален.", keyboards.UserManagementMenu)

			for _, student := range students {
				h.SendMessage(student.ID, "⚠️ Ваш куратор был отстранен от курса.\nПожалуйста, отправьте команду /start, чтобы выбрать нового куратора!", nil)
			}
		}

		h.state.ClearState(msg.From.ID)
		h.state.ClearData(msg.From.ID)
	}
}

func (h *Handler) HandleBulkTransfer(ctx context.Context, msg *tgbotapi.Message) {
	h.deleteUserMessage(msg)
	if !h.checkAdminPermission(ctx, msg.Chat.ID, msg.From.ID) {
		return
	}
	h.SendMessage(msg.Chat.ID, "Выберите тип переноса:", keyboards.BulkTransferMenu)
}

func (h *Handler) HandleStartTransferByCurator(ctx context.Context, msg *tgbotapi.Message) {
	h.deleteUserMessage(msg)
	if !h.checkAdminPermission(ctx, msg.Chat.ID, msg.From.ID) {
		return
	}
	h.state.SetState(msg.From.ID, StateWaitingForTransferSource)
	h.SendMessage(msg.Chat.ID, "Введите ID куратора, ОТ КОТОРОГО нужно забрать учеников:", keyboards.BackButton)
}

func (h *Handler) HandleTransferSourceInput(ctx context.Context, msg *tgbotapi.Message) {
	h.deleteUserMessage(msg)
	if !h.checkAdminPermission(ctx, msg.Chat.ID, msg.From.ID) {
		return
	}
	sourceID, _ := strconv.ParseInt(msg.Text, 10, 64)
	h.state.SetData(msg.From.ID, "transfer_source", sourceID)

	h.state.SetState(msg.From.ID, StateWaitingForTransferTarget)
	h.SendMessage(msg.Chat.ID, "Теперь введите ID куратора, КОМУ передать учеников:", keyboards.BackButton)
}

func (h *Handler) HandleTransferTargetInput(ctx context.Context, msg *tgbotapi.Message) {
	h.deleteUserMessage(msg)
	if !h.checkAdminPermission(ctx, msg.Chat.ID, msg.From.ID) {
		return
	}
	targetID, _ := strconv.ParseInt(msg.Text, 10, 64)

	rawSource := h.state.GetData(msg.From.ID, "transfer_source")
	var sourceID int64
	switch v := rawSource.(type) {
	case int64:
		sourceID = v
	case float64:
		sourceID = int64(v)
	default:
		h.SendMessage(msg.Chat.ID, "❌ Сессия устарела. Начните перенос заново.", keyboards.AdminMenu)
		h.state.ClearState(msg.From.ID)
		return
	}

	h.state.SetData(msg.From.ID, "transfer_target", targetID)
	h.state.SetState(msg.From.ID, StateWaitingForTransferConfirm)

	text := fmt.Sprintf("⚠️ Подтвердите перенос:\n\nОт куратора ID: %d\nК куратору ID: %d", sourceID, targetID)

	h.SendMessage(msg.Chat.ID, text, keyboards.ConfirmTransferMenu)
}

func (h *Handler) HandleTransferConfirm(ctx context.Context, msg *tgbotapi.Message) {
	h.deleteUserMessage(msg)
	if !h.checkAdminPermission(ctx, msg.Chat.ID, msg.From.ID) {
		return
	}

	if msg.Text == "❌ Отмена" || msg.Text == "❌ Нет, отмена" {
		h.state.ClearState(msg.From.ID)
		h.SendMessage(msg.Chat.ID, "Перенос отменен.", keyboards.AdminMenu)
		return
	}

	if msg.Text == "✅ Подтвердить перенос" {
		var sourceID int64
		switch v := h.state.GetData(msg.From.ID, "transfer_source").(type) {
		case int64:
			sourceID = v
		case float64:
			sourceID = int64(v)
		}

		var targetID int64
		switch v := h.state.GetData(msg.From.ID, "transfer_target").(type) {
		case int64:
			targetID = v
		case float64:
			targetID = int64(v)
		}

		if sourceID == 0 || targetID == 0 {
			h.SendMessage(msg.Chat.ID, "❌ Ошибка чтения ID. Попробуйте снова.", keyboards.AdminMenu)
			return
		}

		err := h.userSvc.TransferStudents(ctx, sourceID, targetID)
		if err != nil {
			h.log.Error("failed to transfer students", "source", sourceID, "target", targetID, "error", err)
			metrics.ErrorsTotal.WithLabelValues("admin_handler").Inc()
			h.SendMessage(msg.Chat.ID, "❌ Ошибка при переносе. Проверьте логи.", keyboards.AdminMenu)
		} else {
			h.SendMessage(msg.Chat.ID, fmt.Sprintf("✅ Заявка на перенос от %d к %d выполнена.", sourceID, targetID), keyboards.AdminMenu)
		}

		h.state.ClearState(msg.From.ID)
		h.state.ClearData(msg.From.ID)
	}
}

func (h *Handler) HandleAdminDailyReport(ctx context.Context, msg *tgbotapi.Message) {
	h.deleteUserMessage(msg)
	text, err := h.userSvc.GetDailyAdminStatsText(ctx)
	if err != nil {
		h.log.Error("failed to get daily admin stats", "error", err)
		metrics.ErrorsTotal.WithLabelValues("admin_handler").Inc()
		h.SendMessage(msg.Chat.ID, "❌ Ошибка сбора ежедневной статистики", nil)
		return
	}

	h.SendMessage(msg.Chat.ID, text, nil)
}

func (h *Handler) HandleAdminWeeklyReport(ctx context.Context, msg *tgbotapi.Message) {
	h.deleteUserMessage(msg)
	if !h.checkAdminPermission(ctx, msg.Chat.ID, msg.From.ID) {
		return
	}
	err := h.reportSvc.RequestReport(ctx, msg.From.ID, "", "admin_weekly")
	if err != nil {
		return
	}
	h.SendMessage(msg.Chat.ID, "Запрос отправлен.", nil)
}

func (h *Handler) HandleAdminDetailedExcel(ctx context.Context, msg *tgbotapi.Message) {
	h.deleteUserMessage(msg)
	if !h.checkAdminPermission(ctx, msg.Chat.ID, msg.From.ID) {
		return
	}
	err := h.reportSvc.RequestReport(ctx, msg.From.ID, "", "admin_detailed_excel")
	if err != nil {
		return
	}
	h.SendMessage(msg.Chat.ID, "Запрос отправлен.", nil)
}

func (h *Handler) HandleAdminWordReport(ctx context.Context, msg *tgbotapi.Message) {
	h.deleteUserMessage(msg)
	if !h.checkAdminPermission(ctx, msg.Chat.ID, msg.From.ID) {
		return
	}
	err := h.reportSvc.RequestReport(ctx, msg.From.ID, "", "word_weekly")
	if err != nil {
		return
	}
	h.SendMessage(msg.Chat.ID, "Запрос отправлен.", nil)
}

func (h *Handler) HandleAdminSubmissionExcel(ctx context.Context, msg *tgbotapi.Message) {
	h.deleteUserMessage(msg)
	if !h.checkAdminPermission(ctx, msg.Chat.ID, msg.From.ID) {
		return
	}
	err := h.reportSvc.RequestReport(ctx, msg.From.ID, "", "excel_submissions")
	if err != nil {
		return
	}
	h.SendMessage(msg.Chat.ID, "Запрос отправлен.", nil)
}

func (h *Handler) HandleAdminStudentSheets(ctx context.Context, msg *tgbotapi.Message) {
	h.deleteUserMessage(msg)
	if !h.checkAdminPermission(ctx, msg.Chat.ID, msg.From.ID) {
		return
	}
	err := h.reportSvc.RequestReport(ctx, msg.From.ID, "", "admin_student_sheets")
	if err != nil {
		return
	}
	h.SendMessage(msg.Chat.ID, "Запрос отправлен.", nil)
}

func (h *Handler) HandleSendGlobalReminders(ctx context.Context, msg *tgbotapi.Message) {
	h.deleteUserMessage(msg)
	if !h.checkAdminPermission(ctx, msg.Chat.ID, msg.From.ID) {
		return
	}
	go func() {
		_ = h.userSvc.SendGlobalReminders(ctx)
	}()
	h.SendMessage(msg.Chat.ID, "🔔 Глобальная рассылка запущена.", nil)
}

func (h *Handler) HandleAdminToggleNotifications(ctx context.Context, msg *tgbotapi.Message) {
	h.deleteUserMessage(msg)
	if !h.checkAdminPermission(ctx, msg.Chat.ID, msg.From.ID) {
		return
	}
	newState := h.userSvc.ToggleAdminNotifications(ctx, msg.From.ID)
	status := "ВКЛЮЧЕНЫ"
	if !newState {
		status = "ВЫКЛЮЧЕНЫ"
	}
	h.SendMessage(msg.Chat.ID, "🔔 Уведомления: "+status, nil)
}

func (h *Handler) HandleAdminCuratorStats(ctx context.Context, msg *tgbotapi.Message) {
	h.deleteUserMessage(msg)
	if !h.checkAdminPermission(ctx, msg.Chat.ID, msg.From.ID) {
		return
	}
	stats, _ := h.userSvc.GetCuratorsStats(ctx)
	h.SendMessage(msg.Chat.ID, stats, nil)
}

func (h *Handler) HandleAdminStudentsByCourse(ctx context.Context, msg *tgbotapi.Message) {
	h.deleteUserMessage(msg)
	if !h.checkAdminPermission(ctx, msg.Chat.ID, msg.From.ID) {
		return
	}

	statsText, err := h.userSvc.GetCourseStatisticsText(ctx)
	if err != nil {
		h.log.Error("failed to get course statistics", "error", err)
		metrics.ErrorsTotal.WithLabelValues("admin_handler").Inc()
		h.SendMessage(msg.Chat.ID, "❌ Ошибка при получении данных базы.", nil)
		return
	}

	h.SendMessage(msg.Chat.ID, statsText, nil)
}

func (h *Handler) HandleAdminCuratorStudents(ctx context.Context, msg *tgbotapi.Message) {
	h.deleteUserMessage(msg)
	if !h.checkAdminPermission(ctx, msg.Chat.ID, msg.From.ID) {
		return
	}

	statsText, err := h.userSvc.GetCuratorStatisticsText(ctx)
	if err != nil {
		h.log.Error("failed to get curator statistics", "error", err)
		metrics.ErrorsTotal.WithLabelValues("admin_handler").Inc()
		h.SendMessage(msg.Chat.ID, "❌ Ошибка при получении данных.", nil)
		return
	}

	h.SendMessage(msg.Chat.ID, statsText, nil)
}

func (h *Handler) HandleCMSChangeBotName(ctx context.Context, msg *tgbotapi.Message) {
	h.deleteUserMessage(msg)
	if !h.checkAdminPermission(ctx, msg.Chat.ID, msg.From.ID) {
		return
	}
	h.state.SetState(msg.From.ID, StateCMSWaitingBotName)
	h.SendMessage(msg.Chat.ID, "Отправьте новое имя для бота:", keyboards.BackButton)
}

func (h *Handler) HandleCMSChangeGreeting(ctx context.Context, msg *tgbotapi.Message) {
	h.deleteUserMessage(msg)
	if !h.checkAdminPermission(ctx, msg.Chat.ID, msg.From.ID) {
		return
	}
	h.state.SetState(msg.From.ID, StateCMSWaitingGreeting)
	h.SendMessage(msg.Chat.ID, "Отправьте новый текст приветствия (/start):", keyboards.BackButton)
}

func (h *Handler) HandleCMSChangeButtons(ctx context.Context, msg *tgbotapi.Message) {
	h.deleteUserMessage(msg)
	if !h.checkAdminPermission(ctx, msg.Chat.ID, msg.From.ID) {
		return
	}
	h.state.SetState(msg.From.ID, StateCMSWaitingButtons)
	h.SendMessage(msg.Chat.ID, "Отправьте новые названия кнопок (в формате JSON или текст):", keyboards.BackButton)
}

func (h *Handler) HandleCMSChangeAllMessages(ctx context.Context, msg *tgbotapi.Message) {
	h.deleteUserMessage(msg)
	if !h.checkAdminPermission(ctx, msg.Chat.ID, msg.From.ID) {
		return
	}
	h.state.SetState(msg.From.ID, StateCMSWaitingMessages)
	h.SendMessage(msg.Chat.ID, "Отправьте новые системные сообщения:", keyboards.BackButton)
}

func (h *Handler) HandleCMSChangeReportTime(ctx context.Context, msg *tgbotapi.Message) {
	h.deleteUserMessage(msg)
	if !h.checkAdminPermission(ctx, msg.Chat.ID, msg.From.ID) {
		return
	}
	h.state.SetState(msg.From.ID, StateCMSWaitingReportTime)
	h.SendMessage(msg.Chat.ID, "Отправьте время для отчетов (МСК), например 20:00 :", keyboards.BackButton)
}

func (h *Handler) HandleCMSChangeReminderTime(ctx context.Context, msg *tgbotapi.Message) {
	h.deleteUserMessage(msg)
	if !h.checkAdminPermission(ctx, msg.Chat.ID, msg.From.ID) {
		return
	}
	h.state.SetState(msg.From.ID, StateCMSWaitingReminderTime)
	h.SendMessage(msg.Chat.ID, "Отправьте время для напоминаний (МСК), например 10:00 :", keyboards.BackButton)
}

func (h *Handler) HandleCMSChangeCheckTime(ctx context.Context, msg *tgbotapi.Message) {
	h.deleteUserMessage(msg)
	if !h.checkAdminPermission(ctx, msg.Chat.ID, msg.From.ID) {
		return
	}
	h.state.SetState(msg.From.ID, StateCMSWaitingCheckTime)
	h.SendMessage(msg.Chat.ID, "Отправьте время проверки работ (МСК):", keyboards.BackButton)
}

func (h *Handler) HandleCMSChangeCuratorReportTime(ctx context.Context, msg *tgbotapi.Message) {
	h.deleteUserMessage(msg)
	if !h.checkAdminPermission(ctx, msg.Chat.ID, msg.From.ID) {
		return
	}
	h.state.SetState(msg.From.ID, StateCMSWaitingCuratorReportTime)
	h.SendMessage(msg.Chat.ID, "Отправьте время отправки отчетов кураторам (МСК):", keyboards.BackButton)
}

func (h *Handler) HandleCMSChangeWeeklyReportDays(ctx context.Context, msg *tgbotapi.Message) {
	h.deleteUserMessage(msg)
	if !h.checkAdminPermission(ctx, msg.Chat.ID, msg.From.ID) {
		return
	}
	h.state.SetState(msg.From.ID, StateCMSWaitingWeeklyDays)
	h.SendMessage(msg.Chat.ID, "Отправьте дни для еженедельного отчета (например: 1,3,5):", keyboards.BackButton)
}

func (h *Handler) HandleCMSViewConfig(ctx context.Context, msg *tgbotapi.Message) {
	h.deleteUserMessage(msg)
	if !h.checkAdminPermission(ctx, msg.Chat.ID, msg.From.ID) {
		return
	}
	settingsStr := "👁️ <b>Текущая конфигурация:</b>\n\n<i>(Функция вывода настроек из базы в разработке)</i>"
	h.SendMessage(msg.Chat.ID, settingsStr, nil)
}

func (h *Handler) HandleCMSResetConfig(ctx context.Context, msg *tgbotapi.Message) {
	h.deleteUserMessage(msg)
	if !h.checkAdminPermission(ctx, msg.Chat.ID, msg.From.ID) {
		return
	}
	h.SendMessage(msg.Chat.ID, "🔄 Функция сброса настроек в разработке.", nil)
}
