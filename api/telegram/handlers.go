package telegram

import (
	"context"
	"kitchenBot/domain/event"
	"kitchenBot/domain/location"
	"log/slog"
	"os"
	"strconv"
	"strings"
)

// Handlers обрабатывает обновления от Telegram и маппит их в вызовы бизнес-сервисов
type Handlers struct {
	locationService *location.Service
	eventService    *event.Service
	client          *Client
	formatter       *Formatter
	adminIDs        []int64
	logger          *slog.Logger
}

// NewHandlers создает новый набор обработчиков
func NewHandlers(
	locationService *location.Service,
	eventService *event.Service,
	client *Client,
) *Handlers {
	adminIDs := parseAdminIDs()
	logger := slog.Default()
	return &Handlers{
		locationService: locationService,
		eventService:    eventService,
		client:          client,
		formatter:       NewFormatter(),
		adminIDs:        adminIDs,
		logger:          logger,
	}
}

// HandleUpdate обрабатывает обновление от Telegram
func (h *Handlers) HandleUpdate(update *Update) {
	if update.Message != nil {
		h.HandleMessage(update.Message)
	}

	if update.CallbackQuery != nil {
		h.HandleCallback(update.CallbackQuery)
	}
}

// HandleMessage обрабатывает текстовые сообщения
func (h *Handlers) HandleMessage(msg *Message) {
	if msg == nil {
		return
	}

	// Проверяем админ-команды
	if strings.HasPrefix(msg.Text, "/admin") {
		h.handleAdminCommand(msg)
		return
	}

	// Обрабатываем обычные команды
	switch msg.Text {
	case "/start":
		h.handleStart(msg)
	default:
		// Если это не команда, проверяем, не создается ли локация админом
		if h.isAdmin(msg.From.ID) && !strings.HasPrefix(msg.Text, "/") {
			h.handleAdminCreateLocation(msg, msg.Text)
		} else {
			if err := h.client.SendMessage(msg.ChatID, "Нажмите /start для меню"); err != nil {
				h.logger.Error("failed to send start prompt", "chat_id", msg.ChatID, "error", err)
			}
		}
	}
}

// HandleCallback обрабатывает callback запросы
func (h *Handlers) HandleCallback(cb *CallbackQuery) {
	if cb == nil || cb.Message == nil {
		return
	}

	// Подтверждаем нажатие кнопки
	if err := h.client.AnswerCallbackQuery(cb.ID); err != nil {
		h.logger.Error("failed to answer callback query", "callback_id", cb.ID, "error", err)
	}

	ctx := context.Background()

	// Проверяем админ callback'и
	if strings.HasPrefix(cb.Data, "admin:") {
		if !h.isAdmin(cb.From.ID) {
			if err := h.client.SendMessage(cb.Message.ChatID, "❌ У вас нет прав администратора"); err != nil {
				h.logger.Error("failed to send admin access denied message", "chat_id", cb.Message.ChatID, "error", err)
			}
			return
		}
		h.handleAdminCallback(ctx, cb)
		return
	}

	// Обрабатываем обычные callback'и
	switch cb.Data {
	case "locations":
		h.handleLocations(ctx, cb)
	case "back:main":
		h.handleBackToMain(cb)
	case "admin":
		// Обработка кнопки "Администратор" из главного меню
		if !h.isAdmin(cb.From.ID) {
			if err := h.client.SendMessage(cb.Message.ChatID, "❌ У вас нет прав администратора"); err != nil {
				h.logger.Error("failed to send admin access denied message", "chat_id", cb.Message.ChatID, "error", err)
			}
			return
		}
		text, keyboard := h.formatter.FormatAdminMenu()
		if err := h.client.EditMessageTextAndMarkup(cb.Message.ChatID, cb.Message.MessageID, text, keyboard); err != nil {
			h.logger.Error("failed to edit message with admin menu", "chat_id", cb.Message.ChatID, "error", err)
		}
	default:
		// Обработка динамических callback'ов
		if strings.HasPrefix(cb.Data, "loc:") {
			h.handleLocationSelection(ctx, cb)
		}
	}
}

// isAdmin проверяет, является ли пользователь администратором
func (h *Handlers) isAdmin(userID int64) bool {
	for _, id := range h.adminIDs {
		if id == userID {
			return true
		}
	}
	return false
}

// parseAdminIDs парсит список ID администраторов из переменной окружения
func parseAdminIDs() []int64 {
	adminIDsStr := os.Getenv("ADMIN_IDS")
	if adminIDsStr == "" {
		return nil
	}

	var adminIDs []int64
	ids := strings.Split(adminIDsStr, ",")
	for _, idStr := range ids {
		idStr = strings.TrimSpace(idStr)
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			continue
		}
		adminIDs = append(adminIDs, id)
	}

	return adminIDs
}
