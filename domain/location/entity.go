package location

import "github.com/google/uuid"

// Location представляет локацию в доменной модели
type Location struct {
	ID            uuid.UUID
	Name          string
	Address       string
	AddressMapUrl string
}
