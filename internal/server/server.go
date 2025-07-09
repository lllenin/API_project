package server

import (
	"log"
	"net/http"
	"project/internal/domain/errors"
	"project/internal/domain/models"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator"
	"github.com/google/uuid"

	"golang.org/x/crypto/bcrypt"
)

type Repository interface {
	GetUserByID(id string) (*models.User, error)
	GetUserByUsername(username string) (*models.User, error)
	UpdateUser(id string, user *models.User) error
	DeleteUser(id string) error
	CreateUser(user *models.User) error

	GetTasks() ([]models.Task, error)
	GetTaskByID(id string) (*models.Task, error)
	CreateTask(task *models.Task) error
	UpdateTask(id string, task *models.Task) error
	DeleteTask(id string) error
}

type LibraryAPI struct {
	httpSrv *http.Server
	repo    Repository
}

func NewLibraryAPI(repo Repository) *LibraryAPI {
	if repo == nil {
		return nil
	}

	httpSrv := http.Server{
		Addr: DefaultConfig.Addr,
	}

	LAPI := LibraryAPI{
		httpSrv: &httpSrv,
		repo:    repo,
	}

	LAPI.configRoutes()

	return &LAPI
}

func (LAPI *LibraryAPI) Start() error {
	if LAPI.httpSrv == nil {
		return errors.ErrInternalServer
	}

	if LAPI.httpSrv.Addr == "" {
		LAPI.httpSrv.Addr = ":8080"
	}

	return LAPI.httpSrv.ListenAndServe()
}

func (LAPI *LibraryAPI) configRoutes() {
	router := gin.Default()

	router.NoMethod(func(ctx *gin.Context) {
		ctx.JSON(http.StatusMethodNotAllowed, gin.H{"error": "использован некорректный HTTP-метод"})
	})

	user := router.Group("/users")
	{
		user.POST("/login", LAPI.login)
		user.POST("/register", LAPI.register)
		user.PUT("/update/:userID", LAPI.updateUser)
		user.DELETE("/delete/:userID", LAPI.deleteUser)
		user.GET("/:userID", LAPI.getUser)
	}

	tasks := router.Group("/tasks")
	{
		tasks.GET("", LAPI.getTasks)
		tasks.GET(":taskID", LAPI.getTaskByID)
		tasks.POST("", LAPI.createTask)
		tasks.PUT(":taskID", LAPI.updateTask)
		tasks.DELETE(":taskID", LAPI.deleteTask)
	}

	LAPI.httpSrv.Handler = router
}

func (LAPI *LibraryAPI) login(ctx *gin.Context) {
	var req models.LoginRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "некорректные данные запроса"})
		return
	}

	valid := validator.New()
	if err := valid.Struct(req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "ошибка валидации", "details": err.Error()})
		return
	}

	user, err := LAPI.repo.GetUserByUsername(req.Username)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "неверные учетные данные"})
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password))
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "неверные учетные данные"})
		return
	}

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

func (LAPI *LibraryAPI) register(ctx *gin.Context) {
	var req models.RegisterRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "некорректные данные пользователя"})
		return
	}
	if req.Role != "" && !allowedUserRoles[req.Role] {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "недопустимая роль пользователя"})
		return
	}
	valid := validator.New()

	if err := valid.Struct(req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": validationErrorToErrorResponse(err).Error()})
		return
	}

	existingUser, _ := LAPI.repo.GetUserByUsername(req.Username)
	if existingUser != nil {
		ctx.JSON(http.StatusConflict, gin.H{"error": "пользователь уже существует"})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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

	log.Println(user.ID)
	err = LAPI.repo.CreateUser(&user)
	if err != nil {
		ctx.JSON(http.StatusConflict, gin.H{"error": err.Error()})
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

func (LAPI *LibraryAPI) getUser(ctx *gin.Context) {
	userID := ctx.Param("userID")

	user, err := LAPI.repo.GetUserByID(userID)
	if err != nil {
		if err == errors.ErrUserNotFound {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "пользователь не найден"})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "не удалось получить пользователя"})
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

func (LAPI *LibraryAPI) updateUser(ctx *gin.Context) {
	userID := ctx.Param("userID")

	var req models.UpdateUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "некорректные данные запроса"})
		return
	}
	if req.Role != "" && !allowedUserRoles[req.Role] {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "недопустимая роль пользователя"})
		return
	}

	user := &models.User{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password,
		Role:     req.Role,
	}

	if err := LAPI.repo.UpdateUser(userID, user); err != nil {
		if err == errors.ErrUserNotFound {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "пользователь не найден"})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "не удалось обновить пользователя"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "пользователь успешно обновлен"})
}

func (LAPI *LibraryAPI) deleteUser(ctx *gin.Context) {
	userID := ctx.Param("userID")

	if err := LAPI.repo.DeleteUser(userID); err != nil {
		if err == errors.ErrUserNotFound {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "пользователь не найден"})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "не удалось удалить пользователя"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "пользователь успешно удален"})
}

func (LAPI *LibraryAPI) getTasks(ctx *gin.Context) {
	tasks, err := LAPI.repo.GetTasks()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": errors.ErrInternalServer.Error()})
		return
	}
	if len(tasks) == 0 {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "задачи не найдены"})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"tasks": tasks})
}

func (LAPI *LibraryAPI) getTaskByID(ctx *gin.Context) {
	id := ctx.Param("taskID")
	task, err := LAPI.repo.GetTaskByID(id)
	if err != nil {
		if err == errors.ErrNotFound {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "задача не найдена"})
		} else {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": errors.ErrInternalServer.Error()})
		}
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

func (LAPI *LibraryAPI) createTask(ctx *gin.Context) {
	var req models.CreateTaskRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": errors.ErrBadRequest.Error()})
		return
	}
	valid := validator.New()
	if err := valid.Struct(req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": validationErrorToErrorResponse(err).Error()})
		return
	}
	task := models.Task{
		Title:       req.Title,
		Description: req.Description,
		Status:      "new",
	}
	if err := LAPI.repo.CreateTask(&task); err != nil {
		if err == errors.ErrConflict {
			ctx.JSON(http.StatusConflict, gin.H{"error": errors.ErrConflict.Error()})
		} else {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": errors.ErrInternalServer.Error()})
		}
		return
	}
	ctx.JSON(http.StatusCreated, gin.H{"task": task})
}

func (LAPI *LibraryAPI) updateTask(ctx *gin.Context) {
	id := ctx.Param("taskID")
	var req models.UpdateTaskRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": errors.ErrBadRequest.Error()})
		return
	}
	valid := validator.New()
	if err := valid.Struct(req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": validationErrorToErrorResponse(err).Error()})
		return
	}
	task, err := LAPI.repo.GetTaskByID(id)
	if err != nil {
		if err == errors.ErrNotFound {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "задача не найдена"})
		} else {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": errors.ErrInternalServer.Error()})
		}
		return
	}
	if req.Status != "" && !allowedTaskStatuses[req.Status] {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "недопустимый статус задачи"})
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
	if err := LAPI.repo.UpdateTask(id, task); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": errors.ErrInternalServer.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"task": task})
}

func (LAPI *LibraryAPI) deleteTask(ctx *gin.Context) {
	id := ctx.Param("taskID")
	if err := LAPI.repo.DeleteTask(id); err != nil {
		if err == errors.ErrNotFound {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "задача не найдена"})
		} else {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": errors.ErrInternalServer.Error()})
		}
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"message": "задача успешно удалена"})
}

func validationErrorToErrorResponse(err error) error {
	if verrs, ok := err.(validator.ValidationErrors); ok {
		for _, verr := range verrs {
			switch verr.Field() {
			case "Username":
				return errors.ErrInvalidUsername
			case "Email":
				return errors.ErrInvalidEmail
			case "Password":
				return errors.ErrInvalidPassword
			case "Role":
				return errors.ErrInvalidRole
			case "Status":
				return errors.ErrInvalidStatus
			case "Title":
				return errors.ErrInvalidTitle
			case "Description":
				return errors.ErrInvalidDescription
			}
		}
	}
	return errors.ErrValidationFailed
}
