package telegram

import (
	"fmt"
	"kitchenBot/domain/location"
)

// Formatter форматирует данные домена для отправки в Telegram
type Formatter struct{}

// NewFormatter создает новый форматтер
func NewFormatter() *Formatter {
	return &Formatter{}
}

// FormatMainMenu форматирует главное меню
func (f *Formatter) FormatMainMenu() (string, *InlineKeyboardMarkup) {
	text := "🏋️ Выберите действие:"
	keyboard := NewInlineKeyboardMarkup(
		NewInlineKeyboardRow(
			NewInlineKeyboardButtonData("📍 Локации", "locations"),
			//NewInlineKeyboardButtonData("📅 Расписание", "schedule"),
		),
		NewInlineKeyboardRow(
			//NewInlineKeyboardButtonData("👤 Профиль", "profile"),
			//NewInlineKeyboardButtonData("ℹ️ Помощь", "help"),
			NewInlineKeyboardButtonData("👨‍ Администратор", "admin"),
		),
	)
	return text, keyboard
}

// FormatLocationsList форматирует список локаций
func (f *Formatter) FormatLocationsList(locations []*location.Location) (string, *InlineKeyboardMarkup) {
	if len(locations) == 0 {
		return "Нет доступных локаций", nil
	}

	// Создаем отдельную строку для каждой локации
	var rows [][]InlineKeyboardButton
	for _, loc := range locations {
		rows = append(rows, NewInlineKeyboardRow(
			NewInlineKeyboardButtonData(
				loc.Name,
				fmt.Sprintf("loc:%s", loc.ID.String()),
			),
		))
	}

	keyboard := NewInlineKeyboardMarkup(rows...)
	return "📍 Доступные локации:", keyboard
}

// FormatLocationDetails форматирует детали локации
func (f *Formatter) FormatLocationDetails(location *location.Location) (string, *InlineKeyboardMarkup) {
	text := fmt.Sprintf("📍 %s", location.Name)
	if location.Address != "" {
		text += fmt.Sprintf("\n🏠 Адрес: %s", location.Address)
	}
	text += fmt.Sprintf("\n\n🔑 ID: %s", location.ID.String())

	var rows [][]InlineKeyboardButton

	// Если есть URL карты, добавляем кнопку с картой
	if location.AddressMapUrl != "" {
		rows = append(rows, NewInlineKeyboardRow(
			NewInlineKeyboardButtonURL("🗺️ Открыть карту", location.AddressMapUrl),
		))
	}

	// Кнопка "Назад"
	rows = append(rows, NewInlineKeyboardRow(
		NewInlineKeyboardButtonData("🏠 Назад к локациям", "locations"),
	))

	keyboard := NewInlineKeyboardMarkup(rows...)
	return text, keyboard
}

// FormatAdminMenu форматирует меню администратора
func (f *Formatter) FormatAdminMenu() (string, *InlineKeyboardMarkup) {
	text := "🔧 Панель администратора\n\nВыберите действие:"
	keyboard := NewInlineKeyboardMarkup(
		NewInlineKeyboardRow(
			NewInlineKeyboardButtonData("➕ Создать локацию", "admin:create_location"),
		),
		NewInlineKeyboardRow(
			NewInlineKeyboardButtonData("➖ Удалить локацию", "admin:delete_location"),
		),
		NewInlineKeyboardRow(
			NewInlineKeyboardButtonData("📋 Список локаций", "admin:list_locations"),
		),
		NewInlineKeyboardRow(
			NewInlineKeyboardButtonData("➕ Создать тренировку", "admin:create_event"),
		),
		NewInlineKeyboardRow(
			NewInlineKeyboardButtonData("🏠 Главное меню", "back:main"),
		),
	)
	return text, keyboard
}

// FormatCreateLocationPrompt форматирует подсказку для создания локации
func (f *Formatter) FormatCreateLocationPrompt() string {
	return "📝 Создание новой локации\n\nОтправьте данные локации в формате:\nНазвание|Адрес|URL карты\n\nИли:\nНазвание|Адрес\n\nИли просто название.\n\nПример:\nСпортзал|ул. Ленина, д. 10|https://maps.google.com/..."
}

// FormatDeleteLocationPrompt форматирует подсказку для удаления локации
func (f *Formatter) FormatDeleteLocationPrompt() string {
	return "📝 Удаление локации\n\nОтправьте название локации одним сообщением.\nИли используйте команду:\n/admin_delete_location <название>"
}

// FormatLocationCreated форматирует сообщение об успешном создании локации
func (f *Formatter) FormatLocationCreated(location *location.Location) (string, *InlineKeyboardMarkup) {
	text := fmt.Sprintf("✅ Локация успешно создана!\n\n📍 Название: %s", location.Name)
	if location.Address != "" {
		text += fmt.Sprintf("\n🏠 Адрес: %s", location.Address)
	}
	if location.AddressMapUrl != "" {
		text += fmt.Sprintf("\n🗺️ Карта: %s", location.AddressMapUrl)
	}
	text += fmt.Sprintf("\n🔑 ID: %s", location.ID.String())

	var rows [][]InlineKeyboardButton

	// Если есть URL карты, добавляем кнопку с картой
	if location.AddressMapUrl != "" {
		rows = append(rows, NewInlineKeyboardRow(
			NewInlineKeyboardButtonURL("🗺️ Открыть карту", location.AddressMapUrl),
		))
	}

	// Кнопка "Назад"
	rows = append(rows, NewInlineKeyboardRow(
		NewInlineKeyboardButtonData("🔙 В меню администратора", "admin:menu"),
	))

	keyboard := NewInlineKeyboardMarkup(rows...)
	return text, keyboard
}

// FormatDeleteLocationList форматирует список локаций для удаления
func (f *Formatter) FormatDeleteLocationList(locations []*location.Location) (string, *InlineKeyboardMarkup) {
	if len(locations) == 0 {
		text := "📋 Нет локаций для удаления"
		keyboard := NewInlineKeyboardMarkup(
			NewInlineKeyboardRow(
				NewInlineKeyboardButtonData("🔙 Назад", "admin:menu"),
			),
		)
		return text, keyboard
	}

	text := "➖ Выберите локацию для удаления:"
	var rows [][]InlineKeyboardButton
	for _, loc := range locations {
		rows = append(rows, NewInlineKeyboardRow(
			NewInlineKeyboardButtonData(
				fmt.Sprintf("🗑️ %s", loc.Name),
				fmt.Sprintf("admin:delete:%s", loc.ID.String()),
			),
		))
	}

	rows = append(rows, NewInlineKeyboardRow(
		NewInlineKeyboardButtonData("🔙 Назад", "admin:menu"),
	))

	keyboard := NewInlineKeyboardMarkup(rows...)
	return text, keyboard
}

// FormatLocationDeleted форматирует сообщение об успешном удалении локации
func (f *Formatter) FormatLocationDeleted(locationName string) (string, *InlineKeyboardMarkup) {
	text := fmt.Sprintf("✅ Локация '%s' успешно удалена!", locationName)
	keyboard := NewInlineKeyboardMarkup(
		NewInlineKeyboardRow(
			NewInlineKeyboardButtonData("🔙 В меню администратора", "admin:menu"),
		),
	)
	return text, keyboard
}

// FormatLocationsListForAdmin форматирует список локаций для администратора
func (f *Formatter) FormatLocationsListForAdmin(locations []*location.Location) (string, *InlineKeyboardMarkup) {
	if len(locations) == 0 {
		text := "📋 Список локаций пуст"
		keyboard := NewInlineKeyboardMarkup(
			NewInlineKeyboardRow(
				NewInlineKeyboardButtonData("➕ Создать локацию", "admin:create_location"),
			),
			NewInlineKeyboardRow(
				NewInlineKeyboardButtonData("🔙 Назад", "admin:menu"),
			),
		)
		return text, keyboard
	}

	text, locationsMarkup := f.FormatLocationsList(locations)

	if locationsMarkup != nil {
		locationsMarkup.InlineKeyboard = append(locationsMarkup.InlineKeyboard, NewInlineKeyboardRow(NewInlineKeyboardButtonData("🔙 Назад", "admin:menu")))
	}

	return text, locationsMarkup

}

func (f *Formatter) FormatLocationsListForUsers(locations []*location.Location) (string, *InlineKeyboardMarkup) {
	if len(locations) == 0 {
		text := "📋 Список локаций пуст"
		keyboard := NewInlineKeyboardMarkup(
			NewInlineKeyboardRow(
				NewInlineKeyboardButtonData("➕ Создать локацию", "admin:create_location"),
			),
			NewInlineKeyboardRow(
				NewInlineKeyboardButtonData("🔙 Назад", "admin:menu"),
			),
		)
		return text, keyboard
	}

	text, locationsMarkup := f.FormatLocationsList(locations)

	if locationsMarkup != nil {
		locationsMarkup.InlineKeyboard = append(locationsMarkup.InlineKeyboard,
			NewInlineKeyboardRow(
				NewInlineKeyboardButtonData("🏠 Главное меню", "back:main"),
			))
	}

	return text, locationsMarkup

}
