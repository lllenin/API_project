package server

import (
	"context"
	"net/http"
	"project/internal/domain/errors"
	"project/internal/domain/models"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator"
	"github.com/google/uuid"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var jwtSecret = []byte("shouldbeinVaultsecret")

func generateJWT(userID string) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

func getUserIDFromJWT(ctx *gin.Context) (string, error) {
	cookie, err := ctx.Cookie("jwt_token")
	if err != nil {
		return "", errors.ErrUnauthorized
	}
	token, err := jwt.Parse(cookie, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})
	if err != nil || !token.Valid {
		return "", errors.ErrUnauthorized
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", errors.ErrUnauthorized
	}
	userID, ok := claims["user_id"].(string)
	if !ok || userID == "" {
		return "", errors.ErrUnauthorized
	}
	return userID, nil
}

// TaskRepository определяет интерфейс для работы с задачами в хранилище.
// Все методы принимают контекст для управления таймаутами и отменой операций.
type TaskRepository interface {
	// CreateTask создает новую задачу в хранилище.
	CreateTask(ctx context.Context, task *models.Task) error
	// GetTaskByID возвращает задачу по её идентификатору.
	GetTaskByID(ctx context.Context, id string) (*models.Task, error)
	// GetTasks возвращает список всех задач для указанного пользователя.
	GetTasks(ctx context.Context, userID string) ([]models.Task, error)
	// UpdateTask обновляет существующую задачу по её идентификатору.
	UpdateTask(ctx context.Context, id string, task *models.Task) error
	// DeleteTask удаляет задачу по её идентификатору.
	DeleteTask(ctx context.Context, id string) error
}

// Repository определяет интерфейс для работы с пользователями в хранилище.
type Repository interface {
	// GetUserByID возвращает пользователя по его идентификатору.
	GetUserByID(id string) (*models.User, error)
	// GetUserByUsername возвращает пользователя по его имени пользователя.
	GetUserByUsername(username string) (*models.User, error)
	// UpdateUser обновляет существующего пользователя по его идентификатору.
	UpdateUser(id string, user *models.User) error
	// DeleteUser удаляет пользователя по его идентификатору.
	DeleteUser(id string) error
	// CreateUser создает нового пользователя в хранилище.
	CreateUser(user *models.User) error
}

// TaskAPI представляет основной API сервер для работы с задачами и пользователями.
// Содержит HTTP сервер, репозитории для пользователей и задач.
type TaskAPI struct {
	httpSrv  *http.Server
	repo     Repository
	taskRepo TaskRepository
	cfg      *Config
}

// NewTaskAPI создает новый экземпляр TaskAPI с указанными репозиториями и конфигурацией.
// Возвращает nil, если repo или taskRepo равны nil.
// Автоматически настраивает маршруты HTTP сервера.
func NewTaskAPI(repo Repository, taskRepo TaskRepository, cfg *Config) *TaskAPI {
	if repo == nil || taskRepo == nil {
		return nil
	}

	httpSrv := http.Server{
		Addr:              cfg.Addr + ":" + strconv.Itoa(cfg.Port),
		ReadHeaderTimeout: 30 * time.Second,
	}

	api := TaskAPI{
		httpSrv:  &httpSrv,
		repo:     repo,
		taskRepo: taskRepo,
		cfg:      cfg,
	}

	api.configRoutes()

	return &api
}

// Start запускает HTTP сервер и начинает прослушивание входящих соединений.
// Если включен HTTPS (флаг -s или переменная окружения ENABLE_HTTPS), использует ListenAndServeTLS.
// При включенном HTTPS сервер работает только через TLS для всего сайта.
// Возвращает ошибку, если сервер не был инициализирован или произошла ошибка при запуске.
func (api *TaskAPI) Start() error {
	if api.httpSrv == nil {
		return errors.ErrInternalServer
	}

	if api.httpSrv.Addr == "" {
		api.httpSrv.Addr = ":8080"
	}

	if api.cfg != nil && api.cfg.EnableHTTPS {
		certFile := api.cfg.CertFile
		keyFile := api.cfg.KeyFile
		if certFile == "" {
			certFile = "server.crt"
		}
		if keyFile == "" {
			keyFile = "server.key"
		}
		return api.httpSrv.ListenAndServeTLS(certFile, keyFile)
	}

	return api.httpSrv.ListenAndServe()
}

// Shutdown выполняет graceful shutdown HTTP сервера.
// Использует переданный контекст для управления таймаутом завершения.
// Возвращает ошибку, если произошла ошибка при завершении работы сервера.
func (api *TaskAPI) Shutdown(ctx context.Context) error {
	if api.httpSrv == nil {
		return nil
	}
	return api.httpSrv.Shutdown(ctx)
}

func (api *TaskAPI) configRoutes() {
	router := gin.Default()

	router.Use(func(ctx *gin.Context) {
		origin := ctx.GetHeader("Origin")

		if origin != "" {
			ctx.Header("Access-Control-Allow-Origin", origin)
			ctx.Header("Access-Control-Allow-Credentials", "true")
		} else {
			ctx.Header("Access-Control-Allow-Origin", "*")
		}

		ctx.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		ctx.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With, Accept, Origin")
		ctx.Header("Access-Control-Expose-Headers", "Content-Length, Content-Type")
		ctx.Header("Access-Control-Max-Age", "3600")

		if ctx.Request.Method == "OPTIONS" {
			ctx.AbortWithStatus(http.StatusNoContent)
			return
		}

		ctx.Next()
	})

	if api.cfg != nil && api.cfg.EnableHTTPS {
		router.Use(func(ctx *gin.Context) {
			if ctx.Request.TLS == nil {
				ctx.JSON(http.StatusBadRequest, gin.H{"error": "требуется HTTPS соединение. Используйте https:// вместо http://"})
				ctx.Abort()
				return
			}
			ctx.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
			ctx.Header("X-Content-Type-Options", "nosniff")
			ctx.Header("X-Frame-Options", "DENY")
			ctx.Header("X-XSS-Protection", "1; mode=block")
			ctx.Header("Referrer-Policy", "strict-origin-when-cross-origin")
			ctx.Next()
		})
	}

	router.NoMethod(func(ctx *gin.Context) {
		ctx.JSON(http.StatusMethodNotAllowed, gin.H{"error": "использован некорректный HTTP-метод"})
	})

	user := router.Group("/users")
	{
		user.POST("/login", api.login)
		user.POST("/register", api.register)
		user.PUT("/update/:userID", api.updateUser)
		user.DELETE("/delete/:userID", api.deleteUser)
		user.GET("/login", func(ctx *gin.Context) {
			ctx.JSON(http.StatusMethodNotAllowed, gin.H{"error": "использован некорректный HTTP-метод"})
		})
		user.GET("/register", func(ctx *gin.Context) {
			ctx.JSON(http.StatusMethodNotAllowed, gin.H{"error": "использован некорректный HTTP-метод"})
		})
		user.GET("/:userID", api.getUser)
	}

	tasks := router.Group("/tasks")
	{
		tasks.GET("", api.getTasks)
		tasks.GET("/:taskID", api.getTaskByID)
		tasks.POST("", api.createTask)
		tasks.PUT("/:taskID", api.updateTask)
		tasks.DELETE("/:taskID", api.deleteTask)
	}

	api.httpSrv.Handler = router
}

// login обрабатывает запрос на вход пользователя.
// Принимает логин и пароль, проверяет их и устанавливает JWT токен в cookie.
func (api *TaskAPI) login(ctx *gin.Context) {
	var req models.LoginRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": errors.ErrInvalidRequest.Error()})
		return
	}

	valid := validator.New()
	if err := valid.Struct(req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": errors.ErrValidationFailed.Error()})
		return
	}

	user, err := api.repo.GetUserByUsername(req.Username)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": errors.ErrInvalidUserCredentials.Error()})
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password))
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": errors.ErrInvalidUserCredentials.Error()})
		return
	}

	token, err := generateJWT(user.ID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": errors.ErrTokenGeneration.Error()})
		return
	}
	http.SetCookie(ctx.Writer, &http.Cookie{
		Name:     "jwt_token",
		Value:    token,
		Path:     "/",
		MaxAge:   3600,
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteStrictMode,
	})

	ctx.JSON(http.StatusOK, gin.H{
		"message": "вход выполнен успешно",
		"user": gin.H{
			"id":       user.ID,
			"username": user.Username,
			"email":    user.Email,
			"role":     user.Role,
		},
	})
}

// register обрабатывает запрос на регистрацию нового пользователя.
// Создает пользователя с хешированным паролем и возвращает информацию о созданном пользователе.
func (api *TaskAPI) register(ctx *gin.Context) {
	var req models.RegisterRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": errors.ErrInvalidRequest.Error()})
		return
	}
	if req.Role != "" && !allowedUserRoles[req.Role] {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": errors.ErrInvalidRole.Error()})
		return
	}
	valid := validator.New()

	if err := valid.Struct(req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": errors.ErrInvalidRequest.Error()})
		return
	}

	existingUser, _ := api.repo.GetUserByUsername(req.Username)
	if existingUser != nil {
		ctx.JSON(http.StatusConflict, gin.H{"error": errors.ErrUserExists.Error()})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": errors.ErrInternalServer.Error()})
		return
	}
	role := req.Role
	if role == "" {
		role = "user"
	}
	user := models.User{
		ID:       uuid.New().String(),
		Username: req.Username,
		Email:    req.Email,
		Password: string(hash),
		Role:     role,
	}

	if err := api.repo.CreateUser(&user); err != nil {
		ctx.JSON(http.StatusConflict, gin.H{"error": errors.ErrUserAlreadyExists.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{
		"message": "пользователь успешно создан",
		"user": gin.H{
			"id":       user.ID,
			"username": user.Username,
			"email":    user.Email,
			"role":     user.Role,
		},
	})
}

// getUser обрабатывает запрос на получение информации о пользователе по его ID.
func (api *TaskAPI) getUser(ctx *gin.Context) {
	userID := ctx.Param("userID")

	user, err := api.repo.GetUserByID(userID)
	if err != nil {
		if err == errors.ErrUserNotFound {
			ctx.JSON(http.StatusNotFound, gin.H{"error": errors.ErrUserNotFound.Error()})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": errors.ErrInternalServer.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"user": gin.H{
			"id":       user.ID,
			"username": user.Username,
			"email":    user.Email,
			"role":     user.Role,
		},
	})
}

// updateUser обрабатывает запрос на обновление информации о пользователе.
// Требует аутентификации и проверяет, что пользователь обновляет только свои данные.
func (api *TaskAPI) updateUser(ctx *gin.Context) {
	userID, err := getUserIDFromJWT(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": errors.ErrNotAuthorized.Error()})
		return
	}
	userIDParam := ctx.Param("userID")
	if userID != userIDParam {
		ctx.JSON(http.StatusForbidden, gin.H{"error": errors.ErrUserUpdateForbidden.Error()})
		return
	}
	var req models.UpdateUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": errors.ErrInvalidRequest.Error()})
		return
	}
	if req.Role != "" && !allowedUserRoles[req.Role] {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": errors.ErrInvalidRole.Error()})
		return
	}

	user := &models.User{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password,
		Role:     req.Role,
	}

	if err := api.repo.UpdateUser(userID, user); err != nil {
		if err == errors.ErrUserNotFound {
			ctx.JSON(http.StatusNotFound, gin.H{"error": errors.ErrUserNotFound.Error()})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": errors.ErrInternalServer.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "пользователь успешно обновлен"})
}

// deleteUser обрабатывает запрос на удаление пользователя.
// Требует аутентификации и проверяет, что пользователь удаляет только свой аккаунт.
func (api *TaskAPI) deleteUser(ctx *gin.Context) {
	userID, err := getUserIDFromJWT(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": errors.ErrNotAuthorized.Error()})
		return
	}
	userIDParam := ctx.Param("userID")
	if userID != userIDParam {
		ctx.JSON(http.StatusForbidden, gin.H{"error": errors.ErrUserDeleteForbidden.Error()})
		return
	}
	if err := api.repo.DeleteUser(userID); err != nil {
		if err == errors.ErrUserNotFound {
			ctx.JSON(http.StatusNotFound, gin.H{"error": errors.ErrUserNotFound.Error()})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": errors.ErrInternalServer.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "пользователь успешно удален"})
}

// getTasks обрабатывает запрос на получение списка всех задач текущего пользователя.
// Требует аутентификации.
func (api *TaskAPI) getTasks(ctx *gin.Context) {
	userID, err := getUserIDFromJWT(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": errors.ErrNotAuthorized.Error()})
		return
	}
	tasks, err := api.taskRepo.GetTasks(ctx.Request.Context(), userID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": errors.ErrInternalServer.Error()})
		return
	}
	if len(tasks) == 0 {
		ctx.JSON(http.StatusNotFound, gin.H{"error": errors.ErrTasksNotFound.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"tasks": tasks})
}

// getTaskByID обрабатывает запрос на получение задачи по её ID.
// Требует аутентификации и проверяет, что задача принадлежит текущему пользователю.
func (api *TaskAPI) getTaskByID(ctx *gin.Context) {
	userID, err := getUserIDFromJWT(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": errors.ErrNotAuthorized.Error()})
		return
	}
	id := ctx.Param("taskID")
	task, err := api.taskRepo.GetTaskByID(ctx.Request.Context(), id)
	if err != nil {
		if err == errors.ErrNotFound {
			ctx.JSON(http.StatusNotFound, gin.H{"error": errors.ErrTaskNotFound.Error()})
		} else {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": errors.ErrInternalServer.Error()})
		}
		return
	}
	if task.UserID != userID {
		ctx.JSON(http.StatusForbidden, gin.H{"error": errors.ErrForbidden.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"task": task})
}

var allowedTaskStatuses = map[string]bool{
	"new":         true,
	"in_progress": true,
	"done":        true,
}

var allowedUserRoles = map[string]bool{
	"user":      true,
	"admin":     true,
	"moderator": true,
}

// createTask обрабатывает запрос на создание новой задачи.
// Требует аутентификации. Созданная задача автоматически привязывается к текущему пользователю.
func (api *TaskAPI) createTask(ctx *gin.Context) {
	userID, err := getUserIDFromJWT(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": errors.ErrNotAuthorized.Error()})
		return
	}
	var req models.CreateTaskRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": errors.ErrBadRequest.Error()})
		return
	}
	valid := validator.New()
	if err := valid.Struct(req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": errors.ErrInvalidRequest.Error()})
		return
	}
	task := models.Task{
		Title:       req.Title,
		Description: req.Description,
		Status:      "new",
		UserID:      userID,
	}
	if err := api.taskRepo.CreateTask(ctx.Request.Context(), &task); err != nil {
		if err == errors.ErrConflict {
			ctx.JSON(http.StatusConflict, gin.H{"error": errors.ErrConflict.Error()})
		} else {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": errors.ErrInternalServer.Error()})
		}
		return
	}
	ctx.JSON(http.StatusCreated, gin.H{"task": task})
}

// updateTask обрабатывает запрос на обновление существующей задачи.
// Требует аутентификации и проверяет, что задача принадлежит текущему пользователю.
func (api *TaskAPI) updateTask(ctx *gin.Context) {
	userID, err := getUserIDFromJWT(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": errors.ErrNotAuthorized.Error()})
		return
	}
	id := ctx.Param("taskID")
	var req models.UpdateTaskRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": errors.ErrBadRequest.Error()})
		return
	}
	valid := validator.New()
	if err := valid.Struct(req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": errors.ErrInvalidRequest.Error()})
		return
	}
	task, err := api.taskRepo.GetTaskByID(ctx.Request.Context(), id)
	if err != nil {
		if err == errors.ErrNotFound {
			ctx.JSON(http.StatusNotFound, gin.H{"error": errors.ErrTaskNotFound.Error()})
		} else {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": errors.ErrInternalServer.Error()})
		}
		return
	}
	if task.UserID != userID {
		ctx.JSON(http.StatusForbidden, gin.H{"error": errors.ErrForbidden.Error()})
		return
	}
	if req.Status != "" && !allowedTaskStatuses[req.Status] {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": errors.ErrTaskStatus.Error()})
		return
	}
	if req.Title != "" {
		task.Title = req.Title
	}
	if req.Description != "" {
		task.Description = req.Description
	}
	if req.Status != "" {
		task.Status = req.Status
	}
	if err := api.taskRepo.UpdateTask(ctx.Request.Context(), id, task); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": errors.ErrInternalServer.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"task": task})
}

// deleteTask обрабатывает запрос на удаление задачи.
// Требует аутентификации и проверяет, что задача принадлежит текущему пользователю.
// Выполняет мягкое удаление (soft delete) задачи.
func (api *TaskAPI) deleteTask(ctx *gin.Context) {
	userID, err := getUserIDFromJWT(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": errors.ErrNotAuthorized.Error()})
		return
	}
	id := ctx.Param("taskID")
	task, err := api.taskRepo.GetTaskByID(ctx.Request.Context(), id)
	if err != nil {
		if err == errors.ErrNotFound {
			ctx.JSON(http.StatusNotFound, gin.H{"error": errors.ErrTaskNotFound.Error()})
		} else {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": errors.ErrInternalServer.Error()})
		}
		return
	}
	if task.UserID != userID {
		ctx.JSON(http.StatusForbidden, gin.H{"error": errors.ErrForbidden.Error()})
		return
	}
	if err := api.taskRepo.DeleteTask(ctx.Request.Context(), id); err != nil {
		if err == errors.ErrNotFound {
			ctx.JSON(http.StatusNotFound, gin.H{"error": errors.ErrTaskNotFound.Error()})
		} else {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": errors.ErrInternalServer.Error()})
		}
		return
	}
	type hardDeleteEnqueuer interface{ EnqueueHardDelete(string) }
	if enq, ok := any(api.taskRepo).(hardDeleteEnqueuer); ok {
		enq.EnqueueHardDelete(id)
	}
	ctx.JSON(http.StatusOK, gin.H{"message": "задача успешно удалена"})
}
