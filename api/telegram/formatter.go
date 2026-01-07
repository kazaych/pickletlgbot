package telegram

import (
	"fmt"
	"pickletlgbot/internal/domain/event"
	"pickletlgbot/internal/domain/location"
	"time"
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
		),
		NewInlineKeyboardRow(
			NewInlineKeyboardButtonData("📅 Записаться на событие", "events"),
		),
		NewInlineKeyboardRow(
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
				fmt.Sprintf("loc:%s", string(loc.ID)),
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
	text += fmt.Sprintf("\n\n🔑 ID: %s", string(location.ID))

	var rows [][]InlineKeyboardButton

	// Если есть URL карты, добавляем кнопку с картой
	if location.AddressMapURL != "" {
		rows = append(rows, NewInlineKeyboardRow(
			NewInlineKeyboardButtonURL("🗺️ Открыть карту", location.AddressMapURL),
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
			NewInlineKeyboardButtonData("📍 Локации", "admin:locations"),
		),
		NewInlineKeyboardRow(
			NewInlineKeyboardButtonData("📅 События", "admin:events"),
		),
		NewInlineKeyboardRow(
			NewInlineKeyboardButtonData("🏠 Главное меню", "back:main"),
		),
	)
	return text, keyboard
}

// FormatAdminLocationsMenu форматирует меню управления локациями
func (f *Formatter) FormatAdminLocationsMenu() (string, *InlineKeyboardMarkup) {
	text := "📍 Управление локациями\n\nВыберите действие:"
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
			NewInlineKeyboardButtonData("🔙 Назад", "admin:menu"),
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
	return "📝 Удаление локации\n\nИспользуйте кнопки ниже для выбора локации для удаления."
}

// FormatCreateEventPrompt форматирует подсказку для создания тренировки
func (f *Formatter) FormatCreateEventPrompt() string {
	return "📅 Создание новой тренировки\n\nСначала выберите локацию, затем укажите название тренировки."
}

// FormatLocationCreated форматирует сообщение об успешном создании локации
func (f *Formatter) FormatLocationCreated(location *location.Location) (string, *InlineKeyboardMarkup) {
	text := fmt.Sprintf("✅ Локация успешно создана!\n\n📍 Название: %s", location.Name)
	if location.Address != "" {
		text += fmt.Sprintf("\n🏠 Адрес: %s", location.Address)
	}
	if location.AddressMapURL != "" {
		text += fmt.Sprintf("\n🗺️ Карта: %s", location.AddressMapURL)
	}
	text += fmt.Sprintf("\n🔑 ID: %s", string(location.ID))

	var rows [][]InlineKeyboardButton

	// Если есть URL карты, добавляем кнопку с картой
	if location.AddressMapURL != "" {
		rows = append(rows, NewInlineKeyboardRow(
			NewInlineKeyboardButtonURL("🗺️ Открыть карту", location.AddressMapURL),
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
				fmt.Sprintf("admin:delete:%s", string(loc.ID)),
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

// FormatLocationsListForUsers форматирует список локаций для пользователей
func (f *Formatter) FormatLocationsListForUsers(locations []*location.Location) (string, *InlineKeyboardMarkup) {
	if len(locations) == 0 {
		text := "📋 Список локаций пуст"
		keyboard := NewInlineKeyboardMarkup(
			NewInlineKeyboardRow(
				NewInlineKeyboardButtonData("🏠 Главное меню", "back:main"),
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

// FormatAdminEventsMenu форматирует меню управления событиями
func (f *Formatter) FormatAdminEventsMenu() (string, *InlineKeyboardMarkup) {
	text := "📅 Управление событиями\n\nВыберите действие:"
	keyboard := NewInlineKeyboardMarkup(
		NewInlineKeyboardRow(
			NewInlineKeyboardButtonData("🏋️ Тренировки", "admin:events:training"),
		),
		NewInlineKeyboardRow(
			NewInlineKeyboardButtonData("🏆 Соревнования", "admin:events:competition"),
		),
		NewInlineKeyboardRow(
			NewInlineKeyboardButtonData("✅ Модерация регистраций", "admin:events:moderation"),
		),
		NewInlineKeyboardRow(
			NewInlineKeyboardButtonData("➕ Создать событие", "admin:create_event"),
		),
		NewInlineKeyboardRow(
			NewInlineKeyboardButtonData("🔙 Назад", "admin:menu"),
		),
	)
	return text, keyboard
}

// FormatEventsList форматирует список событий
func (f *Formatter) FormatEventsList(events []event.Event, eventType string, locationNames map[location.LocationID]string) (string, *InlineKeyboardMarkup) {
	if len(events) == 0 {
		typeName := "тренировок"
		if eventType == "competition" {
			typeName = "соревнований"
		}
		text := fmt.Sprintf("📋 Нет %s", typeName)
		keyboard := NewInlineKeyboardMarkup(
			NewInlineKeyboardRow(
				NewInlineKeyboardButtonData("🔙 Назад", "admin:events"),
			),
		)
		return text, keyboard
	}

	typeName := "Тренировки"
	if eventType == "competition" {
		typeName = "Соревнования"
	}
	text := fmt.Sprintf("📅 %s:", typeName)

	var rows [][]InlineKeyboardButton
	for _, evt := range events {
		timeStr := evt.Date.Format("15:04")
		locationName := locationNames[evt.LocationID]
		if locationName == "" {
			locationName = string(evt.LocationID)
		}
		freePlaces := evt.Remaining

		// Компактный формат: Название | Место | Время | 🆓N
		buttonText := fmt.Sprintf("%s | %s | %s | 🆓%d", evt.Name, locationName, timeStr, freePlaces)
		// Ограничиваем длину текста кнопки (Telegram рекомендует до 64 символов)
		if len(buttonText) > 60 {
			buttonText = buttonText[:57] + "..."
		}

		rows = append(rows, NewInlineKeyboardRow(
			NewInlineKeyboardButtonData(
				buttonText,
				fmt.Sprintf("admin:event:%s", string(evt.ID)),
			),
		))
	}

	rows = append(rows, NewInlineKeyboardRow(
		NewInlineKeyboardButtonData("🔙 Назад", "admin:events"),
	))

	keyboard := NewInlineKeyboardMarkup(rows...)
	return text, keyboard
}

// FormatEventDetails форматирует детали события
func (f *Formatter) FormatEventDetails(evt event.Event) (string, *InlineKeyboardMarkup) {
	text := fmt.Sprintf("📅 %s\n", evt.Name)
	text += fmt.Sprintf("🗓️ Дата: %s\n", evt.Date.Format("2006-01-02 15:04"))
	text += fmt.Sprintf("👥 Мест: %d/%d\n", evt.MaxPlayers-evt.Remaining, evt.MaxPlayers)
	text += fmt.Sprintf("📍 Локация ID: %s\n", string(evt.LocationID))
	if evt.Trainer != "" {
		text += fmt.Sprintf("👨‍🏫 Тренер: %s\n", evt.Trainer)
	}
	if evt.Description != "" {
		text += fmt.Sprintf("📝 %s\n", evt.Description)
	}

	var rows [][]InlineKeyboardButton
	rows = append(rows, NewInlineKeyboardRow(
		NewInlineKeyboardButtonData("✅ Модерация", fmt.Sprintf("admin:event:moderation:%s", string(evt.ID))),
	))
	rows = append(rows, NewInlineKeyboardRow(
		NewInlineKeyboardButtonData("🔙 Назад", "admin:events"),
	))

	keyboard := NewInlineKeyboardMarkup(rows...)
	return text, keyboard
}

// FormatPendingRegistrations форматирует список ожидающих регистраций
func (f *Formatter) FormatPendingRegistrations(eventName string, registrations []event.EventRegistration) (string, *InlineKeyboardMarkup) {
	if len(registrations) == 0 {
		text := fmt.Sprintf("✅ Нет заявок на модерацию для события:\n📅 %s", eventName)
		keyboard := NewInlineKeyboardMarkup(
			NewInlineKeyboardRow(
				NewInlineKeyboardButtonData("🔙 Назад", "admin:events"),
			),
		)
		return text, keyboard
	}

	text := fmt.Sprintf("🔔 Заявки на модерацию:\n📅 %s\n\n", eventName)

	var rows [][]InlineKeyboardButton
	for _, reg := range registrations {
		timeAgo := time.Since(reg.CreatedAt)
		var timeStr string
		if timeAgo < time.Minute {
			timeStr = "только что"
		} else if timeAgo < time.Hour {
			timeStr = fmt.Sprintf("%.0f мин назад", timeAgo.Minutes())
		} else {
			timeStr = fmt.Sprintf("%.0f ч назад", timeAgo.Hours())
		}

		text += fmt.Sprintf("👤 Пользователь ID: %d\n⏰ %s\n\n", reg.UserID, timeStr)

		rows = append(rows, NewInlineKeyboardRow(
			NewInlineKeyboardButtonData(
				fmt.Sprintf("👤 ID: %d (%s)", reg.UserID, timeStr),
				fmt.Sprintf("admin:reg:%d", reg.UserID),
			),
		))
	}

	rows = append(rows, NewInlineKeyboardRow(
		NewInlineKeyboardButtonData("🔙 Назад", "admin:events"),
	))

	keyboard := NewInlineKeyboardMarkup(rows...)
	return text, keyboard
}

// FormatRegistrationModeration форматирует модерацию конкретной регистрации
func (f *Formatter) FormatRegistrationModeration(eventName string, userID int64, eventID string) (string, *InlineKeyboardMarkup) {
	text := fmt.Sprintf("🔔 Модерация регистрации\n\n📅 Событие: %s\n👤 Пользователь ID: %d\n\nВыберите действие:", eventName, userID)
	keyboard := NewInlineKeyboardMarkup(
		NewInlineKeyboardRow(
			NewInlineKeyboardButtonData("✅ Подтвердить", fmt.Sprintf("admin:reg:approve:%s:%d", eventID, userID)),
			NewInlineKeyboardButtonData("❌ Отклонить", fmt.Sprintf("admin:reg:reject:%s:%d", eventID, userID)),
		),
		NewInlineKeyboardRow(
			NewInlineKeyboardButtonData("🔙 Назад", fmt.Sprintf("admin:event:moderation:%s", eventID)),
		),
	)
	return text, keyboard
}

// FormatEventsListForUsers форматирует список событий для пользователей
func (f *Formatter) FormatEventsListForUsers(events []event.Event, locationNames map[location.LocationID]string) (string, *InlineKeyboardMarkup) {
	if len(events) == 0 {
		text := "📋 Нет доступных событий"
		keyboard := NewInlineKeyboardMarkup(
			NewInlineKeyboardRow(
				NewInlineKeyboardButtonData("🏠 Главное меню", "back:main"),
			),
		)
		return text, keyboard
	}

	text := "📅 Доступные события:"
	var rows [][]InlineKeyboardButton
	for _, evt := range events {
		timeStr := evt.Date.Format("15:04")
		locationName := locationNames[evt.LocationID]
		if locationName == "" {
			locationName = string(evt.LocationID)
		}
		freePlaces := evt.Remaining

		// Компактный формат: Название | Место | Время | 🆓N
		buttonText := fmt.Sprintf("%s | %s | %s | 🆓%d", evt.Name, locationName, timeStr, freePlaces)
		// Ограничиваем длину текста кнопки (Telegram рекомендует до 64 символов)
		if len(buttonText) > 60 {
			buttonText = buttonText[:57] + "..."
		}

		rows = append(rows, NewInlineKeyboardRow(
			NewInlineKeyboardButtonData(
				buttonText,
				fmt.Sprintf("event:%s", string(evt.ID)),
			),
		))
	}

	rows = append(rows, NewInlineKeyboardRow(
		NewInlineKeyboardButtonData("🏠 Главное меню", "back:main"),
	))

	keyboard := NewInlineKeyboardMarkup(rows...)
	return text, keyboard
}

// FormatEventDetailsForUsers форматирует детали события для пользователей
func (f *Formatter) FormatEventDetailsForUsers(evt *event.Event, userID int64) (string, *InlineKeyboardMarkup) {
	typeEmoji := "🏋️"
	typeName := "Тренировка"
	if evt.Type == event.EventTypeCompetition {
		typeEmoji = "🏆"
		typeName = "Соревнование"
	}

	text := fmt.Sprintf("%s %s\n\n", typeEmoji, evt.Name)
	text += fmt.Sprintf("📅 Тип: %s\n", typeName)
	text += fmt.Sprintf("🗓️ Дата: %s\n", evt.Date.Format("02.01.2006 15:04"))
	text += fmt.Sprintf("👥 Мест: %d/%d\n", evt.MaxPlayers-evt.Remaining, evt.MaxPlayers)
	if evt.Trainer != "" {
		text += fmt.Sprintf("👨‍🏫 Тренер: %s\n", evt.Trainer)
	}
	if evt.Description != "" {
		text += fmt.Sprintf("📝 %s\n", evt.Description)
	}

	var rows [][]InlineKeyboardButton

	// Проверяем статус регистрации пользователя
	reg, isRegistered := evt.Registrations[userID]
	if isRegistered {
		switch reg.Status {
		case event.RegistrationStatusPending:
			text += "\n⏳ Ваша заявка ожидает подтверждения"
			rows = append(rows, NewInlineKeyboardRow(
				NewInlineKeyboardButtonData("❌ Отменить заявку", fmt.Sprintf("event:unregister:%s", string(evt.ID))),
			))
		case event.RegistrationStatusApproved:
			text += "\n✅ Вы зарегистрированы на это событие"
			rows = append(rows, NewInlineKeyboardRow(
				NewInlineKeyboardButtonData("❌ Отменить регистрацию", fmt.Sprintf("event:unregister:%s", string(evt.ID))),
			))
		case event.RegistrationStatusRejected:
			text += "\n❌ Ваша заявка была отклонена"
			if evt.Remaining > 0 {
				rows = append(rows, NewInlineKeyboardRow(
					NewInlineKeyboardButtonData("🔄 Подать заявку снова", fmt.Sprintf("event:register:%s", string(evt.ID))),
				))
			}
		}
	} else {
		// Пользователь не зарегистрирован
		if evt.Remaining > 0 {
			rows = append(rows, NewInlineKeyboardRow(
				NewInlineKeyboardButtonData("✅ Записаться на событие", fmt.Sprintf("event:register:%s", string(evt.ID))),
			))
		} else {
			text += "\n❌ Все места заняты"
		}
	}

	rows = append(rows, NewInlineKeyboardRow(
		NewInlineKeyboardButtonData("🔙 К списку событий", "events"),
	))

	keyboard := NewInlineKeyboardMarkup(rows...)
	return text, keyboard
}
