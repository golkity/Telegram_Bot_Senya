package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"telegram_bot/internal/bot/keyboards"
	"telegram_bot/internal/infra/word"
	"telegram_bot/internal/modules/cms"
	"telegram_bot/internal/modules/report"
	"telegram_bot/internal/modules/submission"
	"telegram_bot/internal/modules/user"
	"telegram_bot/pkg/metrics"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Handler struct {
	userSvc       *user.Service
	submissionSvc *submission.Service
	reportSvc     *report.Service
	cmsSvc        *cms.Service

	state   StateContext
	bot     *tgbotapi.BotAPI
	log     *slog.Logger
	wordGen *word.Generator

	lastBotMessages sync.Map
}

func NewHandler(
	userSvc *user.Service,
	submissionSvc *submission.Service,
	reportSvc *report.Service,
	cmsSvc *cms.Service,
	state StateContext,
	bot *tgbotapi.BotAPI,
	log *slog.Logger,
	wordGen *word.Generator,
) *Handler {
	return &Handler{
		userSvc:       userSvc,
		submissionSvc: submissionSvc,
		reportSvc:     reportSvc,
		cmsSvc:        cmsSvc,
		state:         state,
		bot:           bot,
		log:           log,
		wordGen:       wordGen,
	}
}

func (h *Handler) SendMessage(chatID int64, text string, kb interface{}) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"
	if kb != nil {
		msg.ReplyMarkup = kb
	}
	if _, err := h.bot.Send(msg); err != nil {
		h.log.Error("failed to send message", "chat_id", chatID, "error", err)
		metrics.ErrorsTotal.WithLabelValues("telegram_api").Inc()
	}
}

func (h *Handler) SendCleanMessage(chatID int64, text string, kb interface{}) {
	if oldMsgVal, ok := h.lastBotMessages.Load(chatID); ok {
		oldMsgID := oldMsgVal.(int)
		go func() {
			_, err := h.bot.Request(tgbotapi.NewDeleteMessage(chatID, oldMsgID))
			if err != nil {
				h.log.Debug("could not delete previous bot message", "err", err)
			}
		}()
	}

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"
	if kb != nil {
		msg.ReplyMarkup = kb
	}

	sentMsg, err := h.bot.Send(msg)
	if err != nil {
		h.log.Error("failed to send clean message", "chat_id", chatID, "error", err)
		metrics.ErrorsTotal.WithLabelValues("telegram_api").Inc()
		return
	}

	h.lastBotMessages.Store(chatID, sentMsg.MessageID)
}

func (h *Handler) EditMessageText(chatID int64, messageID int, text string, kb interface{}) {
	msg := tgbotapi.NewEditMessageText(chatID, messageID, text)
	msg.ParseMode = "HTML"
	if kb != nil {
		if markup, ok := kb.(tgbotapi.InlineKeyboardMarkup); ok {
			msg.ReplyMarkup = &markup
		}
	}
	if _, err := h.bot.Send(msg); err != nil {
		h.log.Error("failed to edit message", "chat_id", chatID, "msg_id", messageID, "error", err)
		metrics.ErrorsTotal.WithLabelValues("telegram_api").Inc()
	}
}

func (h *Handler) SendFile(chatID int64, fileData interface{}, fileName string, caption string) {
	if fileData == nil {
		if caption != "" {
			h.SendMessage(chatID, caption, nil)
		}
		return
	}

	var fileRequest tgbotapi.Chattable

	switch data := fileData.(type) {
	case string:
		if data == "" {
			if caption != "" {
				h.SendMessage(chatID, caption, nil)
			}
			return
		}
		doc := tgbotapi.NewDocument(chatID, tgbotapi.FileID(data))
		doc.Caption = caption
		fileRequest = doc
	case []byte:
		if data == nil || len(data) == 0 {
			if caption != "" {
				h.SendMessage(chatID, caption, nil)
			}
			return
		}
		fileBytes := tgbotapi.FileBytes{
			Name:  fileName,
			Bytes: data,
		}
		doc := tgbotapi.NewDocument(chatID, fileBytes)
		doc.Caption = caption
		fileRequest = doc
	default:
		if caption != "" {
			h.SendMessage(chatID, caption, nil)
		}
		return
	}

	if fileRequest != nil {
		if _, err := h.bot.Send(fileRequest); err != nil {
			h.log.Error("failed to send file", "chat_id", chatID, "error", err)
			metrics.ErrorsTotal.WithLabelValues("telegram_api").Inc()
		}
	}
}

func (h *Handler) AnswerCallback(callbackID string, text string) {
	resp := tgbotapi.NewCallback(callbackID, text)
	if _, err := h.bot.Request(resp); err != nil {
		h.log.Error("failed to answer callback", "callback_id", callbackID, "error", err)
		metrics.ErrorsTotal.WithLabelValues("telegram_api").Inc()
	}
}

func (h *Handler) HandleGenericText(ctx context.Context, msg *tgbotapi.Message) {
	rawState := h.state.GetState(msg.From.ID)
	state := UserState(rawState)

	switch state {
	case StateWaitingForTask:
		h.HandleTaskSelection(ctx, msg)
	case StateWaitingForSubtask:
		h.HandleSubtaskSelection(ctx, msg)
	case StateWaitingForComment:
		h.HandleSubmissionFinalize(ctx, msg)
	case StateWaitingForWeeklyReport:
		h.HandleWeeklyReportText(ctx, msg)
	case StateWaitingForCourseName:
		h.HandleCourseCreationFlow(ctx, msg)
	case StateWaitingForCourseSelection:
		h.HandleStudentJoinCourse(ctx, msg)
	case StateWaitingForCuratorSelection:
		h.HandleStudentJoinCurator(ctx, msg)
	case StateWaitingForReminderText:
		h.HandleCuratorSendReminderText(ctx, msg)
	case StateWaitingForBotName, StateWaitingForWelcomeText, StateWaitingForReportTime:
		h.HandleCustomizationText(ctx, msg)
	case StateWaitingForFiles:
		h.HandleFileUpload(ctx, msg)
	case StateWaitingForDeleteInput:
		h.HandleDeleteUserInput(ctx, msg)
	case StateWaitingForInputUserStats:
		h.deleteUserMessage(msg)
		h.SendCleanMessage(msg.Chat.ID, "Поиск по тексту пока не реализован, используйте меню.", nil)
	case StateWaitingForTransferSource:
		h.HandleTransferSourceInput(ctx, msg)
	case StateWaitingForTransferTarget:
		h.HandleTransferTargetInput(ctx, msg)

	case StateCMSWaitingBotName, StateCMSWaitingGreeting, StateCMSWaitingButtons,
		StateCMSWaitingMessages, StateCMSWaitingReportTime, StateCMSWaitingReminderTime,
		StateCMSWaitingCheckTime, StateCMSWaitingCuratorReportTime, StateCMSWaitingWeeklyDays:

		h.deleteUserMessage(msg)

		keyToUpdate := ""
		switch state {
		case StateCMSWaitingBotName:
			keyToUpdate = "bot_name"
		case StateCMSWaitingGreeting:
			keyToUpdate = "greeting_text"
		case StateCMSWaitingButtons:
			keyToUpdate = "button_texts"
		case StateCMSWaitingMessages:
			keyToUpdate = "system_messages"
		case StateCMSWaitingReportTime:
			keyToUpdate = "report_time"
		case StateCMSWaitingReminderTime:
			keyToUpdate = "reminder_time"
		case StateCMSWaitingCheckTime:
			keyToUpdate = "check_time"
		case StateCMSWaitingCuratorReportTime:
			keyToUpdate = "curator_report_time"
		case StateCMSWaitingWeeklyDays:
			keyToUpdate = "weekly_report_days"
		}

		newValue := msg.Text

		err := h.cmsSvc.UpdateSetting(ctx, keyToUpdate, newValue)
		if err != nil {
			h.log.Error("failed to update cms setting", "key", keyToUpdate, "err", err)
			metrics.ErrorsTotal.WithLabelValues("cms_db").Inc()
			h.SendCleanMessage(msg.Chat.ID, "❌ Ошибка при сохранении настройки.", keyboards.CustomizationMenu)
		} else {
			h.SendCleanMessage(msg.Chat.ID, fmt.Sprintf("✅ Настройка <b>%s</b> успешно обновлена!\n\n<i>Новое значение:</i> %s", keyToUpdate, newValue), keyboards.CustomizationMenu)
		}

		h.state.ClearState(msg.From.ID)

	default:
		h.HandleUnknown(ctx, msg)
	}
}

func (h *Handler) HandleWeeklyReportText(ctx context.Context, msg *tgbotapi.Message) {
	h.deleteUserMessage(msg)
	reportText := msg.Text
	h.log.Info("received weekly report", "user_id", msg.From.ID, "text", reportText)

	h.state.ClearState(msg.From.ID)
	h.SendCleanMessage(msg.Chat.ID, "✅ Ваш еженедельный отчет принят и сохранен.", keyboards.StudentMenu)
}

func (h *Handler) HandleCourseCreationFlow(ctx context.Context, msg *tgbotapi.Message) {
	h.deleteUserMessage(msg)
	courseName := msg.Text
	h.log.Info("creating new course", "name", courseName)

	h.state.ClearState(msg.From.ID)
	h.SendCleanMessage(msg.Chat.ID, "✅ Курс '"+courseName+"' успешно создан.", keyboards.AdminMenu)
}

func (h *Handler) HandleCuratorSendReminderText(ctx context.Context, msg *tgbotapi.Message) {
	h.deleteUserMessage(msg)
	reminderText := msg.Text
	curatorID := msg.From.ID

	err := h.userSvc.BroadcastToStudents(ctx, curatorID, reminderText)
	if err != nil {
		h.log.Error("failed to broadcast reminder", "curator_id", curatorID, "error", err)
		metrics.ErrorsTotal.WithLabelValues("bot_broadcast").Inc()
		h.SendCleanMessage(msg.Chat.ID, "❌ Ошибка при отправке рассылки.", nil)
	} else {
		h.SendCleanMessage(msg.Chat.ID, "✅ Напоминание отправлено всем вашим студентам.", keyboards.CuratorMenu)
	}

	h.state.ClearState(curatorID)
}

func (h *Handler) HandleCustomizationText(ctx context.Context, msg *tgbotapi.Message) {
	h.deleteUserMessage(msg)
	settingValue := msg.Text
	state := h.state.GetState(msg.From.ID)

	h.log.Info("updating bot customization", "admin_id", msg.From.ID, "state", state, "value", settingValue)

	h.state.ClearState(msg.From.ID)
	h.SendCleanMessage(msg.Chat.ID, "✅ Настройки бота успешно обновлены.", keyboards.CustomizationMenu)
}

func (h *Handler) HandleUnknown(ctx context.Context, msg *tgbotapi.Message) {
	h.deleteUserMessage(msg)
	h.SendCleanMessage(msg.Chat.ID, "Я не понимаю это сообщение. Пожалуйста, используйте меню.", nil)
}
