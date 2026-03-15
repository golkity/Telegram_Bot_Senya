package handlers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"telegram_bot/internal/bot/keyboards"
	"telegram_bot/pkg/metrics"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (h *Handler) HandleStudentJoinCourse(ctx context.Context, msg *tgbotapi.Message) {
	h.deleteUserMessage(msg)

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
		metrics.ErrorsTotal.WithLabelValues("student_handler").Inc()
		h.SendMessage(msg.Chat.ID, "❌ Ошибка при записи на курс. Попробуйте позже.", nil)
		return
	}

	h.state.ClearState(msg.From.ID)

	h.SendMessage(msg.Chat.ID, fmt.Sprintf("✅ Вы успешно записаны на курс: <b>%s</b>", course.Name), nil)

	h.HandleCuratorSelectionForStudent(ctx, msg)
}

func (h *Handler) HandleGetArchive(ctx context.Context, msg *tgbotapi.Message) {
	h.deleteUserMessage(msg)

	waitMsgConfig := tgbotapi.NewMessage(msg.Chat.ID, "📦 Запрос принят. Начинаю сборку архива всех ваших файлов...\nЭто может занять несколько минут.")
	waitMsg, _ := h.bot.Send(waitMsgConfig)

	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		archiveURL, err := h.submissionSvc.GenerateUserArchive(bgCtx, msg.From.ID)

		if waitMsg.MessageID != 0 {
			go func() {
				h.bot.Request(tgbotapi.NewDeleteMessage(msg.Chat.ID, waitMsg.MessageID))
			}()
		}

		if err != nil {
			h.log.Error("failed to generate archive", "user_id", msg.From.ID, "error", err)
			metrics.ErrorsTotal.WithLabelValues("student_handler").Inc()
			h.SendMessage(msg.Chat.ID, "❌ Ошибка при создании архива. Попробуйте позже или обратитесь к администратору.", nil)
			return
		}

		text := fmt.Sprintf("✅ <b>Ваш архив готов!</b>\n\n🔗 [Скачать архив](%s)\n\nСсылка действительна 24 часа.", archiveURL)

		msgObj := tgbotapi.NewMessage(msg.Chat.ID, text)
		msgObj.ParseMode = "Markdown"
		h.bot.Send(msgObj)
	}()
}

func (h *Handler) HandleDailyStatistics(ctx context.Context, msg *tgbotapi.Message) {
	h.deleteUserMessage(msg)

	stats, err := h.userSvc.GetDailyStats(ctx, msg.From.ID)
	if err != nil {
		h.log.Error("failed to get stats", "error", err)
		metrics.ErrorsTotal.WithLabelValues("student_handler").Inc()
		h.SendMessage(msg.Chat.ID, "⚠️ Не удалось загрузить статистику.", nil)
		return
	}

	text := fmt.Sprintf("📊 <b>Статистика за сегодня (%s):</b>\n\n"+
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
	h.deleteUserMessage(msg)

	if !h.reportSvc.IsReportDay(time.Now().Weekday()) {
		h.SendMessage(msg.Chat.ID, "❌ Еженедельные отчеты принимаются только в Воскресенье и Понедельник.", keyboards.StudentMenu)
		return
	}

	h.state.SetState(msg.From.ID, StateWaitingForWeeklyReport)
	h.SendMessage(msg.Chat.ID,
		"📝 <b>Еженедельный отчет куратору</b>\n\n"+
			"Напишите одним сообщением:\n"+
			"1. Что изучили за неделю?\n"+
			"2. Какие были трудности?\n"+
			"3. Планы на следующую неделю.",
		keyboards.CancelButton,
	)
}

func (h *Handler) HandleStudentJoinCurator(ctx context.Context, msg *tgbotapi.Message) {
	h.deleteUserMessage(msg)

	text := msg.Text
	curatorName := strings.TrimPrefix(text, "👤 ")

	u, err := h.userSvc.GetUserInfo(ctx, msg.From.ID)
	if err != nil || u == nil || u.CourseID == nil {
		h.SendMessage(msg.Chat.ID, "❌ Произошла ошибка. Начните заново: /start", nil)
		return
	}

	curators, err := h.userSvc.GetCuratorsByCourse(ctx, *u.CourseID)
	if err != nil {
		metrics.ErrorsTotal.WithLabelValues("student_handler").Inc()
		h.SendMessage(msg.Chat.ID, "❌ Ошибка при поиске кураторов.", nil)
		return
	}

	var targetCuratorID int64
	for _, c := range curators {
		if c.FirstName == curatorName {
			targetCuratorID = c.ID
			break
		}
	}

	if targetCuratorID == 0 {
		h.SendMessage(msg.Chat.ID, "❌ Куратор не найден. Пожалуйста, используйте кнопки.", nil)
		return
	}

	err = h.userSvc.AssignCurator(ctx, msg.From.ID, targetCuratorID, *u.CourseID)
	if err != nil {
		h.log.Error("failed to assign curator", "user_id", msg.From.ID, "error", err)
		metrics.ErrorsTotal.WithLabelValues("student_handler").Inc()
		h.SendMessage(msg.Chat.ID, "❌ Ошибка при сохранении куратора.", nil)
		return
	}

	h.state.ClearState(msg.From.ID)

	h.SendMessage(msg.Chat.ID, "✅ Куратор успешно выбран!", nil)
	h.SendMessage(msg.Chat.ID, fmt.Sprintf("👋 С возвращением, %s! Выбери действие:", u.FirstName), keyboards.StudentMenu)
}

func (h *Handler) HandleTaskStatus(ctx context.Context, msg *tgbotapi.Message, isDone bool) {
	h.deleteUserMessage(msg)
	h.state.ClearState(msg.From.ID)

	var response string
	if isDone {
		response = "🎉 Молодец! Отметка о выполнении принята.\n\nВозвращаю в главное меню 👇"
	} else {
		response = "💪 Ничего страшного, обязательно получится в следующий раз!\n\nВозвращаю в главное меню 👇"
	}

	h.SendCleanMessage(msg.Chat.ID, response, keyboards.StudentMenu)
}
