package main

import (
	"log"
	"os"
	"pickletlgbot/api/telegram"
	"pickletlgbot/internal/domain/event"
	"pickletlgbot/internal/domain/location"
	"pickletlgbot/internal/models"
	"pickletlgbot/repositories/postgres"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
	postgresdriver "gorm.io/driver/postgres"
	"gorm.io/gorm"
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

	// Инициализация PostgreSQL через GORM
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("❌ DATABASE_URL не найден в .env или окружении")
	}

	db, err := gorm.Open(postgresdriver.Open(dbURL), &gorm.Config{})
	if err != nil {
		log.Fatalf("❌ Ошибка подключения к PostgreSQL: %v", err)
	}

	// Автомиграция моделей
	if err := db.AutoMigrate(&models.LocationGORM{}, &models.EventGORM{}, &models.EventRegistrationGORM{}); err != nil {
		log.Fatalf("❌ Ошибка миграции: %v", err)
	}

	log.Println("✅ Подключение к PostgreSQL установлено")

	// Инициализация репозиториев
	locationRepo := postgres.NewLocationRepository(db)
	eventRepo := postgres.NewEventRepository(db)

	// Инициализация доменных сервисов (бизнес-логика)
	locationService := location.NewService(locationRepo)
	eventService := event.NewEventService(eventRepo, locationService)

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
