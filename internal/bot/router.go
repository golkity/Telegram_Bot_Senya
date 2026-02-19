package bot

import (
	"context"
	"log/slog"
	"time"

	"telegram_bot/internal/bot/handlers"
	"telegram_bot/internal/modules/submission"
	"telegram_bot/internal/modules/telemetry"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Router struct {
	bot       *tgbotapi.BotAPI
	log       *slog.Logger
	handler   *handlers.Handler
	telemetry *telemetry.LatencyOptimizer
}

func NewRouter(
	bot *tgbotapi.BotAPI,
	log *slog.Logger,
	handler *handlers.Handler,
	telemetry *telemetry.LatencyOptimizer,
) *Router {
	return &Router{
		bot:       bot,
		log:       log,
		handler:   handler,
		telemetry: telemetry,
	}
}

func (r *Router) Start() {
	r.log.Info("Bot started")

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := r.bot.GetUpdatesChan(u)
	safeHandler := r.WithMiddleware(r.handleUpdate)

	for update := range updates {
		go func(upd tgbotapi.Update) {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()
			safeHandler(ctx, upd)
		}(update)
	}
}

func (r *Router) handleUpdate(ctx context.Context, update tgbotapi.Update) {
	if !r.telemetry.IsThroughputStable() {
		if update.Message != nil {
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "⚠️ Служба временно недоступна.\n\nПроводится плановая оптимизация базы данных.")
			_, _ = r.bot.Send(msg)
		} else if update.CallbackQuery != nil {
			alert := tgbotapi.NewCallback(update.CallbackQuery.ID, "Maintenance mode active")
			_, _ = r.bot.Request(alert)
		}
		return
	}

	if update.CallbackQuery != nil {
		r.handler.HandleCallback(ctx, update.CallbackQuery)
		return
	}

	if update.Message != nil {
		r.handleMessage(ctx, update.Message)
	}
}

func (r *Router) handleMessage(ctx context.Context, msg *tgbotapi.Message) {
	if msg.IsCommand() {
		switch msg.Command() {
		case "start":
			r.handler.HandleStart(ctx, msg)
		case "admin":
			r.handler.HandleAdminPanel(ctx, msg)
		case "curator":
			r.handler.HandleCuratorMenu(ctx, msg)
		case "developer":
			r.handler.HandleDeveloperMenu(ctx, msg)
		default:
			r.handler.HandleUnknown(ctx, msg)
		}
		return
	}

	switch msg.Text {
	case "↩️ Назад", "Отмена", "↩️ Назад в меню":
		r.handler.HandleBack(ctx, msg)
		return
	case "↩️ Назад в админ-панель":
		r.handler.HandleBackToAdmin(ctx, msg)
		return
	case "↩️ Назад к ролям":
		r.handler.HandleBackToRoles(ctx, msg)
		return
	case "↩️ Назад к списку":
		r.handler.HandleBackToUserList(ctx, msg)
		return
	}

	switch msg.Text {
	case "📚 Сдать ДЗ":
		r.handler.HandleSubmissionStart(ctx, msg, submission.TypeHomework)
	case "📝 Сдать конспект":
		r.handler.HandleSubmissionStart(ctx, msg, submission.TypeNotes)
	case "📦 Получить архив":
		r.handler.HandleGetArchive(ctx, msg)
	case "📈 Ежедневная статистика":
		r.handler.HandleDailyStatistics(ctx, msg)
	case "📝 Еженедельный отчет куратору":
		r.handler.HandleWeeklyReportSubmission(ctx, msg)
	case "Готово":
		r.handler.HandleSubmissionDone(ctx, msg)
	case "Пропустить":
		r.handler.HandleSubmissionFinalize(ctx, msg)

	case "📝 Отправить напоминание":
		r.handler.HandleCuratorReminder(ctx, msg)
	case "📊 Ежедневный отчет":
		r.handler.HandleCuratorDailyReport(ctx, msg)
	case "📈 Еженедельный отчет":
		r.handler.HandleCuratorWeeklyReport(ctx, msg)
	case "📋 Word отчет (еженедельный)":
		r.handler.HandleWordReport(ctx, msg)
	case "📋 Отчет по сдаче (Excel)":
		r.handler.HandleSubmissionReportExcel(ctx, msg)
	case "👤 Мои ученики":
		r.handler.HandleCuratorMyStudents(ctx, msg)
	case "📊 Сводный отчет Excel":
		r.handler.HandleSummaryExcel(ctx, msg)
	case "📁 Просмотреть работы учеников":
		r.handler.HandleViewStudentWorks(ctx, msg)

	case "📈 Ежедневный отчет (Админ)":
		r.handler.HandleAdminDailyReport(ctx, msg)
	case "📋Подробный отчет Excel":
		r.handler.HandleAdminDetailedExcel(ctx, msg)
	case "👥 Листы по ученикам":
		r.handler.HandleAdminStudentSheets(ctx, msg)
	case "👤 Управление пользователями":
		r.handler.HandleUserManagementMenu(ctx, msg)
	case "👥 Управление ролями":
		r.handler.HandleRoleManagement(ctx, msg)
	case "⏰ Отправить напоминания":
		r.handler.HandleSendGlobalReminders(ctx, msg)
	case "🎨 Кастомизация бота":
		r.handler.HandleCustomization(ctx, msg)
	case "🔔 Уведомления администраторам":
		r.handler.HandleAdminToggleNotifications(ctx, msg)
	case "👨‍🏫 Перенос учеников":
		r.handler.HandleBulkTransfer(ctx, msg)
	case "📊 Статистика кураторов":
		r.handler.HandleAdminCuratorStats(ctx, msg)
	case "📚 Ученики по курсам":
		r.handler.HandleAdminStudentsByCourse(ctx, msg)

	case "🗑️ Удалить пользователя":
		r.handler.HandleStartDeleteUser(ctx, msg)
	case "📊 Статистика пользователя":
		r.handler.HandleUserStatisticsRequest(ctx, msg)
	case "✅ Да, удалить":
		r.handler.HandleDeleteConfirm(ctx, msg)
	case "❌ Нет, отмена":
		r.handler.HandleDeleteConfirm(ctx, msg)

	case "👥 Перенести всех учеников куратора":
		r.handler.HandleStartTransferByCurator(ctx, msg)
	case "✅ Подтвердить перенос":
		r.handler.HandleTransferConfirm(ctx, msg)

	case "👥 Выбрать куратора":
		r.handler.HandleSelectCurator(ctx, msg)

	default:
		r.handler.HandleGenericText(ctx, msg)
	}
}
