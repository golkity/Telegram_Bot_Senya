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

	h.deleteUserMessage(msg)

	u := user.User{
		ID:        msg.From.ID,
		Username:  msg.From.UserName,
		FirstName: msg.From.FirstName,
		LastName:  msg.From.LastName,
		Role:      user.RoleStudent,
	}

	if err := h.userSvc.RegisterOrUpdate(ctx, u); err != nil {
		h.log.Error("registration failed", "error", err)
		h.SendCleanMessage(msg.Chat.ID, "⚠️ Произошла ошибка системы. Пожалуйста, попробуй позже.", nil)
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
		h.SendCleanMessage(chatID, msg, keyboards.AdminMenu)

	case user.RoleCurator:
		msg := fmt.Sprintf("👨‍🏫 <b>Привет, %s!</b>\n\nТвои ученики и отчеты ждут. Выбирай действие в меню 👇", u.FirstName)
		h.SendCleanMessage(chatID, msg, keyboards.CuratorMenu)

	case user.RoleDeveloper:
		h.handleDeveloperWelcome(ctx, chatID, u)

	case user.RoleStudent:
		h.handleStudentWelcome(ctx, chatID, u)

	default:
		h.SendCleanMessage(chatID, "❌ Твоя роль не определена. Пожалуйста, обратись к администратору.", nil)
	}
}

func (h *Handler) handleStudentWelcome(ctx context.Context, chatID int64, u *user.User) {
	if u.CourseID == nil || *u.CourseID == "" {
		h.HandleCourseSelection(ctx, &tgbotapi.Message{
			Chat: &tgbotapi.Chat{ID: chatID},
			From: &tgbotapi.User{ID: u.ID},
		})
		return
	}

	if u.CuratorID == nil || *u.CuratorID == 0 {
		h.HandleCuratorSelectionForStudent(ctx, &tgbotapi.Message{
			Chat: &tgbotapi.Chat{ID: chatID},
			From: &tgbotapi.User{ID: u.ID},
		})
		return
	}

	msg := fmt.Sprintf("👋 <b>С возвращением, %s!</b>\n\nЧто будем делать сегодня? Выбирай действие 👇", u.FirstName)
	h.SendCleanMessage(chatID, msg, keyboards.StudentMenu)
}

func (h *Handler) handleDeveloperWelcome(_ context.Context, chatID int64, u *user.User) {
	if u.CuratorID == nil || *u.CuratorID == 0 {
		msg := fmt.Sprintf("👨‍💻 <b>Привет, %s (Developer)!</b>\n\n⚠️ Чтобы тестировать сдачу ДЗ, тебе нужно <b>прикрепиться к куратору</b> через меню 👇", u.FirstName)
		h.SendCleanMessage(chatID, msg, keyboards.DeveloperMenu)
		return
	}

	msg := fmt.Sprintf("👨‍💻 <b>Привет, %s!</b>\n\nРежим разработчика активен. Доступные инструменты в меню 👇", u.FirstName)
	h.SendCleanMessage(chatID, msg, keyboards.DeveloperMenu)
}

func (h *Handler) HandleCourseSelection(ctx context.Context, msg *tgbotapi.Message) {
	h.deleteUserMessage(msg)

	courses, err := h.userSvc.GetAllCourses(ctx)
	if err != nil {
		h.SendCleanMessage(msg.Chat.ID, "❌ Ошибка загрузки курсов. Попробуй позже.", nil)
		return
	}

	rows := make([][]tgbotapi.KeyboardButton, 0, len(courses))
	for _, courseName := range courses {
		btnText := fmt.Sprintf("📚 %s", courseName)
		rows = append(rows, tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton(btnText)))
	}
	kb := tgbotapi.NewReplyKeyboard(rows...)

	h.state.SetState(msg.From.ID, StateWaitingForCourseSelection)

	text := "👋 <b>Добро пожаловать!</b> Давай настроим твой профиль.\n\n📚 <b>Доступные направления:</b>\nВыбери свой курс из меню ниже 👇"
	h.SendCleanMessage(msg.Chat.ID, text, kb)
}

func (h *Handler) HandleCuratorSelectionForStudent(ctx context.Context, msg *tgbotapi.Message) {
	h.deleteUserMessage(msg)

	u, err := h.userSvc.GetUserInfo(ctx, msg.From.ID)
	if err != nil || u == nil || u.CourseID == nil {
		h.SendCleanMessage(msg.Chat.ID, "⚠️ Сначала нужно выбрать курс.", nil)
		h.HandleCourseSelection(ctx, msg)
		return
	}

	curators, err := h.userSvc.GetCuratorsByCourse(ctx, *u.CourseID)
	if err != nil {
		h.log.Error("failed to load curators", "course_id", *u.CourseID, "error", err)
		h.SendCleanMessage(msg.Chat.ID, "❌ Ошибка загрузки списка кураторов.", nil)
		return
	}

	rows := make([][]tgbotapi.KeyboardButton, 0, len(curators))
	for _, c := range curators {
		btnText := fmt.Sprintf("👤 %s", c.FirstName)
		rows = append(rows, tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton(btnText)))
	}

	if len(rows) == 0 {
		h.SendCleanMessage(msg.Chat.ID, "😔 На этом курсе пока нет доступных кураторов. Напиши администратору.", nil)
		return
	}

	kb := tgbotapi.NewReplyKeyboard(rows...)

	h.state.SetState(msg.From.ID, StateWaitingForCuratorSelection)

	text := "✅ <b>Отлично! Курс сохранен.</b>\n\n👤 <b>Наставники курса:</b>\nТеперь последний шаг: выбери своего куратора из списка ниже 👇"
	h.SendCleanMessage(msg.Chat.ID, text, kb)
}
