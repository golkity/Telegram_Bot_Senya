package handlers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"telegram_bot/internal/bot/keyboards"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (h *Handler) HandleStudentJoinCourse(ctx context.Context, msg *tgbotapi.Message) {
	text := msg.Text
	courseName := strings.TrimPrefix(text, "📚 ")

	course, err := h.userSvc.GetCourseByName(ctx, courseName)
	if err != nil {
		h.SendMessage(msg.Chat.ID, "❌ Курс не найден. Пожалуйста, выберите курс, используя кнопки меню.", nil)
		return
	}

	err = h.userSvc.SetUserCourse(ctx, msg.From.ID, course.ID)
	if err != nil {
		h.log.Error("failed to set user course", "user_id", msg.From.ID, "error", err)
		h.SendMessage(msg.Chat.ID, "❌ Ошибка при записи на курс. Попробуйте позже.", nil)
		return
	}

	h.state.ClearState(msg.From.ID)

	h.SendMessage(msg.Chat.ID, fmt.Sprintf("✅ Вы успешно записаны на курс: **%s**", course.Name), nil)

	h.HandleCuratorSelectionForStudent(ctx, msg)
}

func (h *Handler) HandleGetArchive(ctx context.Context, msg *tgbotapi.Message) {
	h.SendMessage(msg.Chat.ID, "📦 Запрос принят. Начинаю сборку архива всех ваших файлов...\nЭто может занять несколько минут.", nil)

	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		archiveURL, err := h.submissionSvc.GenerateUserArchive(bgCtx, msg.From.ID)
		if err != nil {
			h.log.Error("failed to generate archive", "user_id", msg.From.ID, "error", err)
			h.SendMessage(msg.Chat.ID, "❌ Ошибка при создании архива. Попробуйте позже или обратитесь к администратору.", nil)
			return
		}

		text := fmt.Sprintf("✅ **Ваш архив готов!**\n\n🔗 [Скачать архив](%s)\n\nСсылка действительна 24 часа.", archiveURL)

		msgObj := tgbotapi.NewMessage(msg.Chat.ID, text)
		msgObj.ParseMode = "Markdown"
		h.bot.Send(msgObj)
	}()
}

func (h *Handler) HandleDailyStatistics(ctx context.Context, msg *tgbotapi.Message) {
	stats, err := h.userSvc.GetDailyStats(ctx, msg.From.ID)
	if err != nil {
		h.log.Error("failed to get stats", "error", err)
		h.SendMessage(msg.Chat.ID, "⚠️ Не удалось загрузить статистику.", nil)
		return
	}

	text := fmt.Sprintf("📊 **Статистика за сегодня (%s):**\n\n"+
		"📚 ДЗ: %s\n"+
		"📝 Конспект: %s\n\n"+
		"Сдано заданий: %d",
		time.Now().Format("02.01.2006"),
		stats.HomeworkStatus,
		stats.NotesStatus,
		stats.TotalFilesToday,
	)

	h.SendMessage(msg.Chat.ID, text, nil)
}

func (h *Handler) HandleWeeklyReportSubmission(ctx context.Context, msg *tgbotapi.Message) {
	if !h.reportSvc.IsReportDay(time.Now().Weekday()) {
		h.SendMessage(msg.Chat.ID, "❌ Еженедельные отчеты принимаются только в Воскресенье и Понедельник.", keyboards.StudentMenu)
		return
	}

	h.state.SetState(msg.From.ID, StateWaitingForWeeklyReport)
	h.SendMessage(msg.Chat.ID,
		"📝 **Еженедельный отчет куратору**\n\n"+
			"Напишите одним сообщением:\n"+
			"1. Что изучили за неделю?\n"+
			"2. Какие были трудности?\n"+
			"3. Планы на следующую неделю.",
		keyboards.CancelButton,
	)
}
