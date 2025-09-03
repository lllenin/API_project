package errors

import "errors"

var (
    ErrUserNotFound           = errors.New("Пользователь не найден")
    ErrInvalidCredentials     = errors.New("Недействительные учетные данные")
    ErrUserAlreadyExists      = errors.New("Пользователь уже существует")
    ErrInvalidInput           = errors.New("Некорректные входные данные")
    ErrDatabaseConnection     = errors.New("Ошибка подключения к базе данных")
    ErrValidationFailed       = errors.New("Ошибка валидации данных")
    ErrUnauthorized           = errors.New("Не авторизован")
    ErrForbidden              = errors.New("Доступ запрещен")
    ErrInternalServer         = errors.New("Внутренняя ошибка сервера")
    ErrBadRequest             = errors.New("Некорректный запрос")
    ErrNotFound               = errors.New("Не найдено")
    ErrConflict               = errors.New("Конфликт данных")

    ErrInvalidUsername        = errors.New("Некорректное имя пользователя")
    ErrInvalidEmail           = errors.New("Некорректный email")
    ErrInvalidPassword        = errors.New("Некорректный пароль")
    ErrInvalidRole            = errors.New("Некорректная роль пользователя")
    ErrInvalidStatus          = errors.New("Некорректный статус задачи")
    ErrInvalidTitle           = errors.New("Некорректный заголовок")
    ErrInvalidDescription     = errors.New("Некорректное описание")

    ErrInvalidRequest         = errors.New("Некорректные данные запроса")
    ErrUserExists             = errors.New("Пользователь уже существует")
    ErrTaskStatus             = errors.New("Некорректный статус задачи")

    ErrInvalidRequestData     = errors.New("Некорректные данные запроса")
    ErrInvalidUserCredentials = errors.New("Неверное имя пользователя или пароль")
    ErrUnauthorizedAction     = errors.New("Недостаточно прав для выполнения действия")
    ErrUserUpdateForbidden    = errors.New("Нельзя обновлять данные другого пользователя")
    ErrUserDeleteForbidden    = errors.New("Нельзя удалять другого пользователя")
    ErrTaskNotFound           = errors.New("Задача не найдена")
    ErrTasksNotFound          = errors.New("Задачи не найдены")
    ErrTokenGeneration        = errors.New("Ошибка генерации токена")
    ErrNotAuthorized          = errors.New("Отсутствует авторизация")

    ErrInvalidGzipRequest     = errors.New("Некорректный gzip-запрос")
    ErrGzipCompressionFailed  = errors.New("Ошибка gzip-сжатия")
)

