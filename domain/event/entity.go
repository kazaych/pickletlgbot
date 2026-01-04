package event

import "time"

// Event представляет событие в доменной модели
type Event struct {
	ID         string
	LocationID string
	Name       string
	Date       time.Time
	Remaining  int
}
