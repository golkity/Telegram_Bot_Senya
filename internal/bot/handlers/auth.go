package handlers

import (
	"context"
	"fmt"
	"telegram_bot/internal/bot/keyboards"
	"telegram_bot/internal/modules/user"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (h *Handler) HandleStart(ctx context.Context, msg *tgbotapi.Message) {
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
		h.SendMessage(msg.Chat.ID, "⚠️ Ошибка системы. Попробуйте позже.", nil)
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
		h.SendMessage(chatID, fmt.Sprintf("👋 Привет, Админ %s! Системы в норме.", u.FirstName), keyboards.AdminMenu)

	case user.RoleCurator:
		h.SendMessage(chatID, fmt.Sprintf("👋 Привет, Куратор %s! Готов к работе.", u.FirstName), keyboards.CuratorMenu)

	case user.RoleDeveloper:
		h.handleDeveloperWelcome(ctx, chatID, u)

	case user.RoleStudent:
		h.handleStudentWelcome(ctx, chatID, u)

	default:
		h.SendMessage(chatID, "Ваша роль не определена. Обратитесь к администратору.", nil)
	}
}

func (h *Handler) handleStudentWelcome(ctx context.Context, chatID int64, u *user.User) {
	if u.CourseID == nil || *u.CourseID == "" {
		h.SendMessage(chatID,
			fmt.Sprintf("👋 Привет, %s!\n\nДля начала выбери курс, на котором учишься:", u.FirstName),
			nil)

		h.HandleCourseSelection(ctx, &tgbotapi.Message{
			Chat: &tgbotapi.Chat{ID: chatID},
			From: &tgbotapi.User{ID: u.ID},
		})
		return
	}

	if u.CuratorID == nil || *u.CuratorID == 0 {
		h.SendMessage(chatID,
			"✅ Курс выбран.\n\nТеперь выбери своего куратора:",
			nil)

		h.HandleCuratorSelectionForStudent(ctx, &tgbotapi.Message{
			Chat: &tgbotapi.Chat{ID: chatID},
			From: &tgbotapi.User{ID: u.ID},
		})
		return
	}

	h.SendMessage(chatID, fmt.Sprintf("👋 С возвращением, %s! Выбери действие:", u.FirstName), keyboards.StudentMenu)
}

func (h *Handler) handleDeveloperWelcome(_ context.Context, chatID int64, u *user.User) {
	if u.CuratorID == nil || *u.CuratorID == 0 {
		h.SendMessage(chatID,
			fmt.Sprintf("👋 Привет, Разработчик %s!\n\nДля работы вам нужно прикрепиться к куратору.", u.FirstName),
			keyboards.DeveloperMenu)
		return
	}

	h.SendMessage(chatID, fmt.Sprintf("👋 Привет, Разработчик %s!", u.FirstName), keyboards.DeveloperMenu)
}

func (h *Handler) HandleCourseSelection(ctx context.Context, msg *tgbotapi.Message) {
	if msg.MessageID != 0 {
		go func() {
			h.bot.Request(tgbotapi.NewDeleteMessage(msg.Chat.ID, msg.MessageID))
		}()
	}

	courses, err := h.userSvc.GetAllCourses(ctx)
	if err != nil {
		h.SendMessage(msg.Chat.ID, "Ошибка загрузки курсов.", nil)
		return
	}

	var rows [][]tgbotapi.KeyboardButton
	for _, courseName := range courses {
		btnText := fmt.Sprintf("📚 %s", courseName)
		rows = append(rows, tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton(btnText)))
	}
	kb := tgbotapi.NewReplyKeyboard(rows...)

	h.state.SetState(msg.From.ID, StateWaitingForCourseSelection)
	h.SendMessage(msg.Chat.ID, "📚 Список доступных курсов:", kb)
}

func (h *Handler) HandleCuratorSelectionForStudent(ctx context.Context, msg *tgbotapi.Message) {
	if msg.MessageID != 0 {
		go func() {
			h.bot.Request(tgbotapi.NewDeleteMessage(msg.Chat.ID, msg.MessageID))
		}()
	}

	u, err := h.userSvc.GetUserInfo(ctx, msg.From.ID)
	if err != nil || u == nil || u.CourseID == nil {
		h.SendMessage(msg.Chat.ID, "Сначала выберите курс.", nil)
		h.HandleCourseSelection(ctx, msg)
		return
	}

	curators, err := h.userSvc.GetCuratorsByCourse(ctx, *u.CourseID)
	if err != nil {
		h.log.Error("failed to load curators", "course_id", *u.CourseID, "error", err)

		h.SendMessage(msg.Chat.ID, "Ошибка загрузки кураторов.", nil)
		return
	}

	var rows [][]tgbotapi.KeyboardButton
	for _, c := range curators {
		btnText := fmt.Sprintf("👤 %s", c.FirstName)
		rows = append(rows, tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton(btnText)))
	}

	if len(rows) == 0 {
		h.SendMessage(msg.Chat.ID, "На этом курсе пока нет свободных кураторов.", nil)
		return
	}

	kb := tgbotapi.NewReplyKeyboard(rows...)

	h.state.SetState(msg.From.ID, StateWaitingForCuratorSelection)
	h.SendMessage(msg.Chat.ID, "👤 Выберите куратора из списка:", kb)
}
