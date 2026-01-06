# Предложение по структуре событий (Events)

## Текущие проблемы

1. **Несоответствие типов**: `LocationID` использует `uuid.UUID`, но `Location` теперь использует `LocationID` (string)
2. **Дублирование данных**: `LocationName` хранится в Event, хотя можно получить из Location
3. **Много параметров**: `CreateEvent` принимает много параметров, лучше использовать DTO
4. **Неправильный тип для Players**: хранится как `[]uuid.UUID`, но пользователи Telegram имеют `int64` ID
5. **Нет валидации**: отсутствуют бизнес-правила и валидация

## Предлагаемая структура

### 1. Entity (entity.go)

```go
package event

import (
	"errors"
	"time"
	"pickletlgbot/internal/domain/location"
)

// EventID - тип для ID события (аналогично LocationID)
type EventID string

// RegistrationStatus - статус регистрации пользователя
type RegistrationStatus string

const (
	RegistrationStatusPending  RegistrationStatus = "pending"  // Ожидает подтверждения
	RegistrationStatusApproved RegistrationStatus = "approved"  // Подтвержден
	RegistrationStatusRejected RegistrationStatus = "rejected" // Отклонен
)

// EventRegistration - регистрация пользователя на событие
type EventRegistration struct {
	UserID    int64
	Status    RegistrationStatus
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Event представляет событие в доменной модели
type Event struct {
	ID          EventID
	Name        string
	Type        EventType
	Date        time.Time
	Remaining   int        // Количество оставшихся мест
	MaxPlayers  int        // Максимальное количество игроков
	Players     []int64    // ID подтвержденных пользователей Telegram
	Registrations map[int64]EventRegistration // Все регистрации (pending + approved + rejected)
	LocationID  location.LocationID
	Description string     // Описание события (опционально)
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type EventType string

const (
	EventTypeTraining    EventType = "training"
	EventTypeCompetition EventType = "competition"
)

// CreateEventInput - DTO для создания события (аналогично CreateLocationInput)
type CreateEventInput struct {
	Name        string
	Type        EventType
	Date        time.Time
	MaxPlayers  int
	LocationID  location.LocationID
	Description string
}

// UpdateEventInput - DTO для обновления события
type UpdateEventInput struct {
	Name        *string
	Type        *EventType
	Date        *time.Time
	MaxPlayers  *int
	Remaining   *int
	Description *string
}

// Validate проверяет валидность входных данных для создания события
func (in CreateEventInput) Validate() error {
	if in.Name == "" {
		return ErrEventNameRequired
	}
	if in.LocationID == "" {
		return ErrLocationIDRequired
	}
	if in.Date.IsZero() {
		return ErrDateRequired
	}
	if in.Date.Before(time.Now()) {
		return ErrDateInPast
	}
	if in.MaxPlayers <= 0 {
		return ErrMaxPlayersInvalid
	}
	return nil
}

// Errors
var (
	ErrEventNameRequired      = errors.New("event name is required")
	ErrLocationIDRequired     = errors.New("location ID is required")
	ErrDateRequired           = errors.New("event date is required")
	ErrDateInPast             = errors.New("event date cannot be in the past")
	ErrMaxPlayersInvalid      = errors.New("max players must be greater than 0")
	ErrEventNotFound          = errors.New("event not found")
	ErrEventFull              = errors.New("event is full")
	ErrUserAlreadyRegistered  = errors.New("user is already registered for this event")
	ErrRegistrationNotFound   = errors.New("registration not found")
	ErrRegistrationAlreadyApproved = errors.New("registration already approved")
	ErrRegistrationAlreadyRejected = errors.New("registration already rejected")
)
```

### 2. Service (service.go)

```go
package event

import (
	"context"
	"errors"
	"time"
	"github.com/google/uuid"
	"pickletlgbot/internal/domain/location"
)

// EventService описывает use-case'ы вокруг событий (аналогично LocationService)
type EventService interface {
	Get(ctx context.Context, id EventID) (*Event, error)
	List(ctx context.Context) ([]Event, error)
	ListByLocation(ctx context.Context, locationID location.LocationID) ([]Event, error)
	ListByUser(ctx context.Context, userID int64) ([]Event, error)
	Create(ctx context.Context, input CreateEventInput) (*Event, error)
	Update(ctx context.Context, id EventID, input UpdateEventInput) (*Event, error)
	Delete(ctx context.Context, id EventID) error
	
	// Регистрация пользователей
	RegisterUser(ctx context.Context, eventID EventID, userID int64) error // Создает регистрацию со статусом pending
	UnregisterUser(ctx context.Context, eventID EventID, userID int64) error
	
	// Модерация регистраций (для админов)
	ApproveRegistration(ctx context.Context, eventID EventID, userID int64) error
	RejectRegistration(ctx context.Context, eventID EventID, userID int64) error
	ListPendingRegistrations(ctx context.Context, eventID EventID) ([]EventRegistration, error)
}

type eventService struct {
	repo            EventRepository
	locationService location.LocationService // Для валидации локации
}

func NewEventService(repo EventRepository, locationService location.LocationService) EventService {
	return &eventService{
		repo:            repo,
		locationService: locationService,
	}
}

func (s *eventService) Create(ctx context.Context, in CreateEventInput) (*Event, error) {
	// Валидация входных данных
	if err := in.Validate(); err != nil {
		return nil, err
	}

	// Проверяем, что локация существует
	_, err := s.locationService.Get(ctx, in.LocationID)
	if err != nil {
		return nil, errors.New("location not found")
	}

	// Создаем событие
	event := &Event{
		ID:            EventID(uuid.New().String()),
		Name:          in.Name,
		Type:          in.Type,
		Date:          in.Date,
		MaxPlayers:    in.MaxPlayers,
		Remaining:     in.MaxPlayers, // Изначально все места свободны
		Players:       []int64{},
		Registrations: make(map[int64]EventRegistration),
		LocationID:    in.LocationID,
		Description:   in.Description,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if err := s.repo.Save(ctx, event); err != nil {
		return nil, err
	}

	return event, nil
}

func (s *eventService) RegisterUser(ctx context.Context, eventID EventID, userID int64) error {
	event, err := s.repo.GetByID(ctx, eventID)
	if err != nil {
		return err
	}

	// Проверяем, не зарегистрирован ли уже пользователь (в любом статусе)
	if reg, exists := event.Registrations[userID]; exists {
		if reg.Status == RegistrationStatusPending {
			return ErrUserAlreadyRegistered // Уже есть pending регистрация
		}
		if reg.Status == RegistrationStatusApproved {
			return ErrUserAlreadyRegistered // Уже подтвержден
		}
		// Если был rejected, можно зарегистрироваться снова
	}

	// Создаем регистрацию со статусом pending
	event.Registrations[userID] = EventRegistration{
		UserID:    userID,
		Status:    RegistrationStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	event.UpdatedAt = time.Now()

	return s.repo.Save(ctx, event)
}

func (s *eventService) ApproveRegistration(ctx context.Context, eventID EventID, userID int64) error {
	event, err := s.repo.GetByID(ctx, eventID)
	if err != nil {
		return err
	}

	// Проверяем, существует ли регистрация
	reg, exists := event.Registrations[userID]
	if !exists {
		return ErrRegistrationNotFound
	}

	// Проверяем статус
	if reg.Status == RegistrationStatusApproved {
		return ErrRegistrationAlreadyApproved
	}
	if reg.Status == RegistrationStatusRejected {
		return errors.New("cannot approve rejected registration")
	}

	// Проверяем, есть ли свободные места
	if event.Remaining <= 0 {
		return ErrEventFull
	}

	// Обновляем статус регистрации
	reg.Status = RegistrationStatusApproved
	reg.UpdatedAt = time.Now()
	event.Registrations[userID] = reg

	// Добавляем пользователя в список подтвержденных игроков
	event.Players = append(event.Players, userID)
	event.Remaining--
	event.UpdatedAt = time.Now()

	return s.repo.Save(ctx, event)
}

func (s *eventService) RejectRegistration(ctx context.Context, eventID EventID, userID int64) error {
	event, err := s.repo.GetByID(ctx, eventID)
	if err != nil {
		return err
	}

	// Проверяем, существует ли регистрация
	reg, exists := event.Registrations[userID]
	if !exists {
		return ErrRegistrationNotFound
	}

	// Проверяем статус
	if reg.Status == RegistrationStatusRejected {
		return ErrRegistrationAlreadyRejected
	}
	if reg.Status == RegistrationStatusApproved {
		// Если был подтвержден, нужно убрать из списка игроков
		for i, playerID := range event.Players {
			if playerID == userID {
				event.Players = append(event.Players[:i], event.Players[i+1:]...)
				event.Remaining++
				break
			}
		}
	}

	// Обновляем статус регистрации
	reg.Status = RegistrationStatusRejected
	reg.UpdatedAt = time.Now()
	event.Registrations[userID] = reg
	event.UpdatedAt = time.Now()

	return s.repo.Save(ctx, event)
}

func (s *eventService) ListPendingRegistrations(ctx context.Context, eventID EventID) ([]EventRegistration, error) {
	event, err := s.repo.GetByID(ctx, eventID)
	if err != nil {
		return nil, err
	}

	var pending []EventRegistration
	for _, reg := range event.Registrations {
		if reg.Status == RegistrationStatusPending {
			pending = append(pending, reg)
		}
	}

	return pending, nil
}
```

### 3. Repository (repository.go)

```go
package event

import (
	"context"
	"pickletlgbot/internal/domain/location"
)

// EventRepository описывает, что нужно домену от хранилища событий
type EventRepository interface {
	GetByID(ctx context.Context, id EventID) (*Event, error)
	List(ctx context.Context) ([]Event, error)
	ListByLocation(ctx context.Context, locationID location.LocationID) ([]Event, error)
	ListByUser(ctx context.Context, userID int64) ([]Event, error)
	Save(ctx context.Context, event *Event) error
	Delete(ctx context.Context, id EventID) error
}
```

## Преимущества предложенной структуры

1. **Консистентность**: Использует тот же паттерн, что и Location (DTO, интерфейсы)
2. **Правильные типы**: `LocationID` как string, `Players` как `[]int64` (Telegram user IDs)
3. **Валидация**: Встроенная валидация входных данных
4. **Бизнес-логика**: Автоматический расчет `Remaining` при создании/обновлении
5. **Расширяемость**: Легко добавить новые поля (Description, CreatedAt, UpdatedAt)
6. **Безопасность**: Проверка существования локации перед созданием события

## Процесс создания события в Telegram

1. Админ выбирает локацию → сохраняется `locationID`
2. Админ вводит название → сохраняется `name`
3. Админ вводит дату → сохраняется `date`
4. Админ вводит количество мест → сохраняется `maxPlayers`
5. Создается событие через `CreateEventInput` с валидацией

## Механизм модерации регистраций

### Процесс регистрации:

```
┌─────────────┐
│  Пользователь│
└──────┬──────┘
       │
       │ 1. Нажимает "Записаться"
       ▼
┌─────────────────────────────┐
│ RegisterUser(eventID, userID)│
│ Создает регистрацию:        │
│ Status: pending             │
└──────┬──────────────────────┘
       │
       │ 2. Регистрация создана
       ▼
┌─────────────────────────────┐
│ Event.Registrations[userID] │
│ = {Status: pending}          │
└──────┬──────────────────────┘
       │
       │ 3. Админ видит список
       ▼
┌─────────────────────────────┐
│ ListPendingRegistrations()  │
│ Возвращает все pending      │
└──────┬──────────────────────┘
       │
       │ 4. Админ выбирает действие
       ├─────────────────┬─────────────────┐
       ▼                 ▼                 ▼
┌──────────────┐  ┌──────────────┐  ┌──────────────┐
│  Approve     │  │   Reject     │  │   Игнорировать│
└──────┬───────┘  └──────┬───────┘  └──────────────┘
       │                 │
       │ 5a. Approved     │ 5b. Rejected
       ▼                 ▼
┌─────────────────┐  ┌─────────────────┐
│ Status: approved│  │ Status: rejected│
│ + Players[]     │  │ (можно          │
│ + Remaining--   │  │  зарегистрироваться│
└─────────────────┘  │  снова)          │
                     └─────────────────┘
```

### Статусы регистрации:

- **`pending`** - ожидает подтверждения админа
- **`approved`** - подтверждена, пользователь в списке игроков
- **`rejected`** - отклонена, пользователь может зарегистрироваться снова

### Структура данных:

- `Registrations map[int64]EventRegistration` - все регистрации (pending, approved, rejected)
- `Players []int64` - только подтвержденные пользователи
- `Remaining int` - количество свободных мест (только для approved)

### Методы для админа:

- `ListPendingRegistrations(eventID)` - получить список ожидающих подтверждения
- `ApproveRegistration(eventID, userID)` - подтвердить регистрацию
- `RejectRegistration(eventID, userID)` - отклонить регистрацию

### UI в Telegram:

#### Для пользователя:

```
📅 Тренировка: Пиклбол
🗓️ Дата: 2024-12-31 18:00
📍 Локация: Спортзал
👥 Мест: 5/10

[📝 Записаться]
```

После нажатия:
```
✅ Ваша заявка отправлена на модерацию.
Ожидайте подтверждения администратора.
```

#### Для админа:

```
🔔 Новые заявки на модерацию

📅 Тренировка: Пиклбол
👤 Пользователь: @username (ID: 123456)
⏰ Заявка: 2 минуты назад

[✅ Подтвердить] [❌ Отклонить]
```

Или список всех pending:
```
📋 Заявки на модерацию (3)

1. @user1 - Пиклбол (5 мин назад)
2. @user2 - Тренировка (10 мин назад)
3. @user3 - Соревнование (1 час назад)

[Выберите заявку для модерации]
```

## Вопросы для обсуждения

1. Нужно ли поле `Description` для событий?
2. Нужно ли поле `Type` (training/competition) или всегда training?
3. Как обрабатывать отмену регистрации пользователя?
4. Нужна ли возможность редактировать событие после создания?
5. Как фильтровать события (только будущие, по дате, по локации)?
6. Нужны ли уведомления админу о новых регистрациях?
7. Можно ли пользователю отменить свою pending регистрацию?

