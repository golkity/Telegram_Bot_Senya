package handlers

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"telegram_bot/internal/bot/keyboards"
	"telegram_bot/internal/modules/user"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (h *Handler) checkAdminPermission(ctx context.Context, chatID int64, userID int64) bool {
	u, err := h.userSvc.GetUserInfo(ctx, userID)
	if err != nil || u.Role != user.RoleAdmin {
		h.SendMessage(chatID, "⛔ У вас нет прав администратора.", nil)
		return false
	}
	return true
}

func (h *Handler) HandleAdminPanel(ctx context.Context, msg *tgbotapi.Message) {
	if !h.checkAdminPermission(ctx, msg.Chat.ID, msg.From.ID) {
		return
	}
	h.SendMessage(msg.Chat.ID, "👨‍💼 Админ-панель", keyboards.AdminMenu)
}

func (h *Handler) HandleUserManagementMenu(ctx context.Context, msg *tgbotapi.Message) {
	if !h.checkAdminPermission(ctx, msg.Chat.ID, msg.From.ID) {
		return
	}
	h.SendMessage(msg.Chat.ID, "👤 Меню управления пользователями:", keyboards.UserManagementMenu)
}

func (h *Handler) HandleRoleManagement(ctx context.Context, msg *tgbotapi.Message) {
	if !h.checkAdminPermission(ctx, msg.Chat.ID, msg.From.ID) {
		return
	}
	h.SendMessage(msg.Chat.ID, "⚙️ Управление ролями:", keyboards.RoleManagementMenu)
}

func (h *Handler) HandleCustomization(ctx context.Context, msg *tgbotapi.Message) {
	if !h.checkAdminPermission(ctx, msg.Chat.ID, msg.From.ID) {
		return
	}
	h.SendMessage(msg.Chat.ID, "🎨 Кастомизация бота:", keyboards.CustomizationMenu)
}

func (h *Handler) HandleUserStatisticsRequest(ctx context.Context, msg *tgbotapi.Message) {
	if !h.checkAdminPermission(ctx, msg.Chat.ID, msg.From.ID) {
		return
	}
	h.state.SetState(msg.From.ID, StateWaitingForInputUserStats)
	h.SendMessage(msg.Chat.ID, "Введите ID пользователя или Username (без @) для просмотра статистики:", keyboards.BackButton)
}

func (h *Handler) HandleStartDeleteUser(ctx context.Context, msg *tgbotapi.Message) {
	if !h.checkAdminPermission(ctx, msg.Chat.ID, msg.From.ID) {
		return
	}
	h.state.SetState(msg.From.ID, StateWaitingForDeleteInput)
	h.SendMessage(msg.Chat.ID, "Введите ID пользователя, которого нужно удалить:", keyboards.BackButton)
}

func (h *Handler) HandleDeleteUserInput(ctx context.Context, msg *tgbotapi.Message) {
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
		targetID := h.state.GetData(msg.From.ID, "delete_target_id").(int64)

		err := h.userSvc.DeleteUser(ctx, targetID)
		if err != nil {
			h.SendMessage(msg.Chat.ID, "❌ Ошибка при удалении.", keyboards.UserManagementMenu)
		} else {
			h.SendMessage(msg.Chat.ID, "✅ Пользователь успешно удален.", keyboards.UserManagementMenu)
		}

		h.state.ClearState(msg.From.ID)
		h.state.ClearData(msg.From.ID)
	}
}

func (h *Handler) HandleBulkTransfer(ctx context.Context, msg *tgbotapi.Message) {
	if !h.checkAdminPermission(ctx, msg.Chat.ID, msg.From.ID) {
		return
	}
	h.SendMessage(msg.Chat.ID, "Выберите тип переноса:", keyboards.BulkTransferMenu)
}

func (h *Handler) HandleStartTransferByCurator(ctx context.Context, msg *tgbotapi.Message) {
	if !h.checkAdminPermission(ctx, msg.Chat.ID, msg.From.ID) {
		return
	}
	h.state.SetState(msg.From.ID, StateWaitingForTransferSource)
	h.SendMessage(msg.Chat.ID, "Введите ID куратора, ОТ КОТОРОГО нужно забрать учеников:", keyboards.BackButton)
}

func (h *Handler) HandleTransferSourceInput(ctx context.Context, msg *tgbotapi.Message) {
	if !h.checkAdminPermission(ctx, msg.Chat.ID, msg.From.ID) {
		return
	}
	sourceID, _ := strconv.ParseInt(msg.Text, 10, 64)
	h.state.SetData(msg.From.ID, "transfer_source", sourceID)

	h.state.SetState(msg.From.ID, StateWaitingForTransferTarget)
	h.SendMessage(msg.Chat.ID, "Теперь введите ID куратора, КОМУ передать учеников:", keyboards.BackButton)
}

func (h *Handler) HandleTransferTargetInput(ctx context.Context, msg *tgbotapi.Message) {
	if !h.checkAdminPermission(ctx, msg.Chat.ID, msg.From.ID) {
		return
	}
	targetID, _ := strconv.ParseInt(msg.Text, 10, 64)
	sourceID := h.state.GetData(msg.From.ID, "transfer_source").(int64)

	h.state.SetData(msg.From.ID, "transfer_target", targetID)
	h.state.SetState(msg.From.ID, StateWaitingForTransferConfirm)

	text := fmt.Sprintf("⚠️ Подтвердите перенос:\n\nОт куратора ID: %d\nК куратору ID: %d", sourceID, targetID)

	h.SendMessage(msg.Chat.ID, text, keyboards.ConfirmTransferMenu)
}

func (h *Handler) HandleTransferConfirm(ctx context.Context, msg *tgbotapi.Message) {
	if !h.checkAdminPermission(ctx, msg.Chat.ID, msg.From.ID) {
		return
	}

	if msg.Text == "❌ Отмена" {
		h.state.ClearState(msg.From.ID)
		h.SendMessage(msg.Chat.ID, "Перенос отменен.", keyboards.AdminMenu)
		return
	}

	if msg.Text == "✅ Подтвердить перенос" {
		sourceID := h.state.GetData(msg.From.ID, "transfer_source").(int64)
		targetID := h.state.GetData(msg.From.ID, "transfer_target").(int64)

		err := h.userSvc.TransferStudents(ctx, sourceID, targetID)
		if err != nil {
			h.SendMessage(msg.Chat.ID, "❌ Ошибка при переносе.", keyboards.AdminMenu)
		} else {
			h.SendMessage(msg.Chat.ID, fmt.Sprintf("✅ Заявка на перенос от %d к %d выполнена.", sourceID, targetID), keyboards.AdminMenu)
		}
		h.state.ClearState(msg.From.ID)
	}
}

func (h *Handler) HandleAdminDailyReport(ctx context.Context, msg *tgbotapi.Message) {
	if !h.checkAdminPermission(ctx, msg.Chat.ID, msg.From.ID) {
		return
	}
	h.SendMessage(msg.Chat.ID, "📊 Генерирую ежедневный отчет...", nil)
	go func() {
		stats, err := h.userSvc.GetDailyStats(ctx, 0)
		if err != nil {
			h.SendMessage(msg.Chat.ID, "Ошибка сбора статистики", nil)
			return
		}

		text := fmt.Sprintf("📊 Отчет за %s\nФайлов: %d\nДЗ: %s",
			time.Now().Format("02.01"), stats.TotalFilesToday, stats.HomeworkStatus)
		h.SendMessage(msg.Chat.ID, text, nil)
	}()
}

func (h *Handler) HandleAdminWeeklyReport(ctx context.Context, msg *tgbotapi.Message) {
	if !h.checkAdminPermission(ctx, msg.Chat.ID, msg.From.ID) {
		return
	}
	err := h.reportSvc.RequestReport(ctx, msg.From.ID, "admin_weekly")
	if err != nil {
		return
	}
	h.SendMessage(msg.Chat.ID, "Запрос отправлен.", nil)
}

func (h *Handler) HandleAdminDetailedExcel(ctx context.Context, msg *tgbotapi.Message) {
	if !h.checkAdminPermission(ctx, msg.Chat.ID, msg.From.ID) {
		return
	}
	err := h.reportSvc.RequestReport(ctx, msg.From.ID, "admin_detailed_excel")
	if err != nil {
		return
	}
	h.SendMessage(msg.Chat.ID, "Запрос отправлен.", nil)
}

func (h *Handler) HandleAdminWordReport(ctx context.Context, msg *tgbotapi.Message) {
	if !h.checkAdminPermission(ctx, msg.Chat.ID, msg.From.ID) {
		return
	}
	err := h.reportSvc.RequestReport(ctx, msg.From.ID, "word_weekly")
	if err != nil {
		return
	}
	h.SendMessage(msg.Chat.ID, "Запрос отправлен.", nil)
}

func (h *Handler) HandleAdminSubmissionExcel(ctx context.Context, msg *tgbotapi.Message) {
	if !h.checkAdminPermission(ctx, msg.Chat.ID, msg.From.ID) {
		return
	}
	err := h.reportSvc.RequestReport(ctx, msg.From.ID, "excel_submissions")
	if err != nil {
		return
	}
	h.SendMessage(msg.Chat.ID, "Запрос отправлен.", nil)
}

func (h *Handler) HandleAdminStudentSheets(ctx context.Context, msg *tgbotapi.Message) {
	if !h.checkAdminPermission(ctx, msg.Chat.ID, msg.From.ID) {
		return
	}
	err := h.reportSvc.RequestReport(ctx, msg.From.ID, "admin_student_sheets")
	if err != nil {
		return
	}
	h.SendMessage(msg.Chat.ID, "Запрос отправлен.", nil)
}

func (h *Handler) HandleSendGlobalReminders(ctx context.Context, msg *tgbotapi.Message) {
	if !h.checkAdminPermission(ctx, msg.Chat.ID, msg.From.ID) {
		return
	}
	go func() {
		_ = h.userSvc.SendGlobalReminders(ctx)
	}()
	h.SendMessage(msg.Chat.ID, "🔔 Глобальная рассылка запущена.", nil)
}

func (h *Handler) HandleAdminToggleNotifications(ctx context.Context, msg *tgbotapi.Message) {
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
	if !h.checkAdminPermission(ctx, msg.Chat.ID, msg.From.ID) {
		return
	}
	stats, _ := h.userSvc.GetCuratorsStats(ctx)
	h.SendMessage(msg.Chat.ID, stats, nil)
}

func (h *Handler) HandleAdminStudentsByCourse(ctx context.Context, msg *tgbotapi.Message) {
	if !h.checkAdminPermission(ctx, msg.Chat.ID, msg.From.ID) {
		return
	}
	h.SendMessage(msg.Chat.ID, "Функция в разработке", nil)
}
