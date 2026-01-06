package telegram

import (
	"context"
	"log/slog"
	"os"
	"pickletlgbot/internal/domain/event"
	"pickletlgbot/internal/domain/location"
	"strconv"
	"strings"
	"time"
)

// EventCreationState хранит состояние создания события
type EventCreationState struct {
	Step       string // "type", "max_players", "name", "date", "trainer"
	LocationID location.LocationID
	EventType  event.EventType
	MaxPlayers int
	EventName  string
	EventDate  time.Time
	Trainer    string
}

// Handlers обрабатывает обновления от Telegram и маппит их в вызовы бизнес-сервисов
type Handlers struct {
	locationService location.LocationService
	eventService    event.EventService
	client          *Client
	formatter       *Formatter
	adminIDs        []int64
	logger          *slog.Logger
	// Временное хранилище для состояния создания событий
	creatingEvents map[int64]*EventCreationState
}

// NewHandlers создает новый набор обработчиков
func NewHandlers(
	locationService location.LocationService,
	eventService event.EventService,
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
		creatingEvents:  make(map[int64]*EventCreationState),
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
		// Если это не команда, проверяем, не создается ли что-то админом
		if h.isAdmin(msg.From.ID) && !strings.HasPrefix(msg.Text, "/") {
			ctx := context.Background()
			// Проверяем, не создается ли событие
			if state := h.getCreatingEventState(msg.ChatID); state != nil {
				h.handleAdminCreateEventStep(ctx, msg, state)
				return
			}
			// Иначе создаем локацию
			h.handleAdminCreateLocation(ctx, msg, msg.Text)
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
	case "events":
		h.handleEvents(ctx, cb)
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
		} else if strings.HasPrefix(cb.Data, "event:") {
			// Обработка callback'ов для событий
			if strings.HasPrefix(cb.Data, "event:register:") {
				h.handleEventRegistration(ctx, cb)
			} else if strings.HasPrefix(cb.Data, "event:unregister:") {
				h.handleEventUnregister(ctx, cb)
			} else if strings.HasPrefix(cb.Data, "event:") {
				// Простой выбор события (формат: event:{id})
				h.handleEventSelection(ctx, cb)
			}
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

// getCreatingEventState возвращает состояние создания события для чата
func (h *Handlers) getCreatingEventState(chatID int64) *EventCreationState {
	return h.creatingEvents[chatID]
}

// isCreatingEvent проверяет, создается ли сейчас событие для данного чата
func (h *Handlers) isCreatingEvent(chatID int64) bool {
	return h.creatingEvents[chatID] != nil
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
