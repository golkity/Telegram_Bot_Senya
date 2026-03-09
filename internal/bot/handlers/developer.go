package handlers

import (
	"context"
	"fmt"

	"telegram_bot/internal/bot/keyboards"
	"telegram_bot/internal/modules/user"
	"telegram_bot/pkg/metrics"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (h *Handler) checkDeveloperPermission(ctx context.Context, chatID int64, userID int64) bool {
	u, err := h.userSvc.GetUserInfo(ctx, userID)
	if err != nil {
		metrics.ErrorsTotal.WithLabelValues("developer_handler").Inc()
	}
	if err != nil || (u.Role != user.RoleDeveloper && u.Role != user.RoleAdmin) {
		h.SendMessage(chatID, "⛔ У вас нет доступа к панели разработчика.", nil)
		return false
	}
	return true
}

func (h *Handler) HandleDeveloperMenu(ctx context.Context, msg *tgbotapi.Message) {
	h.deleteUserMessage(msg)

	if !h.checkDeveloperPermission(ctx, msg.Chat.ID, msg.From.ID) {
		return
	}

	h.SendMessage(msg.Chat.ID, "👨‍💻 Панель разработчика", keyboards.DeveloperMenu)
}

func (h *Handler) HandleSelectCurator(ctx context.Context, msg *tgbotapi.Message) {
	h.deleteUserMessage(msg)

	if !h.checkDeveloperPermission(ctx, msg.Chat.ID, msg.From.ID) {
		return
	}

	curators, err := h.userSvc.GetAllCurators(ctx)
	if err != nil {
		h.log.Error("failed to fetch curators", "error", err)
		metrics.ErrorsTotal.WithLabelValues("developer_handler").Inc()
		h.SendMessage(msg.Chat.ID, "❌ Ошибка при получении списка кураторов.", nil)
		return
	}

	if len(curators) == 0 {
		h.SendMessage(msg.Chat.ID, "📭 В системе пока нет зарегистрированных кураторов.", nil)
		return
	}

	rows := make([][]tgbotapi.InlineKeyboardButton, 0, len(curators))
	for _, c := range curators {
		btnText := fmt.Sprintf("%s %s", c.FirstName, c.LastName)
		data := fmt.Sprintf("dev_attach:%d", c.ID)
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(btnText, data),
		))
	}

	h.SendMessage(msg.Chat.ID, "👥 Выберите куратора, к которому хотите прикрепиться:",
		tgbotapi.NewInlineKeyboardMarkup(rows...))
}
