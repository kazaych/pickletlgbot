package telegram

import (
	"context"
	"fmt"
	"pickletlgbot/internal/domain/event"
	"pickletlgbot/internal/domain/location"
	"strconv"
	"strings"
	"time"
)

// handleAdminCommand обрабатывает команды администратора
func (h *Handlers) handleAdminCommand(msg *Message) {
	if !h.isAdmin(msg.From.ID) {
		if err := h.client.SendMessage(msg.ChatID, "❌ У вас нет прав администратора"); err != nil {
			h.logger.Error("failed to send admin access denied message", "chat_id", msg.ChatID, "error", err)
		}
		return
	}

	parts := strings.Fields(msg.Text)
	if len(parts) < 1 {
		return
	}

	command := parts[0]

	switch command {
	case "/admin":
		text, keyboard := h.formatter.FormatAdminMenu()
		if err := h.client.SendMessageWithKeyboard(msg.ChatID, text, keyboard); err != nil {
			h.logger.Error("failed to send admin menu", "chat_id", msg.ChatID, "error", err)
		}
	case "/admin_create_location":
		// Создаем состояние для создания локации
		h.creatingLocations[msg.ChatID] = &LocationCreationState{
			Step: "name",
		}
		text := "📍 Создание новой локации\n\nВведите название локации:"
		if err := h.client.SendMessage(msg.ChatID, text); err != nil {
			h.logger.Error("failed to send create location prompt", "chat_id", msg.ChatID, "error", err)
		}

	case "/admin_delete_location":
		text := h.formatter.FormatDeleteLocationPrompt()
		if err := h.client.SendMessage(msg.ChatID, text); err != nil {
			h.logger.Error("failed to send delete location prompt", "chat_id", msg.ChatID, "error", err)
		}

	default:
		// Команда /admin_create_location с параметрами больше не поддерживается
		// Используйте /admin_create_location без параметров для пошагового создания
	}
}

// handleAdminCallback обрабатывает callback-запросы администратора
func (h *Handlers) handleAdminCallback(ctx context.Context, cb *CallbackQuery) {
	switch cb.Data {
	case "admin:locations":
		text, keyboard := h.formatter.FormatAdminLocationsMenu()
		if err := h.client.EditMessageTextAndMarkup(cb.Message.ChatID, cb.Message.MessageID, text, keyboard); err != nil {
			h.logger.Error("failed to edit message with admin locations menu", "chat_id", cb.Message.ChatID, "error", err)
		}
	case "admin:create_location":
		h.handleAdminStartCreateLocation(ctx, cb)
	case "admin:delete_location":
		h.handleAdminDeleteLocation(ctx, cb)
	case "admin:list_locations":
		h.handleAdminListLocations(ctx, cb)
	case "admin:list_events":
		h.handleAdminListAllEvents(ctx, cb)
	case "admin:events":
		text, keyboard := h.formatter.FormatAdminEventsMenu()
		if err := h.client.EditMessageTextAndMarkup(cb.Message.ChatID, cb.Message.MessageID, text, keyboard); err != nil {
			h.logger.Error("failed to edit message with admin events menu", "chat_id", cb.Message.ChatID, "error", err)
		}
	case "admin:events:training":
		h.handleAdminListEvents(ctx, cb, event.EventTypeTraining)
	case "admin:events:competition":
		h.handleAdminListEvents(ctx, cb, event.EventTypeCompetition)
	case "admin:events:moderation":
		h.handleAdminModerationList(ctx, cb)
	case "admin:create_event":
		h.handleAdminCreateEvent(ctx, cb)
	case "admin:menu":
		text, keyboard := h.formatter.FormatAdminMenu()
		if err := h.client.EditMessageTextAndMarkup(cb.Message.ChatID, cb.Message.MessageID, text, keyboard); err != nil {
			h.logger.Error("failed to edit message with admin menu", "chat_id", cb.Message.ChatID, "error", err)
		}
	case "admin:set_channel":
		h.handleAdminSetChannelStart(cb)
	case "admin:delete_event":
		h.handleAdminDeleteEventList(ctx, cb)
	default:
		// Обработка динамических callback'ов для удаления (формат: admin:delete:{locationID})
		if strings.HasPrefix(cb.Data, "admin:delete:") {
			h.handleAdminConfirmDeleteLocation(ctx, cb)
		}
		// Обработка выбора локации для создания события (формат: admin:create_event:loc:{locationID})
		if strings.HasPrefix(cb.Data, "admin:create_event:loc:") {
			h.handleAdminSelectLocationForEvent(ctx, cb)
		}
		// Обработка выбора типа события (формат: admin:create_event:type:{locationID}:{type})
		if strings.HasPrefix(cb.Data, "admin:create_event:type:") {
			h.handleAdminSelectEventType(ctx, cb)
		}
		// Обработка модерации регистраций для события (формат: admin:event:moderation:{eventID})
		if strings.HasPrefix(cb.Data, "admin:event:moderation:") {
			h.handleAdminEventModeration(ctx, cb)
			return
		}
		// Обработка выбора события (формат: admin:event:{eventID})
		if strings.HasPrefix(cb.Data, "admin:event:") {
			h.handleAdminEventDetails(ctx, cb)
			return
		}
		// Обработка модерации регистрации (формат: admin:reg:{userID} или admin:reg:approve:{eventID}:{userID})
		if strings.HasPrefix(cb.Data, "admin:reg:") {
			h.handleAdminRegistrationModeration(ctx, cb)
		}
		// Обработка подтверждения удаления события (формат: admin:delete_event:confirm:{eventID})
		if strings.HasPrefix(cb.Data, "admin:delete_event:confirm:") {
			h.handleAdminConfirmDeleteEvent(ctx, cb)
		}
	}
}

// handleAdminSelectLocationForEvent обрабатывает выбор локации для создания тренировки
func (h *Handlers) handleAdminSelectLocationForEvent(ctx context.Context, cb *CallbackQuery) {
	// Парсим ID локации из callback data (формат: admin:create_event:loc:{locationID})
	parts := strings.Split(cb.Data, ":")
	if len(parts) != 4 {
		h.logger.Warn("invalid create event location callback data format", "callback_data", cb.Data, "chat_id", cb.Message.ChatID)
		if err := h.client.SendMessage(cb.Message.ChatID, "❌ Ошибка обработки запроса"); err != nil {
			h.logger.Error("failed to send error message", "chat_id", cb.Message.ChatID, "error", err)
		}
		return
	}

	locationIDStr := parts[3]
	locationID := location.LocationID(locationIDStr)

	// Получаем информацию о локации
	loc, err := h.locationService.Get(ctx, locationID)
	if err != nil {
		h.logger.Error("failed to get location for event", "location_id", locationIDStr, "chat_id", cb.Message.ChatID, "error", err)
		if sendErr := h.client.SendMessage(cb.Message.ChatID, "❌ Локация не найдена"); sendErr != nil {
			h.logger.Error("failed to send error message", "chat_id", cb.Message.ChatID, "error", sendErr)
		}
		return
	}

	// Сохраняем выбранную локацию для создания события
	h.creatingEvents[cb.Message.ChatID] = &EventCreationState{
		Step:       "type",
		LocationID: locationID,
	}

	text := fmt.Sprintf("📅 Создание события для локации: %s\n\nВыберите тип события:", loc.Name)
	keyboard := NewInlineKeyboardMarkup(
		NewInlineKeyboardRow(
			NewInlineKeyboardButtonData("🏋️ Тренировка", "admin:create_event:type:training"),
		),
		NewInlineKeyboardRow(
			NewInlineKeyboardButtonData("🏆 Соревнование", "admin:create_event:type:competition"),
		),
		NewInlineKeyboardRow(
			NewInlineKeyboardButtonData("🔙 Назад", "admin:menu"),
		),
	)
	if err := h.client.EditMessageTextAndMarkup(cb.Message.ChatID, cb.Message.MessageID, text, keyboard); err != nil {
		h.logger.Error("failed to send event type selection", "chat_id", cb.Message.ChatID, "error", err)
	}
}

// handleAdminSelectEventType обрабатывает выбор типа события
func (h *Handlers) handleAdminSelectEventType(ctx context.Context, cb *CallbackQuery) {
	// Парсим данные (формат: admin:create_event:type:{type})
	parts := strings.Split(cb.Data, ":")
	if len(parts) != 4 {
		h.logger.Warn("invalid create event type callback data format", "callback_data", cb.Data, "chat_id", cb.Message.ChatID)
		return
	}

	eventTypeStr := parts[3]

	var eventType event.EventType
	if eventTypeStr == "training" {
		eventType = event.EventTypeTraining
	} else if eventTypeStr == "competition" {
		eventType = event.EventTypeCompetition
	} else {
		h.logger.Warn("invalid event type", "type", eventTypeStr, "chat_id", cb.Message.ChatID)
		return
	}

	// Получаем состояние из памяти (locationID уже сохранен)
	state := h.creatingEvents[cb.Message.ChatID]
	if state == nil || state.LocationID == "" {
		h.logger.Error("event creation state not found", "chat_id", cb.Message.ChatID)
		if sendErr := h.client.SendMessage(cb.Message.ChatID, "❌ Ошибка получения состояния. Начните заново."); sendErr != nil {
			h.logger.Error("failed to send error message", "chat_id", cb.Message.ChatID, "error", sendErr)
		}
		return
	}

	// Обновляем состояние
	state.Step = "max_players"
	state.EventType = eventType

	typeName := "Тренировка"
	if eventType == event.EventTypeCompetition {
		typeName = "Соревнование"
	}

	text := fmt.Sprintf("📅 Тип события: %s\n\nВведите количество мест:", typeName)
	if err := h.client.EditMessageText(cb.Message.ChatID, cb.Message.MessageID, text); err != nil {
		h.logger.Error("failed to edit message for max players prompt", "chat_id", cb.Message.ChatID, "error", err)
	}
}

// handleAdminCreateEventStep обрабатывает шаги создания события
func (h *Handlers) handleAdminCreateEventStep(ctx context.Context, msg *Message, state *EventCreationState) {
	switch state.Step {
	case "max_players":
		h.handleAdminEnterMaxPlayers(ctx, msg, state)
	case "name":
		h.handleAdminEnterEventName(ctx, msg, state)
	case "date":
		h.handleAdminEnterEventDate(ctx, msg, state)
	case "trainer":
		h.handleAdminEnterTrainer(ctx, msg, state)
	case "payment_phone":
		h.handleAdminEnterPaymentPhone(ctx, msg, state)
	case "price":
		h.handleAdminEnterPrice(ctx, msg, state)
	default:
		// Неожиданный шаг, очищаем состояние
		delete(h.creatingEvents, msg.ChatID)
		if err := h.client.SendMessage(msg.ChatID, "❌ Ошибка процесса создания. Начните заново."); err != nil {
			h.logger.Error("failed to send error message", "chat_id", msg.ChatID, "error", err)
		}
	}
}

// handleAdminEnterMaxPlayers обрабатывает ввод количества мест
func (h *Handlers) handleAdminEnterMaxPlayers(ctx context.Context, msg *Message, state *EventCreationState) {
	maxPlayers, err := strconv.Atoi(strings.TrimSpace(msg.Text))
	if err != nil || maxPlayers <= 0 {
		if err := h.client.SendMessage(msg.ChatID, "❌ Введите корректное количество мест (положительное число):"); err != nil {
			h.logger.Error("failed to send error message", "chat_id", msg.ChatID, "error", err)
		}
		return
	}

	state.MaxPlayers = maxPlayers
	state.Step = "name"

	text := fmt.Sprintf("👥 Количество мест: %d\n\nВведите название события:", maxPlayers)
	if err := h.client.SendMessage(msg.ChatID, text); err != nil {
		h.logger.Error("failed to send event name prompt", "chat_id", msg.ChatID, "error", err)
	}
}

// handleAdminEnterEventName обрабатывает ввод названия события
func (h *Handlers) handleAdminEnterEventName(ctx context.Context, msg *Message, state *EventCreationState) {
	eventName := strings.TrimSpace(msg.Text)
	if eventName == "" {
		if err := h.client.SendMessage(msg.ChatID, "❌ Название события не может быть пустым. Введите название:"); err != nil {
			h.logger.Error("failed to send error message", "chat_id", msg.ChatID, "error", err)
		}
		return
	}

	state.EventName = eventName
	state.Step = "date"

	text := fmt.Sprintf("📝 Название: %s\n\nВведите дату и время начала события в формате:\n📅 ДД.ММ.ГГГГ ЧЧ:ММ\n\nПример: 15.01.2026 18:00", eventName)
	if err := h.client.SendMessage(msg.ChatID, text); err != nil {
		h.logger.Error("failed to send event date prompt", "chat_id", msg.ChatID, "error", err)
	}
}

// handleAdminEnterEventDate обрабатывает ввод даты и времени события
func (h *Handlers) handleAdminEnterEventDate(ctx context.Context, msg *Message, state *EventCreationState) {
	dateStr := strings.TrimSpace(msg.Text)

	// Парсим дату в формате "02.01.2006 15:04"
	eventDate, err := time.Parse("02.01.2006 15:04", dateStr)
	if err != nil {
		// Пробуем альтернативный формат "02.01.2006 15:4" (без ведущего нуля в минутах)
		eventDate, err = time.Parse("02.01.2006 15:4", dateStr)
		if err != nil {
			// Пробуем формат без времени
			eventDate, err = time.Parse("02.01.2006", dateStr)
			if err != nil {
				if sendErr := h.client.SendMessage(msg.ChatID, "❌ Неверный формат даты. Используйте формат:\n📅 ДД.ММ.ГГГГ ЧЧ:ММ\n\nПример: 15.01.2026 18:00"); sendErr != nil {
					h.logger.Error("failed to send error message", "chat_id", msg.ChatID, "error", sendErr)
				}
				return
			}
			// Если время не указано, устанавливаем на 18:00 по умолчанию
			eventDate = time.Date(eventDate.Year(), eventDate.Month(), eventDate.Day(), 18, 0, 0, 0, eventDate.Location())
		}
	}

	// Проверяем, что дата не в прошлом
	if eventDate.Before(time.Now()) {
		if sendErr := h.client.SendMessage(msg.ChatID, "❌ Дата события не может быть в прошлом. Введите корректную дату:"); sendErr != nil {
			h.logger.Error("failed to send error message", "chat_id", msg.ChatID, "error", sendErr)
		}
		return
	}

	state.EventDate = eventDate
	state.Step = "trainer"

	text := fmt.Sprintf("🗓️ Дата: %s\n\nВведите имя тренера:", eventDate.Format("02.01.2006 15:04"))
	if err := h.client.SendMessage(msg.ChatID, text); err != nil {
		h.logger.Error("failed to send trainer prompt", "chat_id", msg.ChatID, "error", err)
	}
}

// handleAdminEnterTrainer обрабатывает ввод тренера
func (h *Handlers) handleAdminEnterTrainer(ctx context.Context, msg *Message, state *EventCreationState) {
	trainer := strings.TrimSpace(msg.Text)
	if trainer == "" {
		if err := h.client.SendMessage(msg.ChatID, "❌ Имя тренера не может быть пустым. Введите имя тренера:"); err != nil {
			h.logger.Error("failed to send error message", "chat_id", msg.ChatID, "error", err)
		}
		return
	}

	state.Trainer = trainer
	state.Step = "payment_phone"

	text := fmt.Sprintf("👨‍🏫 Тренер: %s\n\nВведите номер телефона для оплаты (например, +79991234567):", trainer)
	if err := h.client.SendMessage(msg.ChatID, text); err != nil {
		h.logger.Error("failed to send payment phone prompt", "chat_id", msg.ChatID, "error", err)
	}
}

// handleAdminEnterPaymentPhone обрабатывает ввод телефона для оплаты
func (h *Handlers) handleAdminEnterPaymentPhone(ctx context.Context, msg *Message, state *EventCreationState) {
	paymentPhone := strings.TrimSpace(msg.Text)
	if paymentPhone == "" {
		if err := h.client.SendMessage(msg.ChatID, "❌ Номер телефона не может быть пустым. Введите номер телефона:"); err != nil {
			h.logger.Error("failed to send error message", "chat_id", msg.ChatID, "error", err)
		}
		return
	}

	state.PaymentPhone = paymentPhone
	state.Step = "price"

	text := fmt.Sprintf("📱 Телефон для оплаты: %s\n\nВведите стоимость тренировки (в рублях, только число):", paymentPhone)
	if err := h.client.SendMessage(msg.ChatID, text); err != nil {
		h.logger.Error("failed to send price prompt", "chat_id", msg.ChatID, "error", err)
	}
}

// handleAdminEnterPrice обрабатывает ввод цены и создает событие
func (h *Handlers) handleAdminEnterPrice(ctx context.Context, msg *Message, state *EventCreationState) {
	priceStr := strings.TrimSpace(msg.Text)
	price, err := strconv.Atoi(priceStr)
	if err != nil || price < 0 {
		if err := h.client.SendMessage(msg.ChatID, "❌ Введите корректную стоимость (положительное число в рублях):"); err != nil {
			h.logger.Error("failed to send error message", "chat_id", msg.ChatID, "error", err)
		}
		return
	}

	state.Price = price

	// Удаляем состояние перед созданием события
	delete(h.creatingEvents, msg.ChatID)

	// Создаем событие
	evt, err := h.eventService.Create(ctx, event.CreateEventInput{
		Name:         state.EventName,
		Type:         state.EventType,
		Date:         state.EventDate,
		MaxPlayers:   state.MaxPlayers,
		LocationID:   state.LocationID,
		Trainer:      state.Trainer,
		Description:  "",
		PaymentPhone: state.PaymentPhone,
		Price:        state.Price,
	})

	if err != nil {
		h.logger.Error("failed to create event", "event_name", state.EventName, "location_id", string(state.LocationID), "chat_id", msg.ChatID, "error", err)
		if sendErr := h.client.SendMessage(msg.ChatID, fmt.Sprintf("❌ Ошибка создания события: %v", err)); sendErr != nil {
			h.logger.Error("failed to send error message", "chat_id", msg.ChatID, "error", sendErr)
		}
		return
	}

	typeName := "Тренировка"
	if state.EventType == event.EventTypeCompetition {
		typeName = "Соревнование"
	}

	text := fmt.Sprintf("✅ %s успешно создано!\n\n📅 Название: %s\n🗓️ Дата: %s\n👥 Мест: %d\n👨‍🏫 Тренер: %s\n🔑 ID: %s",
		typeName, evt.Name, evt.Date.Format("02.01.2006 15:04"), evt.MaxPlayers, evt.Trainer, string(evt.ID))
	keyboard := NewInlineKeyboardMarkup(
		NewInlineKeyboardRow(
			NewInlineKeyboardButtonData("🔙 В меню администратора", "admin:menu"),
		),
	)
	if err := h.client.SendMessageWithKeyboard(msg.ChatID, text, keyboard); err != nil {
		h.logger.Error("failed to send event created message", "chat_id", msg.ChatID, "error", err)
	}

	// Публикуем анонс в канал (если настроен)
	h.publishEventToChannel(ctx, evt)
}

// handleAdminStartCreateLocation начинает процесс создания локации
func (h *Handlers) handleAdminStartCreateLocation(ctx context.Context, cb *CallbackQuery) {
	// Создаем состояние для создания локации
	h.creatingLocations[cb.Message.ChatID] = &LocationCreationState{
		Step: "name",
	}

	text := "📍 Создание новой локации\n\nВведите название локации:"
	if err := h.client.EditMessageText(cb.Message.ChatID, cb.Message.MessageID, text); err != nil {
		h.logger.Error("failed to edit message for location name prompt", "chat_id", cb.Message.ChatID, "error", err)
	}
}

// handleAdminCreateLocationStep обрабатывает шаги создания локации
func (h *Handlers) handleAdminCreateLocationStep(ctx context.Context, msg *Message, state *LocationCreationState) {
	switch state.Step {
	case "name":
		h.handleAdminEnterLocationName(ctx, msg, state)
	case "address":
		h.handleAdminEnterLocationAddress(ctx, msg, state)
	case "map_url":
		h.handleAdminEnterLocationMapURL(ctx, msg, state)
	default:
		// Неожиданный шаг, очищаем состояние
		h.clearCreatingLocationState(msg.ChatID)
		if err := h.client.SendMessage(msg.ChatID, "❌ Ошибка процесса создания. Начните заново."); err != nil {
			h.logger.Error("failed to send error message", "chat_id", msg.ChatID, "error", err)
		}
	}
}

// handleAdminEnterLocationName обрабатывает ввод названия локации
func (h *Handlers) handleAdminEnterLocationName(ctx context.Context, msg *Message, state *LocationCreationState) {
	name := strings.TrimSpace(msg.Text)
	if name == "" {
		if err := h.client.SendMessage(msg.ChatID, "❌ Название локации не может быть пустым. Введите название:"); err != nil {
			h.logger.Error("failed to send error message", "chat_id", msg.ChatID, "error", err)
		}
		return
	}

	state.Name = name
	state.Step = "address"

	text := fmt.Sprintf("📍 Название: %s\n\nВведите адрес локации:", name)
	if err := h.client.SendMessage(msg.ChatID, text); err != nil {
		h.logger.Error("failed to send location address prompt", "chat_id", msg.ChatID, "error", err)
	}
}

// handleAdminEnterLocationAddress обрабатывает ввод адреса локации
func (h *Handlers) handleAdminEnterLocationAddress(ctx context.Context, msg *Message, state *LocationCreationState) {
	address := strings.TrimSpace(msg.Text)
	if address == "" {
		if err := h.client.SendMessage(msg.ChatID, "❌ Адрес локации не может быть пустым. Введите адрес:"); err != nil {
			h.logger.Error("failed to send error message", "chat_id", msg.ChatID, "error", err)
		}
		return
	}

	state.Address = address
	state.Step = "map_url"

	text := fmt.Sprintf("📍 Название: %s\n📍 Адрес: %s\n\nВведите ссылку на карту (или отправьте \"-\" чтобы пропустить):", state.Name, address)
	if err := h.client.SendMessage(msg.ChatID, text); err != nil {
		h.logger.Error("failed to send location map URL prompt", "chat_id", msg.ChatID, "error", err)
	}
}

// handleAdminEnterLocationMapURL обрабатывает ввод ссылки на карту и завершает создание локации
func (h *Handlers) handleAdminEnterLocationMapURL(ctx context.Context, msg *Message, state *LocationCreationState) {
	addressUrl := strings.TrimSpace(msg.Text)
	if addressUrl == "-" || addressUrl == "" {
		addressUrl = ""
	}

	state.AddressMapURL = addressUrl

	// Создаем локацию
	loc, err := h.locationService.Create(ctx, location.CreateLocationInput{
		Name:          state.Name,
		Address:       state.Address,
		AddressMapURL: state.AddressMapURL,
		Description:   "", // Описание можно добавить позже
	})
	if err != nil {
		h.logger.Error("failed to create location", "location_name", state.Name, "chat_id", msg.ChatID, "error", err)
		if sendErr := h.client.SendMessage(msg.ChatID, fmt.Sprintf("❌ Ошибка создания локации: %v", err)); sendErr != nil {
			h.logger.Error("failed to send error message", "chat_id", msg.ChatID, "error", sendErr)
		}
		h.clearCreatingLocationState(msg.ChatID)
		return
	}

	// Очищаем состояние
	h.clearCreatingLocationState(msg.ChatID)

	text, keyboard := h.formatter.FormatLocationCreated(loc)
	if err := h.client.SendMessageWithKeyboard(msg.ChatID, text, keyboard); err != nil {
		h.logger.Error("failed to send location created message", "chat_id", msg.ChatID, "location_id", string(loc.ID), "error", err)
	}
}

// handleAdminListLocations обрабатывает запрос списка локаций администратором
func (h *Handlers) handleAdminListLocations(ctx context.Context, cb *CallbackQuery) {
	locations, err := h.locationService.List(ctx)
	if err != nil {
		h.logger.Error("failed to list locations for admin", "chat_id", cb.Message.ChatID, "error", err)
		if sendErr := h.client.SendMessage(cb.Message.ChatID, "❌ Ошибка получения списка локаций"); sendErr != nil {
			h.logger.Error("failed to send error message", "chat_id", cb.Message.ChatID, "error", sendErr)
		}
		return
	}

	// Конвертируем []Location в []*Location для форматтера
	locationPtrs := make([]*location.Location, len(locations))
	for i := range locations {
		locationPtrs[i] = &locations[i]
	}

	text, keyboard := h.formatter.FormatLocationsListForAdmin(locationPtrs)

	// Пытаемся отредактировать сообщение, если не получается - отправляем новое
	err = h.client.EditMessageTextAndMarkup(cb.Message.ChatID, cb.Message.MessageID, text, keyboard)
	if err != nil {
		h.logger.Warn("failed to edit message, sending new one", "chat_id", cb.Message.ChatID, "error", err)
		// Если редактирование не удалось, отправляем новое сообщение
		if sendErr := h.client.SendMessageWithKeyboard(cb.Message.ChatID, text, keyboard); sendErr != nil {
			h.logger.Error("failed to send new message with locations list", "chat_id", cb.Message.ChatID, "error", sendErr)
		}
	}
}

// handleAdminDeleteLocation обрабатывает запрос на удаление локации
func (h *Handlers) handleAdminDeleteLocation(ctx context.Context, cb *CallbackQuery) {
	locations, err := h.locationService.List(ctx)
	if err != nil {
		h.logger.Error("failed to list locations for deletion", "chat_id", cb.Message.ChatID, "error", err)
		if sendErr := h.client.SendMessage(cb.Message.ChatID, "❌ Ошибка получения списка локаций"); sendErr != nil {
			h.logger.Error("failed to send error message", "chat_id", cb.Message.ChatID, "error", sendErr)
		}
		return
	}

	// Конвертируем []Location в []*Location для форматтера
	locationPtrs := make([]*location.Location, len(locations))
	for i := range locations {
		locationPtrs[i] = &locations[i]
	}

	text, keyboard := h.formatter.FormatDeleteLocationList(locationPtrs)
	if err := h.client.EditMessageTextAndMarkup(cb.Message.ChatID, cb.Message.MessageID, text, keyboard); err != nil {
		h.logger.Error("failed to edit message with delete location list", "chat_id", cb.Message.ChatID, "error", err)
	}
}

// handleAdminConfirmDeleteLocation обрабатывает подтверждение удаления локации
func (h *Handlers) handleAdminConfirmDeleteLocation(ctx context.Context, cb *CallbackQuery) {
	// Парсим ID из callback data (формат: admin:delete:{locationID})
	parts := strings.Split(cb.Data, ":")
	if len(parts) != 3 {
		h.logger.Warn("invalid delete location callback data format", "callback_data", cb.Data, "chat_id", cb.Message.ChatID)
		if err := h.client.SendMessage(cb.Message.ChatID, "❌ Ошибка обработки запроса"); err != nil {
			h.logger.Error("failed to send error message", "chat_id", cb.Message.ChatID, "error", err)
		}
		return
	}

	locationIDStr := parts[2]
	locationID := location.LocationID(locationIDStr)

	// Получаем информацию о локации перед удалением
	loc, err := h.locationService.Get(ctx, locationID)
	if err != nil {
		h.logger.Error("failed to get location for deletion", "location_id", locationIDStr, "chat_id", cb.Message.ChatID, "error", err)
		if sendErr := h.client.SendMessage(cb.Message.ChatID, "❌ Локация не найдена"); sendErr != nil {
			h.logger.Error("failed to send error message", "chat_id", cb.Message.ChatID, "error", sendErr)
		}
		return
	}

	locationName := loc.Name

	// Удаляем локацию
	err = h.locationService.Delete(ctx, locationID)
	if err != nil {
		h.logger.Error("failed to delete location", "location_id", locationIDStr, "chat_id", cb.Message.ChatID, "error", err)
		if sendErr := h.client.SendMessage(cb.Message.ChatID, fmt.Sprintf("❌ Ошибка удаления локации: %v", err)); sendErr != nil {
			h.logger.Error("failed to send error message", "chat_id", cb.Message.ChatID, "error", sendErr)
		}
		return
	}

	text, keyboard := h.formatter.FormatLocationDeleted(locationName)
	if err := h.client.EditMessageTextAndMarkup(cb.Message.ChatID, cb.Message.MessageID, text, keyboard); err != nil {
		h.logger.Error("failed to edit message with location deleted confirmation", "chat_id", cb.Message.ChatID, "error", err)
	}
}

// handleAdminCreateEvent обрабатывает создание тренировки администратором
func (h *Handlers) handleAdminCreateEvent(ctx context.Context, cb *CallbackQuery) {
	// Показываем список локаций для выбора
	locations, err := h.locationService.List(ctx)
	if err != nil {
		h.logger.Error("failed to list locations for event creation", "chat_id", cb.Message.ChatID, "error", err)
		if sendErr := h.client.SendMessage(cb.Message.ChatID, "❌ Ошибка получения списка локаций"); sendErr != nil {
			h.logger.Error("failed to send error message", "chat_id", cb.Message.ChatID, "error", sendErr)
		}
		return
	}

	if len(locations) == 0 {
		if err := h.client.SendMessage(cb.Message.ChatID, "❌ Нет доступных локаций. Сначала создайте локацию."); err != nil {
			h.logger.Error("failed to send no locations message", "chat_id", cb.Message.ChatID, "error", err)
		}
		return
	}

	text := "📅 Выберите локацию для тренировки:"
	var rows [][]InlineKeyboardButton
	for _, loc := range locations {
		rows = append(rows, NewInlineKeyboardRow(
			NewInlineKeyboardButtonData(
				loc.Name,
				fmt.Sprintf("admin:create_event:loc:%s", string(loc.ID)),
			),
		))
	}
	rows = append(rows, NewInlineKeyboardRow(
		NewInlineKeyboardButtonData("🔙 Назад", "admin:menu"),
	))

	keyboard := NewInlineKeyboardMarkup(rows...)
	if err := h.client.EditMessageTextAndMarkup(cb.Message.ChatID, cb.Message.MessageID, text, keyboard); err != nil {
		h.logger.Error("failed to edit message with locations list for event", "chat_id", cb.Message.ChatID, "error", err)
	}
}

// handleAdminListEvents обрабатывает список событий по типу
func (h *Handlers) handleAdminListEvents(ctx context.Context, cb *CallbackQuery, eventType event.EventType) {
	allEvents, err := h.eventService.List(ctx)
	if err != nil {
		h.logger.Error("failed to list events", "chat_id", cb.Message.ChatID, "error", err)
		if sendErr := h.client.SendMessage(cb.Message.ChatID, "❌ Ошибка получения списка событий"); sendErr != nil {
			h.logger.Error("failed to send error message", "chat_id", cb.Message.ChatID, "error", sendErr)
		}
		return
	}

	// Фильтруем по типу
	var filteredEvents []event.Event
	locationIDs := make(map[location.LocationID]bool)
	for _, evt := range allEvents {
		if evt.Type == eventType {
			filteredEvents = append(filteredEvents, evt)
			locationIDs[evt.LocationID] = true
		}
	}

	// Загружаем названия локаций
	locationNames := make(map[location.LocationID]string)
	for locID := range locationIDs {
		loc, err := h.locationService.Get(ctx, locID)
		if err == nil && loc != nil {
			locationNames[locID] = loc.Name
		}
	}

	text, keyboard := h.formatter.FormatEventsList(filteredEvents, string(eventType), locationNames)
	if err := h.client.EditMessageTextAndMarkup(cb.Message.ChatID, cb.Message.MessageID, text, keyboard); err != nil {
		h.logger.Error("failed to edit message with events list", "chat_id", cb.Message.ChatID, "error", err)
	}
}

// handleAdminListAllEvents обрабатывает список всех событий
func (h *Handlers) handleAdminListAllEvents(ctx context.Context, cb *CallbackQuery) {
	allEvents, err := h.eventService.List(ctx)
	if err != nil {
		h.logger.Error("failed to list events", "chat_id", cb.Message.ChatID, "error", err)
		if sendErr := h.client.SendMessage(cb.Message.ChatID, "❌ Ошибка получения списка событий"); sendErr != nil {
			h.logger.Error("failed to send error message", "chat_id", cb.Message.ChatID, "error", sendErr)
		}
		return
	}

	// Собираем уникальные LocationID
	locationIDs := make(map[location.LocationID]bool)
	for _, evt := range allEvents {
		locationIDs[evt.LocationID] = true
	}

	// Загружаем названия локаций
	locationNames := make(map[location.LocationID]string)
	for locID := range locationIDs {
		loc, err := h.locationService.Get(ctx, locID)
		if err == nil && loc != nil {
			locationNames[locID] = loc.Name
		}
	}

	text, keyboard := h.formatter.FormatEventsList(allEvents, "all", locationNames)
	if err := h.client.EditMessageTextAndMarkup(cb.Message.ChatID, cb.Message.MessageID, text, keyboard); err != nil {
		h.logger.Error("failed to edit message with events list", "chat_id", cb.Message.ChatID, "error", err)
	}
}

// handleAdminEventDetails обрабатывает детали события
func (h *Handlers) handleAdminEventDetails(ctx context.Context, cb *CallbackQuery) {
	// Парсим eventID из callback data (формат: admin:event:{eventID})
	parts := strings.Split(cb.Data, ":")
	if len(parts) != 3 {
		h.logger.Warn("invalid event callback data format", "callback_data", cb.Data, "chat_id", cb.Message.ChatID)
		return
	}

	eventID := event.EventID(parts[2])
	evt, err := h.eventService.Get(ctx, eventID)
	if err != nil {
		h.logger.Error("failed to get event", "event_id", string(eventID), "chat_id", cb.Message.ChatID, "error", err)
		if sendErr := h.client.SendMessage(cb.Message.ChatID, "❌ Событие не найдено"); sendErr != nil {
			h.logger.Error("failed to send error message", "chat_id", cb.Message.ChatID, "error", sendErr)
		}
		return
	}

	text, keyboard := h.formatter.FormatEventDetails(*evt)
	if err := h.client.EditMessageTextAndMarkup(cb.Message.ChatID, cb.Message.MessageID, text, keyboard); err != nil {
		h.logger.Error("failed to edit message with event details", "chat_id", cb.Message.ChatID, "error", err)
	}
}

// handleAdminEventModeration обрабатывает показ pending регистраций для события
func (h *Handlers) handleAdminEventModeration(ctx context.Context, cb *CallbackQuery) {
	// Парсим ID события из callback data (формат: admin:event:moderation:{eventID})
	parts := strings.Split(cb.Data, ":")
	if len(parts) != 4 {
		h.logger.Warn("invalid event moderation callback data format", "callback_data", cb.Data, "chat_id", cb.Message.ChatID)
		if err := h.client.SendMessage(cb.Message.ChatID, "❌ Ошибка обработки запроса"); err != nil {
			h.logger.Error("failed to send error message", "chat_id", cb.Message.ChatID, "error", err)
		}
		return
	}

	eventID := event.EventID(parts[3])
	evt, err := h.eventService.Get(ctx, eventID)
	if err != nil {
		h.logger.Error("failed to get event", "event_id", string(eventID), "chat_id", cb.Message.ChatID, "error", err)
		if sendErr := h.client.SendMessage(cb.Message.ChatID, "❌ Событие не найдено"); sendErr != nil {
			h.logger.Error("failed to send error message", "chat_id", cb.Message.ChatID, "error", sendErr)
		}
		return
	}

	if evt == nil {
		if sendErr := h.client.SendMessage(cb.Message.ChatID, "❌ Событие не найдено"); sendErr != nil {
			h.logger.Error("failed to send error message", "chat_id", cb.Message.ChatID, "error", sendErr)
		}
		return
	}

	pending, err := h.eventService.ListPendingRegistrations(ctx, eventID)
	if err != nil {
		h.logger.Error("failed to list pending registrations", "event_id", string(eventID), "chat_id", cb.Message.ChatID, "error", err)
		if sendErr := h.client.SendMessage(cb.Message.ChatID, "❌ Ошибка получения списка регистраций"); sendErr != nil {
			h.logger.Error("failed to send error message", "chat_id", cb.Message.ChatID, "error", sendErr)
		}
		return
	}

	// Получаем данные пользователей для каждой регистрации
	registrationsWithUsers := make([]RegistrationWithUser, 0, len(pending))
	for _, reg := range pending {
		usr, err := h.userService.GetByTelegramID(ctx, reg.UserID)
		if err != nil {
			h.logger.Warn("failed to get user", "telegram_id", reg.UserID, "error", err)
		}

		var userName, userSurname string
		if usr != nil {
			userName = usr.Name
			userSurname = usr.Surname
		}

		registrationsWithUsers = append(registrationsWithUsers, RegistrationWithUser{
			Registration: reg,
			UserName:     userName,
			UserSurname:  userSurname,
		})
	}

	text, keyboard := h.formatter.FormatPendingRegistrations(evt.Name, registrationsWithUsers)
	if err := h.client.EditMessageTextAndMarkup(cb.Message.ChatID, cb.Message.MessageID, text, keyboard); err != nil {
		h.logger.Error("failed to edit message with pending registrations", "chat_id", cb.Message.ChatID, "error", err)
	}
}

// handleAdminModerationList обрабатывает список событий для модерации
func (h *Handlers) handleAdminModerationList(ctx context.Context, cb *CallbackQuery) {
	allEvents, err := h.eventService.List(ctx)
	if err != nil {
		h.logger.Error("failed to list events for moderation", "chat_id", cb.Message.ChatID, "error", err)
		if sendErr := h.client.SendMessage(cb.Message.ChatID, "❌ Ошибка получения списка событий"); sendErr != nil {
			h.logger.Error("failed to send error message", "chat_id", cb.Message.ChatID, "error", sendErr)
		}
		return
	}

	// Находим события с pending регистрациями
	var eventsWithPending []event.Event
	locationIDs := make(map[location.LocationID]bool)
	for _, evt := range allEvents {
		pending, err := h.eventService.ListPendingRegistrations(ctx, evt.ID)
		if err != nil {
			continue
		}
		if len(pending) > 0 {
			eventsWithPending = append(eventsWithPending, evt)
			locationIDs[evt.LocationID] = true
		}
	}

	// Загружаем названия локаций
	locationNames := make(map[location.LocationID]string)
	for locID := range locationIDs {
		loc, err := h.locationService.Get(ctx, locID)
		if err == nil && loc != nil {
			locationNames[locID] = loc.Name
		}
	}

	if len(eventsWithPending) == 0 {
		text := "✅ Нет событий с заявками на модерацию"
		keyboard := NewInlineKeyboardMarkup(
			NewInlineKeyboardRow(
				NewInlineKeyboardButtonData("🔙 Назад", "admin:menu"),
			),
		)
		if err := h.client.EditMessageTextAndMarkup(cb.Message.ChatID, cb.Message.MessageID, text, keyboard); err != nil {
			h.logger.Error("failed to edit message", "chat_id", cb.Message.ChatID, "error", err)
		}
		return
	}

	text := "🔔 События с заявками на модерацию:\n\n"
	var rows [][]InlineKeyboardButton
	for _, evt := range eventsWithPending {
		pending, _ := h.eventService.ListPendingRegistrations(ctx, evt.ID)

		// Получаем название локации
		locationName := locationNames[evt.LocationID]
		if locationName == "" {
			locationName = string(evt.LocationID)
		}

		// Форматируем дату и время
		dateStr := evt.Date.Format("02.01.2006")
		timeStr := evt.Date.Format("15:04")

		// Формируем текст с информацией о событии
		text += fmt.Sprintf("📅 %s\n", evt.Name)
		text += fmt.Sprintf("📍 %s\n", locationName)
		text += fmt.Sprintf("🗓️ %s в %s\n", dateStr, timeStr)
		text += fmt.Sprintf("⏳ %d заявок\n\n", len(pending))

		// Формируем текст кнопки с информацией
		buttonText := fmt.Sprintf("%s | %s | %s (%d)", evt.Name, locationName, timeStr, len(pending))
		// Ограничиваем длину текста кнопки (Telegram рекомендует до 64 символов)
		if len(buttonText) > 60 {
			buttonText = buttonText[:57] + "..."
		}

		rows = append(rows, NewInlineKeyboardRow(
			NewInlineKeyboardButtonData(
				buttonText,
				fmt.Sprintf("admin:event:moderation:%s", string(evt.ID)),
			),
		))
	}

	rows = append(rows, NewInlineKeyboardRow(
		NewInlineKeyboardButtonData("🔙 Назад", "admin:menu"),
	))

	keyboard := NewInlineKeyboardMarkup(rows...)
	if err := h.client.EditMessageTextAndMarkup(cb.Message.ChatID, cb.Message.MessageID, text, keyboard); err != nil {
		h.logger.Error("failed to edit message with moderation list", "chat_id", cb.Message.ChatID, "error", err)
	}
}

// handleAdminRegistrationModeration обрабатывает модерацию регистраций
func (h *Handlers) handleAdminRegistrationModeration(ctx context.Context, cb *CallbackQuery) {
	parts := strings.Split(cb.Data, ":")

	// Формат: admin:reg:{userID} - показать детали регистрации
	if len(parts) == 3 && parts[1] == "reg" {
		userIDStr := parts[2]
		var userID int64
		fmt.Sscanf(userIDStr, "%d", &userID)

		// Находим событие с этой регистрацией
		allEvents, err := h.eventService.List(ctx)
		if err != nil {
			h.logger.Error("failed to list events", "chat_id", cb.Message.ChatID, "error", err)
			return
		}

		for _, evt := range allEvents {
			if reg, exists := evt.Registrations[userID]; exists && reg.Status == event.RegistrationStatusPending {
				// Получаем данные пользователя для отображения имени и фамилии
				usr, err := h.userService.GetByTelegramID(ctx, userID)
				if err != nil {
					h.logger.Error("failed to get user", "user_id", userID, "error", err)
				}

				var userName, userSurname string
				if usr != nil {
					userName = usr.Name
					userSurname = usr.Surname
				}

				text, keyboard := h.formatter.FormatRegistrationModeration(evt.Name, userID, userName, userSurname, string(evt.ID))
				if err := h.client.EditMessageTextAndMarkup(cb.Message.ChatID, cb.Message.MessageID, text, keyboard); err != nil {
					h.logger.Error("failed to edit message with registration moderation", "chat_id", cb.Message.ChatID, "error", err)
				}
				return
			}
		}
		return
	}

	// Формат: admin:reg:approve:{eventID}:{userID} или admin:reg:reject:{eventID}:{userID}
	if len(parts) == 5 && (parts[2] == "approve" || parts[2] == "reject") {
		eventID := event.EventID(parts[3])
		var userID int64
		fmt.Sscanf(parts[4], "%d", &userID)

		if parts[2] == "approve" {
			// Получаем данные пользователя для вывода имени и фамилии
			usr, err := h.userService.GetByTelegramID(ctx, userID)
			if err != nil {
				h.logger.Error("failed to get user", "user_id", userID, "error", err)
			}

			err = h.eventService.ApproveRegistration(ctx, eventID, userID)
			if err != nil {
				h.logger.Error("failed to approve registration", "event_id", string(eventID), "user_id", userID, "error", err)
				if sendErr := h.client.SendMessage(cb.Message.ChatID, fmt.Sprintf("❌ Ошибка подтверждения: %v", err)); sendErr != nil {
					h.logger.Error("failed to send error message", "chat_id", cb.Message.ChatID, "error", sendErr)
				}
				return
			}

			// Формируем сообщение с именем и фамилией пользователя
			message := "✅ Регистрация подтверждена"
			if usr != nil {
				message = fmt.Sprintf("✅ Регистрация подтверждена\n\n👤 Пользователь: %s %s", usr.Name, usr.Surname)
			}

			adminMenuKeyboard := NewInlineKeyboardMarkup(
				NewInlineKeyboardRow(
					NewInlineKeyboardButtonData("🔙 В меню администратора", "admin:menu"),
				),
			)
			if err := h.client.SendMessageWithKeyboard(cb.Message.ChatID, message, adminMenuKeyboard); err != nil {
				h.logger.Error("failed to send success message", "chat_id", cb.Message.ChatID, "error", err)
			}
		} else {
			err := h.eventService.RejectRegistration(ctx, eventID, userID)
			if err != nil {
				h.logger.Error("failed to reject registration", "event_id", string(eventID), "user_id", userID, "error", err)
				if sendErr := h.client.SendMessage(cb.Message.ChatID, fmt.Sprintf("❌ Ошибка отклонения: %v", err)); sendErr != nil {
					h.logger.Error("failed to send error message", "chat_id", cb.Message.ChatID, "error", sendErr)
				}
				return
			}
			adminMenuKeyboard := NewInlineKeyboardMarkup(
				NewInlineKeyboardRow(
					NewInlineKeyboardButtonData("🔙 В меню администратора", "admin:menu"),
				),
			)
			if err := h.client.SendMessageWithKeyboard(cb.Message.ChatID, "❌ Регистрация отклонена", adminMenuKeyboard); err != nil {
				h.logger.Error("failed to send success message", "chat_id", cb.Message.ChatID, "error", err)
			}
		}

		// Возвращаемся к списку pending регистраций для этого события
		evt, err := h.eventService.Get(ctx, eventID)
		if err == nil && evt != nil {
			pending, _ := h.eventService.ListPendingRegistrations(ctx, eventID)

			// Получаем данные пользователей для каждой регистрации
			registrationsWithUsers := make([]RegistrationWithUser, 0, len(pending))
			for _, reg := range pending {
				usr, err := h.userService.GetByTelegramID(ctx, reg.UserID)
				if err != nil {
					h.logger.Warn("failed to get user", "telegram_id", reg.UserID, "error", err)
				}

				var userName, userSurname string
				if usr != nil {
					userName = usr.Name
					userSurname = usr.Surname
				}

				registrationsWithUsers = append(registrationsWithUsers, RegistrationWithUser{
					Registration: reg,
					UserName:     userName,
					UserSurname:  userSurname,
				})
			}

			text, keyboard := h.formatter.FormatPendingRegistrations(evt.Name, registrationsWithUsers)
			if err := h.client.EditMessageTextAndMarkup(cb.Message.ChatID, cb.Message.MessageID, text, keyboard); err != nil {
				h.logger.Error("failed to edit message with pending registrations", "chat_id", cb.Message.ChatID, "error", err)
			}
		}
		return
	}
}

// publishEventToChannel публикует анонс события в настроенный канал
func (h *Handlers) publishEventToChannel(ctx context.Context, evt *event.Event) {
	channelID, err := h.settingsService.GetChannelID(ctx)
	if err != nil || channelID == 0 {
		return
	}

	var locationName string
	if loc, err := h.locationService.Get(ctx, evt.LocationID); err == nil && loc != nil {
		locationName = loc.Name
	}

	text, keyboard := h.formatter.FormatChannelEventAnnouncement(evt, locationName, h.client.Username())
	if err := h.client.SendMessageWithKeyboard(channelID, text, keyboard); err != nil {
		h.logger.Error("failed to publish event to channel", "channel_id", channelID, "event_id", string(evt.ID), "error", err)
	}
}

// publishEventCancelledToChannel публикует уведомление об отмене события в канал
func (h *Handlers) publishEventCancelledToChannel(ctx context.Context, evt *event.Event) {
	channelID, err := h.settingsService.GetChannelID(ctx)
	if err != nil || channelID == 0 {
		return
	}

	text := h.formatter.FormatChannelEventCancelled(evt)
	if err := h.client.SendMessage(channelID, text); err != nil {
		h.logger.Error("failed to publish event cancellation to channel", "channel_id", channelID, "event_id", string(evt.ID), "error", err)
	}
}

// handleAdminSetChannelStart начинает процесс настройки канала
func (h *Handlers) handleAdminSetChannelStart(cb *CallbackQuery) {
	h.settingChannel[cb.Message.ChatID] = true
	text := "📢 Настройка канала для публикации событий\n\n" +
		"Выберите способ:\n" +
		"• <b>Переслать</b> любое сообщение из канала сюда\n" +
		"• <b>Ввести ID</b> вручную (например: <code>-1001234567890</code>)\n\n" +
		"Для отмены отправьте /cancel"
	if err := h.client.EditMessageText(cb.Message.ChatID, cb.Message.MessageID, text); err != nil {
		h.logger.Error("failed to edit message for channel setup", "chat_id", cb.Message.ChatID, "error", err)
	}
}

// handleSetChannelInput обрабатывает ввод ID канала или пересланное сообщение
func (h *Handlers) handleSetChannelInput(ctx context.Context, msg *Message) {
	delete(h.settingChannel, msg.ChatID)

	if msg.Text == "/cancel" {
		text, keyboard := h.formatter.FormatAdminMenu()
		if err := h.client.SendMessageWithKeyboard(msg.ChatID, text, keyboard); err != nil {
			h.logger.Error("failed to send admin menu", "chat_id", msg.ChatID, "error", err)
		}
		return
	}

	var channelID int64

	// Вариант 1: пересланное сообщение из канала
	if msg.ForwardFromChatID != 0 {
		channelID = msg.ForwardFromChatID
	} else {
		// Вариант 2: ручной ввод ID
		id, err := strconv.ParseInt(strings.TrimSpace(msg.Text), 10, 64)
		if err != nil {
			keyboard := NewInlineKeyboardMarkup(
				NewInlineKeyboardRow(
					NewInlineKeyboardButtonData("🔙 В меню администратора", "admin:menu"),
				),
			)
			if err := h.client.SendMessageWithKeyboard(msg.ChatID, "❌ Некорректный ID канала. Попробуйте ещё раз или перешлите сообщение из канала.", keyboard); err != nil {
				h.logger.Error("failed to send error message", "chat_id", msg.ChatID, "error", err)
			}
			return
		}
		channelID = id
	}

	if err := h.settingsService.SetChannelID(ctx, channelID); err != nil {
		h.logger.Error("failed to save channel id", "channel_id", channelID, "error", err)
		if sendErr := h.client.SendMessage(msg.ChatID, "❌ Ошибка сохранения канала"); sendErr != nil {
			h.logger.Error("failed to send error message", "chat_id", msg.ChatID, "error", sendErr)
		}
		return
	}

	keyboard := NewInlineKeyboardMarkup(
		NewInlineKeyboardRow(
			NewInlineKeyboardButtonData("🔙 В меню администратора", "admin:menu"),
		),
	)
	if err := h.client.SendMessageWithKeyboard(msg.ChatID, fmt.Sprintf("✅ Канал <code>%d</code> успешно настроен!", channelID), keyboard); err != nil {
		h.logger.Error("failed to send success message", "chat_id", msg.ChatID, "error", err)
	}
}

// handleAdminDeleteEventList показывает список событий для удаления
func (h *Handlers) handleAdminDeleteEventList(ctx context.Context, cb *CallbackQuery) {
	events, err := h.eventService.List(ctx)
	if err != nil {
		h.logger.Error("failed to list events for deletion", "chat_id", cb.Message.ChatID, "error", err)
		if sendErr := h.client.SendMessage(cb.Message.ChatID, "❌ Ошибка получения списка событий"); sendErr != nil {
			h.logger.Error("failed to send error message", "chat_id", cb.Message.ChatID, "error", sendErr)
		}
		return
	}

	if len(events) == 0 {
		keyboard := NewInlineKeyboardMarkup(
			NewInlineKeyboardRow(
				NewInlineKeyboardButtonData("🔙 Назад", "admin:menu"),
			),
		)
		if err := h.client.EditMessageTextAndMarkup(cb.Message.ChatID, cb.Message.MessageID, "📋 Нет событий для удаления", keyboard); err != nil {
			h.logger.Error("failed to edit message", "chat_id", cb.Message.ChatID, "error", err)
		}
		return
	}

	var rows [][]InlineKeyboardButton
	for _, evt := range events {
		label := fmt.Sprintf("🗑️ %s | %s", evt.Name, evt.Date.Format("02.01.2006 15:04"))
		if len(label) > 60 {
			label = label[:57] + "..."
		}
		rows = append(rows, NewInlineKeyboardRow(
			NewInlineKeyboardButtonData(label, fmt.Sprintf("admin:delete_event:confirm:%s", string(evt.ID))),
		))
	}
	rows = append(rows, NewInlineKeyboardRow(
		NewInlineKeyboardButtonData("🔙 Назад", "admin:menu"),
	))

	keyboard := NewInlineKeyboardMarkup(rows...)
	if err := h.client.EditMessageTextAndMarkup(cb.Message.ChatID, cb.Message.MessageID, "🗑️ Выберите событие для удаления:", keyboard); err != nil {
		h.logger.Error("failed to edit message with event deletion list", "chat_id", cb.Message.ChatID, "error", err)
	}
}

// handleAdminConfirmDeleteEvent удаляет событие и уведомляет канал
func (h *Handlers) handleAdminConfirmDeleteEvent(ctx context.Context, cb *CallbackQuery) {
	parts := strings.Split(cb.Data, ":")
	if len(parts) != 4 {
		h.logger.Warn("invalid delete event callback data", "callback_data", cb.Data)
		return
	}

	eventID := event.EventID(parts[3])
	evt, err := h.eventService.Get(ctx, eventID)
	if err != nil || evt == nil {
		if sendErr := h.client.SendMessage(cb.Message.ChatID, "❌ Событие не найдено"); sendErr != nil {
			h.logger.Error("failed to send error message", "chat_id", cb.Message.ChatID, "error", sendErr)
		}
		return
	}

	if err := h.eventService.Delete(ctx, eventID); err != nil {
		h.logger.Error("failed to delete event", "event_id", string(eventID), "error", err)
		if sendErr := h.client.SendMessage(cb.Message.ChatID, fmt.Sprintf("❌ Ошибка удаления: %v", err)); sendErr != nil {
			h.logger.Error("failed to send error message", "chat_id", cb.Message.ChatID, "error", sendErr)
		}
		return
	}

	// Уведомляем канал об отмене
	h.publishEventCancelledToChannel(ctx, evt)

	keyboard := NewInlineKeyboardMarkup(
		NewInlineKeyboardRow(
			NewInlineKeyboardButtonData("🔙 В меню администратора", "admin:menu"),
		),
	)
	if err := h.client.EditMessageTextAndMarkup(cb.Message.ChatID, cb.Message.MessageID,
		fmt.Sprintf("✅ Событие «%s» удалено", evt.Name), keyboard); err != nil {
		h.logger.Error("failed to edit message after event deletion", "chat_id", cb.Message.ChatID, "error", err)
	}
}
