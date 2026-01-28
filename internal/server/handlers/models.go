package handlers

// Здесь будут модели для уровня хендлеров (DTO),
// если они отличаются от gRPC generated моделей или моделей БД.

// Пример:
type LoginRequestDTO struct {
	Login    string
	Password string // или AuthKey
}
