package handlers

import (
	"context"
	"telegram_bot/internal/bot/keyboards"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (h *Handler) HandleBack(ctx context.Context, msg *tgbotapi.Message) {
	h.deleteUserMessage(msg)

	rawState := h.state.GetState(msg.From.ID)
	state := UserState(rawState)

	switch state {
	case StateCMSWaitingBotName, StateCMSWaitingGreeting, StateCMSWaitingButtons,
		StateCMSWaitingMessages, StateCMSWaitingReportTime, StateCMSWaitingReminderTime,
		StateCMSWaitingCheckTime, StateCMSWaitingCuratorReportTime, StateCMSWaitingWeeklyDays:

		h.HandleBackToCustomization(ctx, msg)
		return
	}

	h.state.ClearState(msg.From.ID)
	h.state.ClearData(msg.From.ID)

	h.HandleStart(ctx, msg)
}

func (h *Handler) HandleCancel(ctx context.Context, msg *tgbotapi.Message) {
	h.deleteUserMessage(msg)
	h.SendMessage(msg.Chat.ID, "❌ Действие отменено.", nil)
	h.HandleBack(ctx, msg)
}

func (h *Handler) HandleBackToAdmin(ctx context.Context, msg *tgbotapi.Message) {
	h.deleteUserMessage(msg)
	h.state.ClearState(msg.From.ID)
	h.state.ClearData(msg.From.ID)
	h.SendMessage(msg.Chat.ID, "👨‍💼 Админ-панель", keyboards.AdminMenu)
}

func (h *Handler) HandleBackToCustomization(ctx context.Context, msg *tgbotapi.Message) {
	h.deleteUserMessage(msg)
	h.state.ClearState(msg.From.ID)
	h.SendMessage(msg.Chat.ID, "🎨 Кастомизация:", keyboards.CustomizationMenu)
}

func (h *Handler) HandleBackToRoles(ctx context.Context, msg *tgbotapi.Message) {
	h.deleteUserMessage(msg)
	h.state.ClearState(msg.From.ID)
	h.SendMessage(msg.Chat.ID, "⚙️ Управление ролями:", keyboards.RoleManagementMenu)
}

func (h *Handler) HandleBackToUserList(ctx context.Context, msg *tgbotapi.Message) {
	h.deleteUserMessage(msg)
	h.state.ClearState(msg.From.ID)
	h.HandleUserManagementMenu(ctx, msg)
}
