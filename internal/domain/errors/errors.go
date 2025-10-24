package errors

import "errors"

var (
	ErrUserNotFound       = errors.New("пользователь не найден")
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

	ErrInvalidRequest = errors.New("некорректные данные запроса")
	ErrUserExists     = errors.New("пользователь уже существует")
	ErrTaskStatus     = errors.New("недопустимый статус задачи")

	ErrInvalidRequestData     = errors.New("некорректные данные запроса")
	ErrInvalidUserCredentials = errors.New("неверные учетные данные")
	ErrUnauthorizedAction     = errors.New("нет прав на выполнение действия")
	ErrUserUpdateForbidden    = errors.New("нет прав на изменение этого пользователя")
	ErrUserDeleteForbidden    = errors.New("нет прав на удаление этого пользователя")
	ErrTaskNotFound           = errors.New("задача не найдена")
	ErrTasksNotFound          = errors.New("задачи не найдены")
	ErrTokenGeneration        = errors.New("ошибка генерации токена")
	ErrNotAuthorized          = errors.New("пользователь не авторизован")

	ErrInvalidGzipRequest    = errors.New("некорректный gzip-запрос")
	ErrGzipCompressionFailed = errors.New("ошибка gzip-сжатия")

	ErrConfigFileNotFound   = errors.New("файл конфигурации не найден")
	ErrConfigFileReadFailed = errors.New("ошибка чтения файла конфигурации")
	ErrConfigParseFailed    = errors.New("ошибка парсинга конфигурации")
	ErrConfigInvalidFormat  = errors.New("неверный формат конфигурации")
)
