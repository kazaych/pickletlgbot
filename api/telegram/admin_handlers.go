package telegram

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
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
		text := h.formatter.FormatCreateLocationPrompt()
		if err := h.client.SendMessage(msg.ChatID, text); err != nil {
			h.logger.Error("failed to send create location prompt", "chat_id", msg.ChatID, "error", err)
		}

	case "/admin_delete_location":
		text := h.formatter.FormatDeleteLocationPrompt()
		if err := h.client.SendMessage(msg.ChatID, text); err != nil {
			h.logger.Error("failed to send delete location prompt", "chat_id", msg.ChatID, "error", err)
		}

	default:
		if strings.HasPrefix(msg.Text, "/admin_create_location ") {
			input := strings.TrimPrefix(msg.Text, "/admin_create_location ")
			input = strings.TrimSpace(input)
			if input == "" {
				if err := h.client.SendMessage(msg.ChatID, "❌ Укажите данные локации\nИспользование: /admin_create_location <название>|<адрес>\nИли: /admin_create_location <название>"); err != nil {
					h.logger.Error("failed to send location data required message", "chat_id", msg.ChatID, "error", err)
				}
				return
			}
			h.handleAdminCreateLocation(msg, input)
		}
	}
}

// handleAdminCallback обрабатывает callback-запросы администратора
func (h *Handlers) handleAdminCallback(ctx context.Context, cb *CallbackQuery) {
	switch cb.Data {
	case "admin:create_location":
		text := h.formatter.FormatCreateLocationPrompt()
		if err := h.client.SendMessage(cb.Message.ChatID, text); err != nil {
			h.logger.Error("failed to send create location prompt", "chat_id", cb.Message.ChatID, "error", err)
		}
	case "admin:delete_location":
		h.handleAdminDeleteLocation(ctx, cb)
	case "admin:list_locations":
		h.handleAdminListLocations(ctx, cb)
	case "admin:menu":
		text, keyboard := h.formatter.FormatAdminMenu()
		if err := h.client.EditMessageTextAndMarkup(cb.Message.ChatID, cb.Message.MessageID, text, keyboard); err != nil {
			h.logger.Error("failed to edit message with admin menu", "chat_id", cb.Message.ChatID, "error", err)
		}
	default:
		// Обработка динамических callback'ов для удаления (формат: admin:delete:{locationID})
		if strings.HasPrefix(cb.Data, "admin:delete:") {
			h.handleAdminConfirmDeleteLocation(ctx, cb)
		}
	}
}

// handleAdminCreateLocation обрабатывает создание локации администратором
// Формат входных данных: "Название|Адрес|URL" или "Название|Адрес" или "Название"
func (h *Handlers) handleAdminCreateLocation(msg *Message, input string) {
	ctx := context.Background()

	// Парсим название, адрес и URL (разделитель: |)
	var name, address, addressUrl string
	parts := strings.Split(input, "|")

	if len(parts) >= 1 {
		name = strings.TrimSpace(parts[0])
	}
	if len(parts) >= 2 {
		address = strings.TrimSpace(parts[1])
	}
	if len(parts) >= 3 {
		addressUrl = strings.TrimSpace(parts[2])
	}

	// Если нет разделителя |, проверяем перенос строки
	if !strings.Contains(input, "|") && strings.Contains(input, "\n") {
		lines := strings.SplitN(input, "\n", 3)
		if len(lines) >= 1 {
			name = strings.TrimSpace(lines[0])
		}
		if len(lines) >= 2 {
			address = strings.TrimSpace(lines[1])
		}
		if len(lines) >= 3 {
			addressUrl = strings.TrimSpace(lines[2])
		}
	}

	if name == "" {
		if err := h.client.SendMessage(msg.ChatID, "❌ Название локации не может быть пустым"); err != nil {
			h.logger.Error("failed to send error message", "chat_id", msg.ChatID, "error", err)
		}
		return
	}

	location, err := h.locationService.CreateLocation(ctx, name, address, addressUrl)
	if err != nil {
		h.logger.Error("failed to create location", "location_name", name, "chat_id", msg.ChatID, "error", err)
		if sendErr := h.client.SendMessage(msg.ChatID, fmt.Sprintf("❌ Ошибка создания локации: %v", err)); sendErr != nil {
			h.logger.Error("failed to send error message", "chat_id", msg.ChatID, "error", sendErr)
		}
		return
	}

	text, keyboard := h.formatter.FormatLocationCreated(location)
	if err := h.client.SendMessageWithKeyboard(msg.ChatID, text, keyboard); err != nil {
		h.logger.Error("failed to send location created message", "chat_id", msg.ChatID, "location_id", location.ID.String(), "error", err)
	}
}

// handleAdminListLocations обрабатывает запрос списка локаций администратором
func (h *Handlers) handleAdminListLocations(ctx context.Context, cb *CallbackQuery) {
	locations, err := h.locationService.ListLocations(ctx)
	if err != nil {
		h.logger.Error("failed to list locations for admin", "chat_id", cb.Message.ChatID, "error", err)
		if sendErr := h.client.SendMessage(cb.Message.ChatID, "❌ Ошибка получения списка локаций"); sendErr != nil {
			h.logger.Error("failed to send error message", "chat_id", cb.Message.ChatID, "error", sendErr)
		}
		return
	}

	text, keyboard := h.formatter.FormatLocationsListForAdmin(locations)

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
	locations, err := h.locationService.ListLocations(ctx)
	if err != nil {
		h.logger.Error("failed to list locations for deletion", "chat_id", cb.Message.ChatID, "error", err)
		if sendErr := h.client.SendMessage(cb.Message.ChatID, "❌ Ошибка получения списка локаций"); sendErr != nil {
			h.logger.Error("failed to send error message", "chat_id", cb.Message.ChatID, "error", sendErr)
		}
		return
	}

	text, keyboard := h.formatter.FormatDeleteLocationList(locations)
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
	locationID, err := uuid.Parse(locationIDStr)
	if err != nil {
		h.logger.Warn("invalid location ID format", "location_id", locationIDStr, "chat_id", cb.Message.ChatID, "error", err)
		if sendErr := h.client.SendMessage(cb.Message.ChatID, "❌ Ошибка обработки запроса"); sendErr != nil {
			h.logger.Error("failed to send error message", "chat_id", cb.Message.ChatID, "error", sendErr)
		}
		return
	}

	// Получаем информацию о локации перед удалением
	location, err := h.locationService.GetLocation(ctx, locationID)
	if err != nil {
		h.logger.Error("failed to get location for deletion", "location_id", locationID.String(), "chat_id", cb.Message.ChatID, "error", err)
		if sendErr := h.client.SendMessage(cb.Message.ChatID, "❌ Локация не найдена"); sendErr != nil {
			h.logger.Error("failed to send error message", "chat_id", cb.Message.ChatID, "error", sendErr)
		}
		return
	}

	locationName := location.Name

	// Удаляем локацию
	err = h.locationService.DeleteLocation(ctx, locationID)
	if err != nil {
		h.logger.Error("failed to delete location", "location_id", locationID.String(), "chat_id", cb.Message.ChatID, "error", err)
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
