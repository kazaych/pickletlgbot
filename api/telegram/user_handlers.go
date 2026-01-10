package telegram

import (
	"context"
	"fmt"
	"pickletlgbot/internal/domain/event"
	"pickletlgbot/internal/domain/location"
	"pickletlgbot/internal/domain/user"
	"strings"
)

// handleStart обрабатывает команду /start
func (h *Handlers) handleStart(msg *Message) {
	text, keyboard := h.formatter.FormatMainMenu()
	if err := h.client.SendMessageWithKeyboard(msg.ChatID, text, keyboard); err != nil {
		h.logger.Error("failed to send main menu", "chat_id", msg.ChatID, "error", err)
	}
}

// handleLocations обрабатывает запрос списка локаций
func (h *Handlers) handleLocations(ctx context.Context, cb *CallbackQuery) {
	locations, err := h.locationService.List(ctx)
	if err != nil {
		h.logger.Error("failed to list locations", "chat_id", cb.Message.ChatID, "error", err)
		if sendErr := h.client.SendMessage(cb.Message.ChatID, "Ошибка получения локаций"); sendErr != nil {
			h.logger.Error("failed to send error message", "chat_id", cb.Message.ChatID, "error", sendErr)
		}
		return
	}

	// Конвертируем []Location в []*Location для форматтера
	locationPtrs := make([]*location.Location, len(locations))
	for i := range locations {
		locationPtrs[i] = &locations[i]
	}

	text, keyboard := h.formatter.FormatLocationsListForUsers(locationPtrs)
	if keyboard != nil {
		if err := h.client.EditMessageTextAndMarkup(cb.Message.ChatID, cb.Message.MessageID, text, keyboard); err != nil {
			h.logger.Error("failed to edit message with locations list", "chat_id", cb.Message.ChatID, "error", err)
		}
	} else {
		if err := h.client.EditMessageText(cb.Message.ChatID, cb.Message.MessageID, text); err != nil {
			h.logger.Error("failed to edit message", "chat_id", cb.Message.ChatID, "error", err)
		}
	}
}

// handleLocationSelection обрабатывает выбор конкретной локации
func (h *Handlers) handleLocationSelection(ctx context.Context, cb *CallbackQuery) {
	// Парсим ID из callback data (формат: loc:{id})
	parts := strings.Split(cb.Data, ":")
	if len(parts) != 2 {
		h.logger.Warn("invalid location callback data format", "callback_data", cb.Data, "chat_id", cb.Message.ChatID)
		if err := h.client.SendMessage(cb.Message.ChatID, "Ошибка обработки локации"); err != nil {
			h.logger.Error("failed to send error message", "chat_id", cb.Message.ChatID, "error", err)
		}
		return
	}

	locationIDStr := parts[1]
	locationID := location.LocationID(locationIDStr)

	loc, err := h.locationService.Get(ctx, locationID)
	if err != nil {
		h.logger.Error("failed to get location", "location_id", locationIDStr, "chat_id", cb.Message.ChatID, "error", err)
		if sendErr := h.client.SendMessage(cb.Message.ChatID, "Локация не найдена"); sendErr != nil {
			h.logger.Error("failed to send error message", "chat_id", cb.Message.ChatID, "error", sendErr)
		}
		return
	}

	text, keyboard := h.formatter.FormatLocationDetails(loc)
	if err := h.client.EditMessageTextAndMarkup(cb.Message.ChatID, cb.Message.MessageID, text, keyboard); err != nil {
		h.logger.Error("failed to edit message with location details", "chat_id", cb.Message.ChatID, "error", err)
	}
}

// handleBackToMain обрабатывает возврат в главное меню
func (h *Handlers) handleBackToMain(cb *CallbackQuery) {
	text, keyboard := h.formatter.FormatMainMenu()
	if err := h.client.EditMessageTextAndMarkup(cb.Message.ChatID, cb.Message.MessageID, text, keyboard); err != nil {
		h.logger.Error("failed to edit message with main menu", "chat_id", cb.Message.ChatID, "error", err)
	}
}

// handleLocationEvents обрабатывает запрос списка событий по локации
func (h *Handlers) handleLocationEvents(ctx context.Context, cb *CallbackQuery) {
	// Парсим ID локации из callback data (формат: loc:events:{locationID})
	parts := strings.Split(cb.Data, ":")
	if len(parts) != 3 {
		h.logger.Warn("invalid location events callback data format", "callback_data", cb.Data, "chat_id", cb.Message.ChatID)
		if err := h.client.SendMessage(cb.Message.ChatID, "❌ Ошибка обработки запроса"); err != nil {
			h.logger.Error("failed to send error message", "chat_id", cb.Message.ChatID, "error", err)
		}
		return
	}

	locationIDStr := parts[2]
	locationID := location.LocationID(locationIDStr)

	// Получаем локацию для отображения названия
	loc, err := h.locationService.Get(ctx, locationID)
	if err != nil {
		h.logger.Error("failed to get location", "location_id", locationIDStr, "chat_id", cb.Message.ChatID, "error", err)
		if sendErr := h.client.SendMessage(cb.Message.ChatID, "❌ Локация не найдена"); sendErr != nil {
			h.logger.Error("failed to send error message", "chat_id", cb.Message.ChatID, "error", sendErr)
		}
		return
	}

	// Получаем события по локации
	events, err := h.eventService.ListByLocation(ctx, locationID)
	if err != nil {
		h.logger.Error("failed to list events by location", "location_id", locationIDStr, "chat_id", cb.Message.ChatID, "error", err)
		if sendErr := h.client.SendMessage(cb.Message.ChatID, "❌ Ошибка получения списка событий"); sendErr != nil {
			h.logger.Error("failed to send error message", "chat_id", cb.Message.ChatID, "error", sendErr)
		}
		return
	}

	// Создаем map с названием локации
	locationNames := make(map[location.LocationID]string)
	locationNames[locationID] = loc.Name

	// Используем кастомную кнопку "Назад" для возврата к локации
	text, keyboard := h.formatter.FormatEventsListForUsersWithBack(events, locationNames, fmt.Sprintf("loc:%s", string(locationID)), "🔙 К локации")
	if keyboard != nil {
		if err := h.client.EditMessageTextAndMarkup(cb.Message.ChatID, cb.Message.MessageID, text, keyboard); err != nil {
			h.logger.Error("failed to edit message with events list", "chat_id", cb.Message.ChatID, "error", err)
		}
	} else {
		if err := h.client.EditMessageText(cb.Message.ChatID, cb.Message.MessageID, text); err != nil {
			h.logger.Error("failed to edit message", "chat_id", cb.Message.ChatID, "error", err)
		}
	}
}

// handleEvents обрабатывает запрос списка событий
func (h *Handlers) handleEvents(ctx context.Context, cb *CallbackQuery) {
	events, err := h.eventService.List(ctx)
	if err != nil {
		h.logger.Error("failed to list events", "chat_id", cb.Message.ChatID, "error", err)
		if sendErr := h.client.SendMessage(cb.Message.ChatID, "❌ Ошибка получения списка событий"); sendErr != nil {
			h.logger.Error("failed to send error message", "chat_id", cb.Message.ChatID, "error", sendErr)
		}
		return
	}

	// Собираем уникальные LocationID
	locationIDs := make(map[location.LocationID]bool)
	for _, evt := range events {
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

	text, keyboard := h.formatter.FormatEventsListForUsers(events, locationNames)
	if keyboard != nil {
		if err := h.client.EditMessageTextAndMarkup(cb.Message.ChatID, cb.Message.MessageID, text, keyboard); err != nil {
			h.logger.Error("failed to edit message with events list", "chat_id", cb.Message.ChatID, "error", err)
		}
	} else {
		if err := h.client.EditMessageText(cb.Message.ChatID, cb.Message.MessageID, text); err != nil {
			h.logger.Error("failed to edit message", "chat_id", cb.Message.ChatID, "error", err)
		}
	}
}

// handleEventSelection обрабатывает выбор конкретного события
func (h *Handlers) handleEventSelection(ctx context.Context, cb *CallbackQuery) {
	// Парсим ID из callback data (формат: event:{id})
	parts := strings.Split(cb.Data, ":")
	if len(parts) != 2 {
		h.logger.Warn("invalid event callback data format", "callback_data", cb.Data, "chat_id", cb.Message.ChatID)
		if err := h.client.SendMessage(cb.Message.ChatID, "❌ Ошибка обработки события"); err != nil {
			h.logger.Error("failed to send error message", "chat_id", cb.Message.ChatID, "error", err)
		}
		return
	}

	eventIDStr := parts[1]
	eventID := event.EventID(eventIDStr)

	evt, err := h.eventService.Get(ctx, eventID)
	if err != nil {
		h.logger.Error("failed to get event", "event_id", eventIDStr, "chat_id", cb.Message.ChatID, "error", err)
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

	text, keyboard := h.formatter.FormatEventDetailsForUsers(evt, cb.From.ID)
	if err := h.client.EditMessageTextAndMarkup(cb.Message.ChatID, cb.Message.MessageID, text, keyboard); err != nil {
		h.logger.Error("failed to edit message with event details", "chat_id", cb.Message.ChatID, "error", err)
	}
}

// handleEventRegistration обрабатывает регистрацию пользователя на событие
func (h *Handlers) handleEventRegistration(ctx context.Context, cb *CallbackQuery) {
	// Парсим ID из callback data (формат: event:register:{id})
	parts := strings.Split(cb.Data, ":")
	if len(parts) != 3 {
		h.logger.Warn("invalid event registration callback data format", "callback_data", cb.Data, "chat_id", cb.Message.ChatID)
		if err := h.client.SendMessage(cb.Message.ChatID, "❌ Ошибка обработки запроса"); err != nil {
			h.logger.Error("failed to send error message", "chat_id", cb.Message.ChatID, "error", err)
		}
		return
	}

	eventIDStr := parts[2]
	eventID := event.EventID(eventIDStr)
	userID := cb.From.ID

	// Проверяем, существует ли пользователь в базе
	exists, err := h.userService.IsUserExists(ctx, userID)
	if err != nil {
		h.logger.Error("failed to check user existence", "user_id", userID, "error", err)
		if sendErr := h.client.SendMessage(cb.Message.ChatID, "❌ Ошибка проверки данных"); sendErr != nil {
			h.logger.Error("failed to send error message", "chat_id", cb.Message.ChatID, "error", sendErr)
		}
		return
	}

	// Если пользователь не существует, начинаем процесс регистрации
	if !exists {
		// Создаем состояние регистрации
		state := &UserRegistrationState{
			EventID: eventID,
			Step:    "name",
		}
		h.setUserRegistrationState(userID, state)

		// Просим ввести имя
		if err := h.client.SendMessage(cb.Message.ChatID, "📝 Для регистрации на событие необходимо указать ваши данные.\n\nВведите ваше имя:"); err != nil {
			h.logger.Error("failed to send name prompt", "chat_id", cb.Message.ChatID, "error", err)
		}
		return
	}

	// Пользователь существует, регистрируем на событие
	h.registerUserToEvent(ctx, eventID, userID, cb.Message.ChatID, cb.Message.MessageID)
}

// registerUserToEvent регистрирует пользователя на событие
func (h *Handlers) registerUserToEvent(ctx context.Context, eventID event.EventID, userID int64, chatID int64, messageID int) {
	err := h.eventService.RegisterUserToEvent(ctx, eventID, userID)
	if err != nil {
		h.logger.Error("failed to register user for event", "event_id", string(eventID), "user_id", userID, "chat_id", chatID, "error", err)

		errorMsg := "❌ Ошибка регистрации"
		if err == event.ErrEventFull {
			errorMsg = "❌ Все места заняты"
		} else if err == event.ErrUserAlreadyRegistered {
			errorMsg = "⚠️ Вы уже зарегистрированы на это событие"
		}

		if sendErr := h.client.SendMessage(chatID, errorMsg); sendErr != nil {
			h.logger.Error("failed to send error message", "chat_id", chatID, "error", sendErr)
		}
		return
	}

	// Получаем обновленное событие для отображения
	evt, err := h.eventService.Get(ctx, eventID)
	if err != nil || evt == nil {
		if sendErr := h.client.SendMessage(chatID, "✅ Заявка подана! Ожидайте подтверждения администратора."); sendErr != nil {
			h.logger.Error("failed to send success message", "chat_id", chatID, "error", sendErr)
		}
		return
	}

	text, keyboard := h.formatter.FormatEventDetailsForUsers(evt, userID)
	if messageID > 0 {
		// Редактируем существующее сообщение
		if err := h.client.EditMessageTextAndMarkup(chatID, messageID, text, keyboard); err != nil {
			h.logger.Error("failed to edit message with event details", "chat_id", chatID, "error", err)
		}
	} else {
		// Отправляем новое сообщение
		if err := h.client.SendMessageWithKeyboard(chatID, text, keyboard); err != nil {
			h.logger.Error("failed to send message with event details", "chat_id", chatID, "error", err)
		}
	}
}

// handleUserRegistrationStep обрабатывает шаги регистрации пользователя (ввод имени и фамилии)
func (h *Handlers) handleUserRegistrationStep(ctx context.Context, msg *Message, state *UserRegistrationState) {
	text := strings.TrimSpace(msg.Text)
	if text == "" {
		if err := h.client.SendMessage(msg.ChatID, "❌ Пожалуйста, введите непустое значение"); err != nil {
			h.logger.Error("failed to send validation error", "chat_id", msg.ChatID, "error", err)
		}
		return
	}

	switch state.Step {
	case "name":
		// Сохраняем имя и просим фамилию
		state.FirstName = text
		state.Step = "surname"
		if err := h.client.SendMessage(msg.ChatID, "✅ Имя сохранено.\n\nВведите вашу фамилию:"); err != nil {
			h.logger.Error("failed to send surname prompt", "chat_id", msg.ChatID, "error", err)
		}

	case "surname":
		// Сохраняем фамилию и создаем пользователя
		user := &user.User{
			TelegramID: msg.From.ID,
			Name:       state.FirstName,
			Surname:    text,
		}

		// Создаем пользователя в базе
		if err := h.userService.CreateUser(ctx, user); err != nil {
			h.logger.Error("failed to create user", "user_id", msg.From.ID, "error", err)
			if sendErr := h.client.SendMessage(msg.ChatID, "❌ Ошибка сохранения данных. Попробуйте позже."); sendErr != nil {
				h.logger.Error("failed to send error message", "chat_id", msg.ChatID, "error", sendErr)
			}
			h.clearUserRegistrationState(msg.From.ID)
			return
		}

		// Очищаем состояние регистрации
		h.clearUserRegistrationState(msg.From.ID)

		// Регистрируем пользователя на событие
		if err := h.client.SendMessage(msg.ChatID, "✅ Данные сохранены! Регистрирую на событие..."); err != nil {
			h.logger.Error("failed to send confirmation", "chat_id", msg.ChatID, "error", err)
		}

		// Регистрируем на событие (messageID = 0, так как это новое сообщение)
		h.registerUserToEvent(ctx, state.EventID, msg.From.ID, msg.ChatID, 0)
	}
}

// handleEventUnregister обрабатывает отмену регистрации пользователя на событие
func (h *Handlers) handleEventUnregister(ctx context.Context, cb *CallbackQuery) {
	// Парсим ID из callback data (формат: event:unregister:{id})
	parts := strings.Split(cb.Data, ":")
	if len(parts) != 3 {
		h.logger.Warn("invalid event unregister callback data format", "callback_data", cb.Data, "chat_id", cb.Message.ChatID)
		if err := h.client.SendMessage(cb.Message.ChatID, "❌ Ошибка обработки запроса"); err != nil {
			h.logger.Error("failed to send error message", "chat_id", cb.Message.ChatID, "error", err)
		}
		return
	}

	eventIDStr := parts[2]
	eventID := event.EventID(eventIDStr)
	userID := cb.From.ID

	// Отменяем регистрацию
	err := h.eventService.UnregisterUser(ctx, eventID, userID)
	if err != nil {
		h.logger.Error("failed to unregister user from event", "event_id", eventIDStr, "user_id", userID, "chat_id", cb.Message.ChatID, "error", err)

		errorMsg := "❌ Ошибка отмены регистрации"
		if err == event.ErrRegistrationNotFound {
			errorMsg = "⚠️ Вы не зарегистрированы на это событие"
		}

		if sendErr := h.client.SendMessage(cb.Message.ChatID, errorMsg); sendErr != nil {
			h.logger.Error("failed to send error message", "chat_id", cb.Message.ChatID, "error", sendErr)
		}
		return
	}

	// Получаем обновленное событие для отображения
	evt, err := h.eventService.Get(ctx, eventID)
	if err != nil || evt == nil {
		if sendErr := h.client.SendMessage(cb.Message.ChatID, "✅ Регистрация отменена"); sendErr != nil {
			h.logger.Error("failed to send success message", "chat_id", cb.Message.ChatID, "error", sendErr)
		}
		return
	}

	text, keyboard := h.formatter.FormatEventDetailsForUsers(evt, userID)
	if err := h.client.EditMessageTextAndMarkup(cb.Message.ChatID, cb.Message.MessageID, text, keyboard); err != nil {
		h.logger.Error("failed to edit message with event details", "chat_id", cb.Message.ChatID, "error", err)
	}
}

// handleEventUsersList обрабатывает запрос списка участников события
func (h *Handlers) handleEventUsersList(ctx context.Context, cb *CallbackQuery) {
	// Парсим ID из callback data (формат: event:users:{id})
	parts := strings.Split(cb.Data, ":")
	if len(parts) != 3 {
		h.logger.Warn("invalid event users callback data format", "callback_data", cb.Data, "chat_id", cb.Message.ChatID)
		if err := h.client.SendMessage(cb.Message.ChatID, "❌ Ошибка обработки запроса"); err != nil {
			h.logger.Error("failed to send error message", "chat_id", cb.Message.ChatID, "error", err)
		}
		return
	}

	eventIDStr := parts[2]
	eventID := event.EventID(eventIDStr)

	// Получаем событие
	evt, err := h.eventService.Get(ctx, eventID)
	if err != nil {
		h.logger.Error("failed to get event", "event_id", eventIDStr, "chat_id", cb.Message.ChatID, "error", err)
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

	// Собираем список пользователей с их статусами
	var usersWithStatus []UserWithStatus
	for telegramID, reg := range evt.Registrations {
		usr, err := h.userService.GetByTelegramID(ctx, telegramID)
		if err != nil {
			h.logger.Warn("failed to get user", "telegram_id", telegramID, "error", err)
			// Продолжаем, даже если не удалось получить пользователя
			continue
		}
		if usr != nil {
			usersWithStatus = append(usersWithStatus, UserWithStatus{
				User:   usr,
				Status: reg.Status,
			})
		}
	}

	// Форматируем и отправляем список
	text, keyboard := h.formatter.FormatEventUsersList(evt.Name, usersWithStatus, string(eventID))
	if err := h.client.EditMessageTextAndMarkup(cb.Message.ChatID, cb.Message.MessageID, text, keyboard); err != nil {
		h.logger.Error("failed to edit message with users list", "chat_id", cb.Message.ChatID, "error", err)
	}
}
