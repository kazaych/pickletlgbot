package main

import (
	"kitchenBot/api/telegram"
	"kitchenBot/domain/event"
	"kitchenBot/domain/location"
	"kitchenBot/storage/adapters/redis"
	"log"
	"os"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
)

func main() {
	// Загружаем .env автоматически
	if err := godotenv.Load(); err != nil {
		log.Println("Не найден .env, используем переменные окружения")
	}

	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		log.Fatal("❌ TELEGRAM_BOT_TOKEN не найден в .env или окружении")
	}

	// Инициализация Telegram Bot API
	tgBot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("✅ Бот @%s запущен", tgBot.Self.UserName)

	// Инициализация Storage (Redis)
	redisClient := redis.NewClient()
	defer redisClient.Close()

	// Инициализация репозиториев
	locationRepo := redis.NewLocationRepository(redisClient)
	// TODO: eventRepo := redis.NewEventRepository(redisClient) когда будет реализован

	// Инициализация доменных сервисов (бизнес-логика)
	locationService := location.NewService(locationRepo)
	// TODO: eventService := event.NewService(eventRepo) когда будет реализован
	var eventService *event.Service = nil // Временно nil, пока не реализован EventRepository

	// Инициализация API слоя (Telegram)
	tgClient := telegram.NewClient(tgBot)
	handlers := telegram.NewHandlers(locationService, eventService, tgClient)

	// Получаем канал обновлений
	updates := tgClient.GetUpdatesChan()

	// Обрабатываем обновления
	for update := range updates {
		handlers.HandleUpdate(update)
	}
}
