package location

type LocationID string

// Location представляет локацию в доменной модели
type Location struct {
	ID            LocationID
	Name          string
	Address       string
	Description   string
	AddressMapURL string
}
