package server

import (
	"context"
	"log"
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

type TaskRepository interface {
	CreateTask(ctx context.Context, task *models.Task) error
	GetTaskByID(ctx context.Context, id string) (*models.Task, error)
	GetTasks(ctx context.Context, userID string) ([]models.Task, error)
	UpdateTask(ctx context.Context, id string, task *models.Task) error
	DeleteTask(ctx context.Context, id string) error
}

type Repository interface {
	GetUserByID(id string) (*models.User, error)
	GetUserByUsername(username string) (*models.User, error)
	UpdateUser(id string, user *models.User) error
	DeleteUser(id string) error
	CreateUser(user *models.User) error
}

type TaskAPI struct {
	httpSrv  *http.Server
	repo     Repository
	taskRepo TaskRepository
}

func NewTaskAPI(repo Repository, taskRepo TaskRepository) *TaskAPI {
	if repo == nil || taskRepo == nil {
		return nil
	}

	cfg := ReadConfig()

	httpSrv := http.Server{
		Addr: cfg.Addr + ":" + strconv.Itoa(cfg.Port),
	}

	api := TaskAPI{
		httpSrv:  &httpSrv,
		repo:     repo,
		taskRepo: taskRepo,
	}

	api.configRoutes()

	return &api
}

func (api *TaskAPI) Start() error {
	if api.httpSrv == nil {
		return errors.ErrInternalServer
	}

	if api.httpSrv.Addr == "" {
		api.httpSrv.Addr = ":8080"
	}

	return api.httpSrv.ListenAndServe()
}

func (api *TaskAPI) configRoutes() {
	router := gin.Default()

	router.NoMethod(func(ctx *gin.Context) {
		ctx.JSON(http.StatusMethodNotAllowed, gin.H{"error": "использован некорректный HTTP-метод"})
	})

	user := router.Group("/users")
	{
		user.POST("/login", api.login)
		user.POST("/register", api.register)
		user.PUT("/update/:userID", api.updateUser)
		user.DELETE("/delete/:userID", api.deleteUser)
		user.GET("/:userID", api.getUser)
	}

	tasks := router.Group("/tasks")
	{
		tasks.GET("", api.getTasks)
		tasks.GET(":taskID", api.getTaskByID)
		tasks.POST("", api.createTask)
		tasks.PUT(":taskID", api.updateTask)
		tasks.DELETE(":taskID", api.deleteTask)
	}

	api.httpSrv.Handler = router
}

func (api *TaskAPI) login(ctx *gin.Context) {
	var req models.LoginRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": errors.ErrInvalidRequest.Error()})
		return
	}

	valid := validator.New()
	if err := valid.Struct(req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": errors.ErrValidationFailed.Error(), "details": err.Error()})
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

	log.Println(user.ID)
	err = api.repo.CreateUser(&user)
	if err != nil {
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

func (api *TaskAPI) getTaskByID(ctx *gin.Context) {
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

func (api *TaskAPI) updateTask(ctx *gin.Context) {
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

func (api *TaskAPI) deleteTask(ctx *gin.Context) {
	id := ctx.Param("taskID")
	if err := api.taskRepo.DeleteTask(ctx.Request.Context(), id); err != nil {
		if err == errors.ErrNotFound {
			ctx.JSON(http.StatusNotFound, gin.H{"error": errors.ErrTaskNotFound.Error()})
		} else {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": errors.ErrInternalServer.Error()})
		}
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"message": "задача успешно удалена"})
}
