package redis

import (
	"log"
	"os"
	"strconv"

	"github.com/go-redis/redis/v8"
	"github.com/joho/godotenv"
)

func init() {
	if err := godotenv.Load(); err != nil {
		log.Println("Не найден .env, используем переменные окружения")
	}
}

// Client обертка над redis.Client для удобства
type Client struct {
	*redis.Client
}

// NewClient создает новый клиент Redis
func NewClient() *Client {
	return &Client{
		Client: redis.NewClient(&redis.Options{
			Addr:     os.Getenv("REDIS_URL"),
			Password: os.Getenv("REDIS_PASSWORD"),
			DB: func() int {
				v, err := strconv.Atoi(os.Getenv("REDIS_DB"))
				if err != nil {
					return 0
				}
				return v
			}(),
		}),
	}
}
