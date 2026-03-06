package handlers

import (
	"context"
	"fmt"
	"sync"
	"time"

	"telegram_bot/internal/bot/keyboards"
	"telegram_bot/internal/modules/user"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var lastStartCall sync.Map

func (h *Handler) HandleStart(ctx context.Context, msg *tgbotapi.Message) {
	userID := msg.From.ID

	if lastCall, ok := lastStartCall.Load(userID); ok {
		if time.Since(lastCall.(time.Time)) < 2*time.Second {
			return
		}
	}
	lastStartCall.Store(userID, time.Now())

	if msg.MessageID != 0 {
		go func() {
			h.bot.Request(tgbotapi.NewDeleteMessage(msg.Chat.ID, msg.MessageID))
		}()
	}

	u := user.User{
		ID:        msg.From.ID,
		Username:  msg.From.UserName,
		FirstName: msg.From.FirstName,
		LastName:  msg.From.LastName,
		Role:      user.RoleStudent,
	}

	if err := h.userSvc.RegisterOrUpdate(ctx, u); err != nil {
		h.log.Error("registration failed", "error", err)
		h.SendMessage(msg.Chat.ID, "⚠️ Произошла ошибка системы. Пожалуйста, попробуй позже.", nil)
		return
	}

	actualUser, err := h.userSvc.GetUserInfo(ctx, u.ID)
	if err != nil {
		h.log.Error("failed to get user info", "error", err)
		actualUser = &u
	}

	h.routeUserByProfile(ctx, msg.Chat.ID, actualUser)
}

func (h *Handler) routeUserByProfile(ctx context.Context, chatID int64, u *user.User) {
	switch u.Role {
	case user.RoleAdmin:
		msg := fmt.Sprintf("👋 <b>Привет, %s!</b>\n\n⚙️ Панель администратора готова к работе. Выбирай нужное действие в меню 👇", u.FirstName)
		h.SendMessage(chatID, msg, keyboards.AdminMenu)

	case user.RoleCurator:
		msg := fmt.Sprintf("👨‍🏫 <b>Привет, %s!</b>\n\nТвои ученики и отчеты ждут. Выбирай действие в меню 👇", u.FirstName)
		h.SendMessage(chatID, msg, keyboards.CuratorMenu)

	case user.RoleDeveloper:
		h.handleDeveloperWelcome(ctx, chatID, u)

	case user.RoleStudent:
		h.handleStudentWelcome(ctx, chatID, u)

	default:
		h.SendMessage(chatID, "❌ Твоя роль не определена. Пожалуйста, обратись к администратору.", nil)
	}
}

func (h *Handler) handleStudentWelcome(ctx context.Context, chatID int64, u *user.User) {
	if u.CourseID == nil || *u.CourseID == "" {
		h.SendMessage(chatID, fmt.Sprintf("👋 <b>Привет, %s!</b> Добро пожаловать!\n\nДавай настроим твой профиль. Для начала <b>выбери свой курс</b> 👇", u.FirstName), nil)
		h.HandleCourseSelection(ctx, &tgbotapi.Message{
			Chat: &tgbotapi.Chat{ID: chatID},
			From: &tgbotapi.User{ID: u.ID},
		})
		return
	}

	if u.CuratorID == nil || *u.CuratorID == 0 {
		h.SendMessage(chatID, "✅ <b>Отлично! Курс сохранен.</b>\n\nТеперь последний шаг: <b>выбери своего куратора</b> 👇", nil)
		h.HandleCuratorSelectionForStudent(ctx, &tgbotapi.Message{
			Chat: &tgbotapi.Chat{ID: chatID},
			From: &tgbotapi.User{ID: u.ID},
		})
		return
	}

	msg := fmt.Sprintf("👋 <b>С возвращением, %s!</b>\n\nЧто будем делать сегодня? Выбирай действие 👇", u.FirstName)
	h.SendMessage(chatID, msg, keyboards.StudentMenu)
}

func (h *Handler) handleDeveloperWelcome(_ context.Context, chatID int64, u *user.User) {
	if u.CuratorID == nil || *u.CuratorID == 0 {
		msg := fmt.Sprintf("👨‍💻 <b>Привет, %s (Developer)!</b>\n\n⚠️ Чтобы тестировать сдачу ДЗ, тебе нужно <b>прикрепиться к куратору</b> через меню 👇", u.FirstName)
		h.SendMessage(chatID, msg, keyboards.DeveloperMenu)
		return
	}

	msg := fmt.Sprintf("👨‍💻 <b>Привет, %s!</b>\n\nРежим разработчика активен. Доступные инструменты в меню 👇", u.FirstName)
	h.SendMessage(chatID, msg, keyboards.DeveloperMenu)
}

func (h *Handler) HandleCourseSelection(ctx context.Context, msg *tgbotapi.Message) {
	if msg.MessageID != 0 {
		go func() {
			h.bot.Request(tgbotapi.NewDeleteMessage(msg.Chat.ID, msg.MessageID))
		}()
	}

	courses, err := h.userSvc.GetAllCourses(ctx)
	if err != nil {
		h.SendMessage(msg.Chat.ID, "❌ Ошибка загрузки курсов. Попробуй позже.", nil)
		return
	}

	var rows [][]tgbotapi.KeyboardButton
	for _, courseName := range courses {
		btnText := fmt.Sprintf("📚 %s", courseName)
		rows = append(rows, tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton(btnText)))
	}
	kb := tgbotapi.NewReplyKeyboard(rows...)

	h.state.SetState(msg.From.ID, StateWaitingForCourseSelection)
	h.SendMessage(msg.Chat.ID, "📚 <b>Доступные направления:</b>\nВыбери свой курс из меню ниже:", kb)
}

func (h *Handler) HandleCuratorSelectionForStudent(ctx context.Context, msg *tgbotapi.Message) {
	if msg.MessageID != 0 {
		go func() {
			h.bot.Request(tgbotapi.NewDeleteMessage(msg.Chat.ID, msg.MessageID))
		}()
	}

	u, err := h.userSvc.GetUserInfo(ctx, msg.From.ID)
	if err != nil || u == nil || u.CourseID == nil {
		h.SendMessage(msg.Chat.ID, "⚠️ Сначала нужно выбрать курс.", nil)
		h.HandleCourseSelection(ctx, msg)
		return
	}

	curators, err := h.userSvc.GetCuratorsByCourse(ctx, *u.CourseID)
	if err != nil {
		h.log.Error("failed to load curators", "course_id", *u.CourseID, "error", err)
		h.SendMessage(msg.Chat.ID, "❌ Ошибка загрузки списка кураторов.", nil)
		return
	}

	var rows [][]tgbotapi.KeyboardButton
	for _, c := range curators {
		btnText := fmt.Sprintf("👤 %s", c.FirstName)
		rows = append(rows, tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton(btnText)))
	}

	if len(rows) == 0 {
		h.SendMessage(msg.Chat.ID, "😔 На этом курсе пока нет доступных кураторов. Напиши администратору.", nil)
		return
	}

	kb := tgbotapi.NewReplyKeyboard(rows...)

	h.state.SetState(msg.From.ID, StateWaitingForCuratorSelection)
	h.SendMessage(msg.Chat.ID, "👤 <b>Наставники курса:</b>\nВыбери своего куратора из списка ниже:", kb)
}
