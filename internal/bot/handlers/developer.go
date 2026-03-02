package handlers

import (
	"context"
	"fmt"

	"telegram_bot/internal/bot/keyboards"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (h *Handler) HandleDeveloperMenu(ctx context.Context, msg *tgbotapi.Message) {
	go func() {
		h.bot.Request(tgbotapi.NewDeleteMessage(msg.Chat.ID, msg.MessageID))
	}()
	h.SendMessage(msg.Chat.ID, "👨‍💻 Панель разработчика", keyboards.DeveloperMenu)
}

func (h *Handler) HandleSelectCurator(ctx context.Context, msg *tgbotapi.Message) {
	go func() {
		h.bot.Request(tgbotapi.NewDeleteMessage(msg.Chat.ID, msg.MessageID))
	}()

	curators, err := h.userSvc.GetAllCurators(ctx)
	if err != nil {
		h.log.Error("failed to fetch curators", "error", err)
		h.SendMessage(msg.Chat.ID, "❌ Ошибка при получении списка кураторов.", nil)
		return
	}

	if len(curators) == 0 {
		h.SendMessage(msg.Chat.ID, "📭 В системе пока нет зарегистрированных кураторов.", nil)
		return
	}

	var rows [][]tgbotapi.InlineKeyboardButton
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
