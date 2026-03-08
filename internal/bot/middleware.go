package bot

import (
	"context"
	"fmt"
	"log/slog"
	"runtime/debug"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type HandlerFunc func(ctx context.Context, update tgbotapi.Update)

func (r *Router) WithMiddleware(next HandlerFunc) HandlerFunc {
	return r.panicRecovery(r.logging(next))
}

func (r *Router) logging(next HandlerFunc) HandlerFunc {
	return func(ctx context.Context, update tgbotapi.Update) {
		start := time.Now()

		var (
			userID   int64
			username string
			action   string
			typeStr  string
		)

		switch {
		case update.Message != nil:
			userID = update.Message.From.ID
			username = update.Message.From.UserName
			action = update.Message.Text
			typeStr = "message"

			caption := update.Message.Caption

			if update.Message.Document != nil {
				action = strings.TrimSpace(fmt.Sprintf("[Document] %s %s", update.Message.Document.FileName, caption))
			} else if update.Message.Photo != nil {
				action = strings.TrimSpace(fmt.Sprintf("[Photo] %s", caption))
			} else if action == "" {
				action = strings.TrimSpace(fmt.Sprintf("[Other Media] %s", caption))
			}

		case update.CallbackQuery != nil:
			userID = update.CallbackQuery.From.ID
			username = update.CallbackQuery.From.UserName
			action = update.CallbackQuery.Data
			typeStr = "callback"

		case update.EditedMessage != nil:
			userID = update.EditedMessage.From.ID
			username = update.EditedMessage.From.UserName
			action = "[Edit] " + update.EditedMessage.Text
			typeStr = "edited_msg"

		default:
			typeStr = "unknown"
		}

		defer func() {
			r.log.Info("update processed",
				slog.String("type", typeStr),
				slog.Int64("user_id", userID),
				slog.String("username", username),
				slog.String("action", action),
				slog.Int("update_id", update.UpdateID),
				slog.Duration("duration", time.Since(start)),
			)
		}()

		next(ctx, update)
	}
}

func (r *Router) panicRecovery(next HandlerFunc) HandlerFunc {
	return func(ctx context.Context, update tgbotapi.Update) {
		defer func() {
			if err := recover(); err != nil {
				r.log.Error("PANIC RECOVERED",
					slog.Any("error", err),
					slog.String("stack", string(debug.Stack())),
					slog.Int("update_id", update.UpdateID),
				)

				if update.Message != nil {
					msg := tgbotapi.NewMessage(update.Message.Chat.ID, "⚠️ Произошла критическая ошибка сервера.")
					_, _ = r.bot.Send(msg)
				} else if update.CallbackQuery != nil {
					alert := tgbotapi.NewCallback(update.CallbackQuery.ID, "⚠️ Ошибка сервера")
					_, _ = r.bot.Request(alert)
				}
			}
		}()

		next(ctx, update)
	}
}
