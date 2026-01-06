package telegram

import (
	"context"
	"pickletlgbot/internal/domain/event"
	"pickletlgbot/internal/domain/location"
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

	// Регистрируем пользователя
	err := h.eventService.RegisterUser(ctx, eventID, userID)
	if err != nil {
		h.logger.Error("failed to register user for event", "event_id", eventIDStr, "user_id", userID, "chat_id", cb.Message.ChatID, "error", err)

		errorMsg := "❌ Ошибка регистрации"
		if err == event.ErrEventFull {
			errorMsg = "❌ Все места заняты"
		} else if err == event.ErrUserAlreadyRegistered {
			errorMsg = "⚠️ Вы уже зарегистрированы на это событие"
		}

		if sendErr := h.client.SendMessage(cb.Message.ChatID, errorMsg); sendErr != nil {
			h.logger.Error("failed to send error message", "chat_id", cb.Message.ChatID, "error", sendErr)
		}
		return
	}

	// Получаем обновленное событие для отображения
	evt, err := h.eventService.Get(ctx, eventID)
	if err != nil || evt == nil {
		if sendErr := h.client.SendMessage(cb.Message.ChatID, "✅ Заявка подана! Ожидайте подтверждения администратора."); sendErr != nil {
			h.logger.Error("failed to send success message", "chat_id", cb.Message.ChatID, "error", sendErr)
		}
		return
	}

	text, keyboard := h.formatter.FormatEventDetailsForUsers(evt, userID)
	if err := h.client.EditMessageTextAndMarkup(cb.Message.ChatID, cb.Message.MessageID, text, keyboard); err != nil {
		h.logger.Error("failed to edit message with event details", "chat_id", cb.Message.ChatID, "error", err)
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
