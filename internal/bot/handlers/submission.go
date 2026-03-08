package handlers

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"telegram_bot/internal/bot/keyboards"
	"telegram_bot/internal/modules/submission"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var userLocks sync.Map

func getUserLock(userID int64) *sync.Mutex {
	m, _ := userLocks.LoadOrStore(userID, &sync.Mutex{})
	return m.(*sync.Mutex)
}

func (h *Handler) trackGarbageMsg(userID int64, msgID int) {
	rawGarbage := h.state.GetData(userID, "garbage_msgs")
	var garbageIDs []int

	if g, ok := rawGarbage.([]int); ok {
		garbageIDs = g
	} else if rawArray, ok := rawGarbage.([]interface{}); ok {
		for _, item := range rawArray {
			if idFloat, ok := item.(float64); ok {
				garbageIDs = append(garbageIDs, int(idFloat))
			}
		}
	}

	garbageIDs = append(garbageIDs, msgID)
	h.state.SetData(userID, "garbage_msgs", garbageIDs)
}

func (h *Handler) sendAndTrack(userID int64, chatID int64, text string, kb interface{}) {
	msgConfig := tgbotapi.NewMessage(chatID, text)
	msgConfig.ParseMode = "HTML"
	if kb != nil {
		msgConfig.ReplyMarkup = kb
	}
	sentMsg, err := h.bot.Send(msgConfig)
	if err == nil {
		h.trackGarbageMsg(userID, sentMsg.MessageID)
	}
}

func (h *Handler) DeleteMessages(chatID int64, messageIDs []int) {
	go func() {
		for _, msgID := range messageIDs {
			delMsg := tgbotapi.NewDeleteMessage(chatID, msgID)
			_, _ = h.bot.Request(delMsg)
			time.Sleep(50 * time.Millisecond) // Защита от лимитов Telegram (Too Many Requests)
		}
	}()
}

func (h *Handler) HandleSubmissionStart(ctx context.Context, msg *tgbotapi.Message, subType submission.Type) {
	h.deleteUserMessage(msg)
	userID := msg.From.ID

	u, err := h.userSvc.GetUserInfo(ctx, userID)
	if err != nil || u == nil {
		h.SendMessage(msg.Chat.ID, "❌ Ошибка получения данных пользователя.", nil)
		return
	}

	if u.CuratorID == nil || *u.CuratorID == 0 {
		h.SendMessage(msg.Chat.ID,
			"❌ Вы не прикреплены к куратору.\nСначала выберите куратора в главном меню, чтобы иметь возможность сдавать работы.",
			nil)
		return
	}

	h.state.SetState(userID, StateWaitingForTask)
	h.state.SetData(userID, "type", string(subType))
	h.state.SetData(userID, "files", []submission.FileDTO{})
	h.state.SetData(userID, "garbage_msgs", []int{})
	h.state.SetData(userID, "task", "")
	h.state.SetData(userID, "curator_id", *u.CuratorID)

	text := "ДЗ"
	if subType == submission.TypeNotes {
		text = "конспект"
	}

	h.sendAndTrack(userID, msg.Chat.ID, fmt.Sprintf("📚 Сдача: %s\n📋 Выберите номер задания:", text), keyboards.GetTaskSelectionKeyboard())
}

func (h *Handler) HandleTaskSelection(ctx context.Context, msg *tgbotapi.Message) {
	h.deleteUserMessage(msg)
	text := msg.Text
	userID := msg.From.ID

	if text == "Созвоны с кураторами" {
		h.state.SetData(userID, "task", "Созвоны")
		h.state.SetState(userID, StateWaitingForFiles)
		h.sendAndTrack(userID, msg.Chat.ID, "✅ Выбрано: Созвоны\n📤 Отправляйте файлы (скриншоты).", keyboards.SubmissionProcessMenu)
		return
	}

	if text == "Практика" {
		h.state.SetState(userID, StateWaitingForSubtask)
		h.state.SetData(userID, "task_category", "Practice")

		categories := []string{"Python", "Теория", "Сети", "Excel", "LibreOffice"}
		rows := make([][]tgbotapi.KeyboardButton, 0, len(categories)+1)
		for _, cat := range categories {
			rows = append(rows, tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton(cat)))
		}
		rows = append(rows, tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton("↩️ Назад")))

		h.sendAndTrack(userID, msg.Chat.ID, "📋 Выберите категорию практики:", tgbotapi.NewReplyKeyboard(rows...))
		return
	}

	if strings.HasPrefix(text, "Задание ") {
		taskNum := strings.TrimPrefix(text, "Задание ")
		subtasks, err := h.submissionSvc.GetSubtasks(ctx, taskNum)

		if err == nil && len(subtasks) > 0 {
			h.state.SetState(userID, StateWaitingForSubtask)
			h.state.SetData(userID, "task_parent", taskNum)

			rows := make([][]tgbotapi.KeyboardButton, 0, len(subtasks)+1)
			for _, sub := range subtasks {
				rows = append(rows, tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton(sub.Name)))
			}
			rows = append(rows, tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton("↩️ Назад")))

			h.sendAndTrack(userID, msg.Chat.ID, fmt.Sprintf("📋 Задание %s имеет подтемы. Выберите:", taskNum), tgbotapi.NewReplyKeyboard(rows...))
			return
		}

		h.state.SetData(userID, "task", taskNum)
		h.state.SetState(userID, StateWaitingForFiles)
		h.sendAndTrack(userID, msg.Chat.ID, fmt.Sprintf("✅ Выбрано: Задание %s\n📤 Отправляйте файлы (можно несколько).", taskNum), keyboards.SubmissionProcessMenu)
		return
	}

	h.SendMessage(msg.Chat.ID, "❌ Пожалуйста, выберите задание, используя кнопки.", nil)
}

func (h *Handler) HandleSubtaskSelection(_ context.Context, msg *tgbotapi.Message) {
	h.deleteUserMessage(msg)
	subtaskName := msg.Text
	userID := msg.From.ID

	h.state.SetData(userID, "task", subtaskName)
	h.state.SetState(userID, StateWaitingForFiles)
	h.sendAndTrack(userID, msg.Chat.ID, fmt.Sprintf("✅ Выбрано: %s\n📤 Отправляйте файлы.", subtaskName), keyboards.SubmissionProcessMenu)
}

func (h *Handler) HandleFileUpload(_ context.Context, msg *tgbotapi.Message) {
	var fileID, name string
	var size int64

	if msg.Document != nil {
		fileID = msg.Document.FileID
		name = msg.Document.FileName
		size = int64(msg.Document.FileSize)
	} else if len(msg.Photo) > 0 {
		p := msg.Photo[len(msg.Photo)-1]
		fileID = p.FileID
		name = fmt.Sprintf("photo_%d.jpg", time.Now().UnixNano())
		size = int64(p.FileSize)
	} else {
		h.deleteUserMessage(msg)
		h.SendMessage(msg.Chat.ID, "❌ Я понимаю только файлы и фото. Текст отправьте позже в комментарии.", nil)
		return
	}

	userID := msg.From.ID

	mu := getUserLock(userID)
	mu.Lock()

	rawFiles := h.state.GetData(userID, "files")
	var files []submission.FileDTO

	if f, ok := rawFiles.([]submission.FileDTO); ok {
		files = f
	} else if rawArray, ok := rawFiles.([]interface{}); ok {
		for _, item := range rawArray {
			if m, ok := item.(map[string]interface{}); ok {
				var file submission.FileDTO
				if fid, ok := m["FileID"].(string); ok {
					file.FileID = fid
				}
				if fname, ok := m["FileName"].(string); ok {
					file.FileName = fname
				}
				if sizeFloat, ok := m["Size"].(float64); ok {
					file.Size = int64(sizeFloat)
				}
				files = append(files, file)
			}
		}
	}

	files = append(files, submission.FileDTO{
		FileID:   fileID,
		FileName: name,
		Size:     size,
	})

	h.state.SetData(userID, "files", files)
	totalFiles := len(files)
	mu.Unlock()

	if msg.MediaGroupID == "" {
		h.sendAndTrack(userID, msg.Chat.ID, fmt.Sprintf("📥 Файл принят! (Всего: %d)\nОтправьте еще или нажмите 'Готово'.", totalFiles), nil)
	} else if totalFiles == 1 {
		h.sendAndTrack(userID, msg.Chat.ID, "📥 Загружаю альбом файлов...\nНажмите 'Готово', когда все файлы прогрузятся.", nil)
	} else {
		h.log.Info("Добавлено фото в альбом", "user_id", userID, "media_group", msg.MediaGroupID, "total", totalFiles)
	}
}

func (h *Handler) HandleSubmissionDone(ctx context.Context, msg *tgbotapi.Message) {
	h.deleteUserMessage(msg)
	userID := msg.From.ID
	rawFiles := h.state.GetData(userID, "files")

	if rawFiles == nil {
		h.SendMessage(msg.Chat.ID, "❌ Вы не прикрепили ни одного файла. Сдача невозможна.", keyboards.SubmissionProcessMenu)
		return
	}

	var files []submission.FileDTO

	if f, ok := rawFiles.([]submission.FileDTO); ok {
		files = f
	} else if rawArray, ok := rawFiles.([]interface{}); ok {
		for _, item := range rawArray {
			if m, ok := item.(map[string]interface{}); ok {
				var file submission.FileDTO

				if fileID, ok := m["FileID"].(string); ok {
					file.FileID = fileID
				}
				if fileName, ok := m["FileName"].(string); ok {
					file.FileName = fileName
				}
				if sizeFloat, ok := m["Size"].(float64); ok {
					file.Size = int64(sizeFloat)
				}

				files = append(files, file)
			}
		}
	}

	if len(files) == 0 {
		h.log.Error("failed to parse files from state", "rawFiles_type", fmt.Sprintf("%T", rawFiles), "data", rawFiles)
		h.SendMessage(msg.Chat.ID, "❌ Ошибка чтения файлов из памяти. Пожалуйста, начните сдачу заново.", keyboards.SubmissionProcessMenu)
		return
	}

	h.state.SetData(userID, "files", files)
	h.state.SetState(userID, StateWaitingForComment)

	h.sendAndTrack(userID, msg.Chat.ID, "📝 Напишите комментарий к работе или нажмите 'Пропустить'", keyboards.CommentMenu)
}

func (h *Handler) HandleSubmissionFinalize(ctx context.Context, msg *tgbotapi.Message) {
	h.deleteUserMessage(msg)
	userID := msg.From.ID
	comment := msg.Text

	if comment == "Пропустить" {
		comment = ""
	}

	rawGarbage := h.state.GetData(userID, "garbage_msgs")
	var garbageIDs []int
	if g, ok := rawGarbage.([]int); ok {
		garbageIDs = g
	} else if rawArray, ok := rawGarbage.([]interface{}); ok {
		for _, item := range rawArray {
			if idFloat, ok := item.(float64); ok {
				garbageIDs = append(garbageIDs, int(idFloat))
			}
		}
	}
	h.DeleteMessages(msg.Chat.ID, garbageIDs)

	waitMsgConfig := tgbotapi.NewMessage(msg.Chat.ID, "⏳ Обрабатываю и сохраняю файлы...")
	waitMsg, err := h.bot.Send(waitMsgConfig)
	if err != nil {
		h.log.Error("failed to send wait message", "error", err)
	}

	rawType := h.state.GetData(userID, "type")
	sType := "homework"
	if t, ok := rawType.(string); ok {
		sType = t
	}

	rawTask := h.state.GetData(userID, "task")
	task := ""
	if t, ok := rawTask.(string); ok {
		task = t
	}

	rawCurator := h.state.GetData(userID, "curator_id")
	var curatorID int64 = 0
	switch v := rawCurator.(type) {
	case int64:
		curatorID = v
	case float64:
		curatorID = int64(v)
	}

	var files []submission.FileDTO
	rawFiles := h.state.GetData(userID, "files")
	if f, ok := rawFiles.([]submission.FileDTO); ok {
		files = f
	} else if rawArray, ok := rawFiles.([]interface{}); ok {
		for _, item := range rawArray {
			if m, ok := item.(map[string]interface{}); ok {
				var file submission.FileDTO
				if fid, ok := m["FileID"].(string); ok {
					file.FileID = fid
				}
				if fname, ok := m["FileName"].(string); ok {
					file.FileName = fname
				}
				if sizeFloat, ok := m["Size"].(float64); ok {
					file.Size = int64(sizeFloat)
				}
				files = append(files, file)
			}
		}
	}

	u, _ := h.userSvc.GetUserInfo(ctx, userID)
	studentName := "Unknown_Student"
	courseName := "Unknown_Course"
	if u != nil {
		studentName = strings.TrimSpace(u.FirstName + " " + u.LastName)
		if u.CourseID != nil {
			courseName = *u.CourseID
		}
	}

	curatorName := "Unknown_Curator"
	if curatorID != 0 {
		c, _ := h.userSvc.GetUserInfo(ctx, curatorID)
		if c != nil {
			curatorName = strings.TrimSpace(c.FirstName + " " + c.LastName)
		}
	}

	input := submission.InputDTO{
		UserID:      userID,
		CuratorID:   curatorID,
		CourseName:  courseName,
		CuratorName: curatorName,
		StudentName: studentName,
		Type:        submission.Type(sType),
		TaskNumber:  task,
		Comment:     comment,
		Files:       files,
	}

	if err := h.submissionSvc.ProcessSubmission(ctx, input); err != nil {
		h.log.Error("submission save failed", "err", err)
		h.SendMessage(msg.Chat.ID, "❌ Ошибка сохранения. Пожалуйста, попробуй позже.", keyboards.StudentMenu)
	} else {
		if curatorID != 0 && curatorID != userID {
			notification := fmt.Sprintf("🔔 <b>Новая сдача!</b>\n👤 %s\n📚 %s: %s\n📎 Файлов: %d",
				studentName, sType, task, len(files))

			go func(cID int64, text string) {
				m := tgbotapi.NewMessage(cID, text)
				m.ParseMode = "HTML"
				if _, err := h.bot.Send(m); err != nil {
					h.log.Error("failed to notify curator (async)", "curator_id", cID, "error", err)
				}
			}(curatorID, notification)
		}

		h.SendMessage(msg.Chat.ID, "✅ Работа успешно сдана!", keyboards.StudentMenu)
	}

	if waitMsg.MessageID != 0 {
		go func() {
			_, _ = h.bot.Request(tgbotapi.NewDeleteMessage(msg.Chat.ID, waitMsg.MessageID))
		}()
	}

	h.state.ClearState(userID)
	h.state.ClearData(userID)
}
