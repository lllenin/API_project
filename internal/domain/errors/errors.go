package errors

import "errors"

var (
	ErrUserNotFound       = errors.New("пользователь не найден")
	ErrBookNotFound       = errors.New("книга не найдена")
	ErrInvalidCredentials = errors.New("неверные учетные данные")
	ErrUserAlreadyExists  = errors.New("пользователь уже существует")
	ErrInvalidInput       = errors.New("некорректные входные данные")
	ErrDatabaseConnection = errors.New("ошибка соединения с базой данных")
	ErrValidationFailed   = errors.New("ошибка валидации")
	ErrUnauthorized       = errors.New("нет доступа")
	ErrForbidden          = errors.New("доступ запрещён")
	ErrInternalServer     = errors.New("внутренняя ошибка сервера")
	ErrBadRequest         = errors.New("неверный запрос")
	ErrNotFound           = errors.New("ресурс не найден")
	ErrConflict           = errors.New("конфликт ресурса")

	ErrInvalidUsername    = errors.New("некорректное имя пользователя")
	ErrInvalidEmail       = errors.New("некорректный email")
	ErrInvalidPassword    = errors.New("некорректный пароль")
	ErrInvalidRole        = errors.New("недопустимая роль пользователя")
	ErrInvalidStatus      = errors.New("недопустимый статус задачи")
	ErrInvalidTitle       = errors.New("некорректный заголовок задачи")
	ErrInvalidDescription = errors.New("некорректное описание задачи")
)
