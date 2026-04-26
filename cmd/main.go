package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"pickletlgbot/api/telegram"
	"pickletlgbot/internal/domain/event"
	"pickletlgbot/internal/domain/location"
	"pickletlgbot/internal/domain/settings"
	"pickletlgbot/internal/domain/user"
	"pickletlgbot/internal/models"
	"pickletlgbot/repositories/postgres"
	"sync"
	"syscall"
	"time"

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

	// Автомиграция моделей (порядок важен: сначала таблицы без foreign keys, потом с foreign keys)
	// Этап 1: Создаем таблицы без foreign keys
	if err := db.AutoMigrate(
		&models.UserGORM{},     // 1. user (нет зависимостей)
		&models.LocationGORM{}, // 2. locations (нет зависимостей)
		&models.EventGORM{},    // 3. events (зависит от locations, но через строку - не foreign key)
	); err != nil {
		log.Fatalf("❌ Ошибка миграции (этап 1): %v", err)
	}

	// Этап 2: Создаем таблицу с foreign keys (после того, как все остальные таблицы созданы)
	// Удаляем старый неправильный индекс, если он существует (был создан только на user_id)
	if err := db.Exec("DROP INDEX IF EXISTS idx_event_user").Error; err != nil {
		log.Printf("⚠️ Предупреждение при удалении старого индекса: %v", err)
	}

	if err := db.AutoMigrate(
		&models.EventRegistrationGORM{}, // 4. event_registrations (зависит от user и events)
		&models.SettingsGORM{},          // 5. settings (нет зависимостей)
	); err != nil {
		log.Fatalf("❌ Ошибка миграции (этап 2): %v", err)
	}

	log.Println("✅ Подключение к PostgreSQL установлено")

	// Инициализация репозиториев
	locationRepo := postgres.NewLocationRepository(db)
	eventRepo := postgres.NewEventRepository(db)
	userRepo := postgres.NewUserRepository(db)
	settingsRepo := postgres.NewSettingsRepository(db)

	// Инициализация доменных сервисов (бизнес-логика)
	locationService := location.NewService(locationRepo)
	userService := user.NewPlayerService(userRepo)
	eventService := event.NewEventService(eventRepo, locationService)
	settingsService := settings.NewService(settingsRepo)

	// Инициализация API слоя (Telegram)
	tgClient := telegram.NewClient(tgBot)
	handlers := telegram.NewHandlers(locationService, eventService, userService, settingsService, tgClient)

	// Получаем канал обновлений
	updates := tgClient.GetUpdatesChan()

	// Создаем контекст для graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// WaitGroup для отслеживания активных горутин
	var wg sync.WaitGroup

	// Канал для сигналов завершения
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Горутина для обработки сигналов завершения
	go func() {
		<-sigChan
		log.Println("🛑 Получен сигнал завершения, ожидаем завершения обработки обновлений...")
		cancel() // Отменяем контекст

		// Даем время на завершение активных обработок (максимум 30 секунд)
		done := make(chan struct{})
		go func() {
			wg.Wait()
			close(done)
		}()

		select {
		case <-done:
			log.Println("✅ Все обновления обработаны, завершаем работу")
		case <-time.After(30 * time.Second):
			log.Println("⚠️  Таймаут ожидания, принудительное завершение")
		}
		os.Exit(0)
	}()

	// Обрабатываем обновления
	for {
		select {
		case update, ok := <-updates:
			if !ok {
				log.Println("Канал обновлений закрыт")
				return
			}
			wg.Add(1) // Увеличиваем счетчик активных горутин
			go func(u *telegram.Update) {
				defer wg.Done() // Уменьшаем счетчик при завершении
				handlers.HandleUpdate(u)
			}(update)
		case <-ctx.Done():
			log.Println("Контекст отменен, прекращаем обработку новых обновлений")
			// Ждем завершения всех активных обработок
			wg.Wait()
			return
		}
	}
}
