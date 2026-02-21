package keyboards

import (
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var StudentMenu = tgbotapi.NewReplyKeyboard(
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("📚 Сдать ДЗ"),
		tgbotapi.NewKeyboardButton("📝 Сдать конспект"),
	),
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("📦 Получить архив"),
	),
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("📈 Ежедневная статистика"),
		tgbotapi.NewKeyboardButton("📝 Еженедельный отчет куратору"),
	),
)

var CuratorMenu = tgbotapi.NewReplyKeyboard(
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("📝 Отправить напоминание"),
		tgbotapi.NewKeyboardButton("📊 Ежедневный отчет"),
	),
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("📈 Еженедельный отчет"),
		tgbotapi.NewKeyboardButton("📋 Word отчет (еженедельный)"),
	),
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("📋 Отчет по сдаче (Excel)"),
		tgbotapi.NewKeyboardButton("👤 Мои ученики"),
	),
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("📊 Сводный отчет Excel"),
		tgbotapi.NewKeyboardButton("📁 Просмотреть работы учеников"),
	),
)

var AdminMenu = tgbotapi.NewReplyKeyboard(
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("📈 Ежедневный отчет"),
		tgbotapi.NewKeyboardButton("📊 Еженедельный отчет"),
	),
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("📋Подробный отчет Excel"),
		tgbotapi.NewKeyboardButton("📋 Word отчет (еженедельный)"),
	),
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("📋 Отчет по сдаче (Excel)"),
		tgbotapi.NewKeyboardButton("👥 Листы по ученикам"),
	),
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("👤 Управление пользователями"),
		tgbotapi.NewKeyboardButton("👥 Управление ролями"),
	),
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("⏰ Отправить напоминания"),
		tgbotapi.NewKeyboardButton("🎨 Кастомизация бота"),
	),
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("🔔 Уведомления администраторам"),
		tgbotapi.NewKeyboardButton("👨‍🏫 Перенос учеников"),
	),
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("👨‍🏫 Ученики кураторов"),
		tgbotapi.NewKeyboardButton("📊 Статистика кураторов"),
	),
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("📚 Ученики по курсам"),
	),
)

var DeveloperMenu = tgbotapi.NewReplyKeyboard(
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("📚 Сдать ДЗ"),
		tgbotapi.NewKeyboardButton("📝 Сдать конспект"),
	),
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("📦 Получить архив"),
		tgbotapi.NewKeyboardButton("👥 Выбрать куратора"),
	),
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("📈 Ежедневная статистика"),
		tgbotapi.NewKeyboardButton("📝 Еженедельный отчет куратору"),
	),
)

var SubmissionProcessMenu = tgbotapi.NewReplyKeyboard(
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("Готово"),
		tgbotapi.NewKeyboardButton("↩️ Назад"),
	),
)

var CommentMenu = tgbotapi.NewReplyKeyboard(
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("Пропустить"),
		tgbotapi.NewKeyboardButton("↩️ Назад"),
	),
)

var RoleManagementMenu = tgbotapi.NewReplyKeyboard(
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("🎓 Назначить роль"),
		tgbotapi.NewKeyboardButton("👤 Просмотреть все роли"),
	),
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("📚 Управление курсами"),
		tgbotapi.NewKeyboardButton("👨‍🏫 Назначить курс куратору"),
	),
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("👨‍💻 Назначить разработчика"),
		tgbotapi.NewKeyboardButton("↩️ Назад в админ-панель"),
	),
)

var CourseManagementMenu = tgbotapi.NewReplyKeyboard(
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("➕ Создать курс"),
		tgbotapi.NewKeyboardButton("📋 Просмотреть все курсы"),
	),
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("👨‍🏫 Назначить курс куратору"),
		tgbotapi.NewKeyboardButton("🗑️ Удалить курс"),
	),
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("↩️ Назад к ролям"),
	),
)

var CustomizationMenu = tgbotapi.NewReplyKeyboard(
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("✏️ Изменить имя бота"),
		tgbotapi.NewKeyboardButton("✏️ Изменить приветствие"),
	),
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("✏️ Изменить тексты кнопок"),
		tgbotapi.NewKeyboardButton("✏️ Изменить все сообщения"),
	),
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("⏰ Время отчета (МСК)"),
		tgbotapi.NewKeyboardButton("⏰ Время напоминаний (МСК)"),
	),
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("⏰ Время проверки (МСК)"),
		tgbotapi.NewKeyboardButton("⏰ Время отчетов кураторам (МСК)"),
	),
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("📅 Дни еженедельного отчета"),
		tgbotapi.NewKeyboardButton("👁️ Просмотреть конфигурацию"),
	),
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("🔄 Сбросить настройки"),
		tgbotapi.NewKeyboardButton("↩️ Назад в админ-панель"),
	),
)

var UserManagementMenu = tgbotapi.NewReplyKeyboard(
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("✏️ Изменить ник"),
		tgbotapi.NewKeyboardButton("🗑️ Удалить пользователя"),
	),
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("📊 Статистика пользователя"),
		tgbotapi.NewKeyboardButton("👨‍🏫 Сменить куратора"),
	),
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("🔄 Исправить статусы"),
		tgbotapi.NewKeyboardButton("↩️ Назад к списку"),
	),
)

var BulkTransferMenu = tgbotapi.NewReplyKeyboard(
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("👥 Перенести всех учеников куратора"),
	),
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("📚 Перенести по курсу"),
	),
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("↩️ Назад в меню"),
	),
)

var ConfirmDeleteMenu = tgbotapi.NewReplyKeyboard(
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("✅ Да, удалить"),
		tgbotapi.NewKeyboardButton("❌ Нет, отмена"),
	),
)

var ConfirmTransferMenu = tgbotapi.NewReplyKeyboard(
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("✅ Подтвердить перенос"),
		tgbotapi.NewKeyboardButton("❌ Отмена"),
	),
)

var BackButton = tgbotapi.NewReplyKeyboard(
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("↩️ Назад"),
	),
)

var CancelButton = tgbotapi.NewReplyKeyboard(
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("↩️ Отмена"),
	),
)

func GetTaskSelectionKeyboard() tgbotapi.ReplyKeyboardMarkup {
	var rows [][]tgbotapi.KeyboardButton
	var currentRow []tgbotapi.KeyboardButton

	for i := 1; i <= 27; i++ {
		button := tgbotapi.NewKeyboardButton(fmt.Sprintf("Задание %d", i))
		currentRow = append(currentRow, button)

		if len(currentRow) == 4 {
			rows = append(rows, currentRow)
			currentRow = []tgbotapi.KeyboardButton{}
		}
	}
	if len(currentRow) > 0 {
		rows = append(rows, currentRow)
	}

	rows = append(rows, []tgbotapi.KeyboardButton{
		tgbotapi.NewKeyboardButton("Практика"),
	})
	rows = append(rows, []tgbotapi.KeyboardButton{
		tgbotapi.NewKeyboardButton("Созвоны с кураторами"),
	})
	rows = append(rows, []tgbotapi.KeyboardButton{
		tgbotapi.NewKeyboardButton("↩️ Назад"),
	})

	return tgbotapi.NewReplyKeyboard(rows...)
}
