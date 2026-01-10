package telegram

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Client обертка над Telegram Bot API
type Client struct {
	bot *tgbotapi.BotAPI
}

// NewClient создает новый клиент для работы с Telegram
func NewClient(bot *tgbotapi.BotAPI) *Client {
	return &Client{bot: bot}
}

// SendMessage отправляет текстовое сообщение
func (c *Client) SendMessage(chatID int64, text string) error {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeHTML // Включаем HTML форматирование
	_, err := c.bot.Send(msg)
	return err
}

// SendMessageWithKeyboard отправляет сообщение с клавиатурой
func (c *Client) SendMessageWithKeyboard(chatID int64, text string, keyboard *InlineKeyboardMarkup) error {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeHTML // Включаем HTML форматирование
	msg.ReplyMarkup = convertInlineKeyboard(keyboard)
	_, err := c.bot.Send(msg)
	return err
}

// EditMessageText редактирует текстовое сообщение
func (c *Client) EditMessageText(chatID int64, messageID int, text string) error {
	edit := tgbotapi.NewEditMessageText(chatID, messageID, text)
	_, err := c.bot.Send(edit)
	return err
}

// EditMessageTextAndMarkup редактирует сообщение с клавиатурой
func (c *Client) EditMessageTextAndMarkup(chatID int64, messageID int, text string, keyboard *InlineKeyboardMarkup) error {
	edit := tgbotapi.NewEditMessageTextAndMarkup(chatID, messageID, text, convertInlineKeyboard(keyboard))
	_, err := c.bot.Send(edit)
	return err
}

// AnswerCallbackQuery отвечает на callback query
func (c *Client) AnswerCallbackQuery(callbackQueryID string) error {
	answer := tgbotapi.NewCallback(callbackQueryID, "")
	_, err := c.bot.Request(answer)
	return err
}

// GetUpdatesChan возвращает канал обновлений
func (c *Client) GetUpdatesChan() <-chan *Update {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := c.bot.GetUpdatesChan(u)

	updateChan := make(chan *Update)
	go func() {
		for update := range updates {
			updateChan <- convertUpdate(update)
		}
		close(updateChan)
	}()

	return updateChan
}

// Update представляет обновление от Telegram
type Update struct {
	Message       *Message
	CallbackQuery *CallbackQuery
}

// Message представляет сообщение от Telegram
type Message struct {
	ChatID    int64
	MessageID int
	Text      string
	From      *User
}

// CallbackQuery представляет callback query от Telegram
type CallbackQuery struct {
	ID      string
	Data    string
	Message *Message
	From    *User
}

// User представляет пользователя Telegram
type User struct {
	ID int64
}

// InlineKeyboardMarkup представляет inline клавиатуру
type InlineKeyboardMarkup struct {
	InlineKeyboard [][]InlineKeyboardButton
}

// InlineKeyboardButton представляет кнопку inline клавиатуры
type InlineKeyboardButton struct {
	Text         string
	CallbackData string
	URL          string // URL для кнопки-ссылки
}

// Helper функции для создания клавиатур
func NewInlineKeyboardMarkup(rows ...[]InlineKeyboardButton) *InlineKeyboardMarkup {
	return &InlineKeyboardMarkup{InlineKeyboard: rows}
}

func NewInlineKeyboardRow(buttons ...InlineKeyboardButton) []InlineKeyboardButton {
	return buttons
}

func NewInlineKeyboardButtonData(text, callbackData string) InlineKeyboardButton {
	return InlineKeyboardButton{
		Text:         text,
		CallbackData: callbackData,
	}
}

func NewInlineKeyboardButtonURL(text, url string) InlineKeyboardButton {
	return InlineKeyboardButton{
		Text: text,
		URL:  url,
	}
}

// convertInlineKeyboard конвертирует нашу клавиатуру в tgbotapi формат
func convertInlineKeyboard(keyboard *InlineKeyboardMarkup) tgbotapi.InlineKeyboardMarkup {
	if keyboard == nil {
		return tgbotapi.InlineKeyboardMarkup{}
	}

	var rows [][]tgbotapi.InlineKeyboardButton
	for _, row := range keyboard.InlineKeyboard {
		var buttons []tgbotapi.InlineKeyboardButton
		for _, btn := range row {
			if btn.URL != "" {
				// Кнопка с URL
				buttons = append(buttons, tgbotapi.NewInlineKeyboardButtonURL(btn.Text, btn.URL))
			} else {
				// Кнопка с callback data
				buttons = append(buttons, tgbotapi.NewInlineKeyboardButtonData(btn.Text, btn.CallbackData))
			}
		}
		rows = append(rows, buttons)
	}
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

// convertUpdate конвертирует tgbotapi.Update в наш Update
func convertUpdate(update tgbotapi.Update) *Update {
	result := &Update{}

	if update.Message != nil {
		result.Message = &Message{
			ChatID:    update.Message.Chat.ID,
			MessageID: update.Message.MessageID,
			Text:      update.Message.Text,
			From:      convertUser(update.Message.From),
		}
	}

	if update.CallbackQuery != nil {
		result.CallbackQuery = &CallbackQuery{
			ID:   update.CallbackQuery.ID,
			Data: update.CallbackQuery.Data,
			From: convertUser(update.CallbackQuery.From),
		}
		if update.CallbackQuery.Message != nil {
			result.CallbackQuery.Message = &Message{
				ChatID:    update.CallbackQuery.Message.Chat.ID,
				MessageID: update.CallbackQuery.Message.MessageID,
				Text:      update.CallbackQuery.Message.Text,
				From:      convertUser(update.CallbackQuery.Message.From),
			}
		}
	}

	return result
}

// convertUser конвертирует tgbotapi.User в наш User
func convertUser(user *tgbotapi.User) *User {
	if user == nil {
		return nil
	}
	return &User{ID: int64(user.ID)}
}
