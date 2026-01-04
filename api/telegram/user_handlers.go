package telegram

import (
	"context"
	"strings"

	"github.com/google/uuid"
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
	locations, err := h.locationService.ListLocations(ctx)
	if err != nil {
		h.logger.Error("failed to list locations", "chat_id", cb.Message.ChatID, "error", err)
		if sendErr := h.client.SendMessage(cb.Message.ChatID, "Ошибка получения локаций"); sendErr != nil {
			h.logger.Error("failed to send error message", "chat_id", cb.Message.ChatID, "error", sendErr)
		}
		return
	}

	text, keyboard := h.formatter.FormatLocationsListForUsers(locations)
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
	locationID, err := uuid.Parse(locationIDStr)
	if err != nil {
		h.logger.Warn("invalid location ID format", "location_id", locationIDStr, "chat_id", cb.Message.ChatID, "error", err)
		if sendErr := h.client.SendMessage(cb.Message.ChatID, "Ошибка обработки локации"); sendErr != nil {
			h.logger.Error("failed to send error message", "chat_id", cb.Message.ChatID, "error", sendErr)
		}
		return
	}

	location, err := h.locationService.GetLocation(ctx, locationID)
	if err != nil {
		h.logger.Error("failed to get location", "location_id", locationID.String(), "chat_id", cb.Message.ChatID, "error", err)
		if sendErr := h.client.SendMessage(cb.Message.ChatID, "Локация не найдена"); sendErr != nil {
			h.logger.Error("failed to send error message", "chat_id", cb.Message.ChatID, "error", sendErr)
		}
		return
	}

	text, keyboard := h.formatter.FormatLocationDetails(location)
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
