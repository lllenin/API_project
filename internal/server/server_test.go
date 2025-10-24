package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"project/internal/domain/errors"
	"project/internal/domain/models"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"
)

type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) GetUserByID(id string) (*models.User, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockRepository) GetUserByUsername(username string) (*models.User, error) {
	args := m.Called(username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockRepository) UpdateUser(id string, user *models.User) error {
	args := m.Called(id, user)
	return args.Error(0)
}

func (m *MockRepository) DeleteUser(id string) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockRepository) CreateUser(user *models.User) error {
	args := m.Called(user)
	return args.Error(0)
}

type MockTaskRepository struct {
	mock.Mock
}

func (m *MockTaskRepository) CreateTask(ctx context.Context, task *models.Task) error {
	args := m.Called(ctx, task)
	return args.Error(0)
}

func (m *MockTaskRepository) GetTaskByID(ctx context.Context, id string) (*models.Task, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Task), args.Error(1)
}

func (m *MockTaskRepository) GetTasks(ctx context.Context, userID string) ([]models.Task, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]models.Task), args.Error(1)
}

func (m *MockTaskRepository) UpdateTask(ctx context.Context, id string, task *models.Task) error {
	args := m.Called(ctx, id, task)
	return args.Error(0)
}

func (m *MockTaskRepository) DeleteTask(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockTaskRepository) EnqueueHardDelete(taskID string) {
	m.Called(taskID)
}

func TestRegister(t *testing.T) {
	tests := []struct {
		name    string
		request models.RegisterRequest
		want    struct {
			statusCode int
			success    bool
		}
		mockSetup func(*MockRepository)
	}{
		{
			name: "successful registration",
			request: models.RegisterRequest{
				Username: "testuser",
				Email:    "test@example.com",
				Password: "password123",
				Role:     "user",
			},
			want: struct {
				statusCode int
				success    bool
			}{
				statusCode: 201,
				success:    true,
			},
			mockSetup: func(mockRepo *MockRepository) {
				mockRepo.On("GetUserByUsername", "testuser").Return(nil, errors.ErrUserNotFound)
				mockRepo.On("CreateUser", mock.AnythingOfType("*models.User")).Return(nil)
			},
		},
		{
			name: "user already exists",
			request: models.RegisterRequest{
				Username: "existinguser",
				Email:    "existing@example.com",
				Password: "password123",
				Role:     "user",
			},
			want: struct {
				statusCode int
				success    bool
			}{
				statusCode: 409,
				success:    false,
			},
			mockSetup: func(mockRepo *MockRepository) {
				existingUser := &models.User{
					ID:       "user1",
					Username: "existinguser",
					Email:    "existing@example.com",
					Password: "password123",
					Role:     "user",
				}
				mockRepo.On("GetUserByUsername", "existinguser").Return(existingUser, nil)
			},
		},
		{
			name: "invalid input data",
			request: models.RegisterRequest{
				Username: "",
				Email:    "invalid-email",
				Password: "123",
				Role:     "invalid",
			},
			want: struct {
				statusCode int
				success    bool
			}{
				statusCode: 400,
				success:    false,
			},
			mockSetup: func(mockRepo *MockRepository) {
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			mockRepo := &MockRepository{}
			mockTaskRepo := &MockTaskRepository{}
			tt.mockSetup(mockRepo)

			api := NewTaskAPI(mockRepo, mockTaskRepo, &Config{})

			jsonData, _ := json.Marshal(tt.request)
			req, _ := http.NewRequest("POST", "/users/register", bytes.NewBuffer(jsonData))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			api.httpSrv.Handler.ServeHTTP(w, req)

			assert.Equal(t, tt.want.statusCode, w.Code)
			if tt.want.success {
				assert.Contains(t, w.Body.String(), "пользователь успешно создан")
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestLogin(t *testing.T) {
	tests := []struct {
		name    string
		request models.LoginRequest
		want    struct {
			statusCode int
			success    bool
		}
		mockSetup func(*MockRepository)
	}{
		{
			name: "successful login",
			request: models.LoginRequest{
				Username: "testuser",
				Password: "password123",
			},
			want: struct {
				statusCode int
				success    bool
			}{
				statusCode: 200,
				success:    true,
			},
			mockSetup: func(mockRepo *MockRepository) {
				hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
				user := &models.User{
					ID:       "user123",
					Username: "testuser",
					Email:    "test@example.com",
					Password: string(hashedPassword),
					Role:     "user",
				}
				mockRepo.On("GetUserByUsername", "testuser").Return(user, nil)
			},
		},
		{
			name: "user not found",
			request: models.LoginRequest{
				Username: "nonexistent",
				Password: "password123",
			},
			want: struct {
				statusCode int
				success    bool
			}{
				statusCode: 401,
				success:    false,
			},
			mockSetup: func(mockRepo *MockRepository) {
				mockRepo.On("GetUserByUsername", "nonexistent").Return(nil, errors.ErrUserNotFound)
			},
		},
		{
			name: "invalid password",
			request: models.LoginRequest{
				Username: "testuser",
				Password: "wrongpassword",
			},
			want: struct {
				statusCode int
				success    bool
			}{
				statusCode: 401,
				success:    false,
			},
			mockSetup: func(mockRepo *MockRepository) {
				hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
				user := &models.User{
					ID:       "user123",
					Username: "testuser",
					Email:    "test@example.com",
					Password: string(hashedPassword),
					Role:     "user",
				}
				mockRepo.On("GetUserByUsername", "testuser").Return(user, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			mockRepo := &MockRepository{}
			mockTaskRepo := &MockTaskRepository{}
			tt.mockSetup(mockRepo)

			api := NewTaskAPI(mockRepo, mockTaskRepo, &Config{})

			jsonData, _ := json.Marshal(tt.request)
			req, _ := http.NewRequest("POST", "/users/login", bytes.NewBuffer(jsonData))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			api.httpSrv.Handler.ServeHTTP(w, req)

			assert.Equal(t, tt.want.statusCode, w.Code)
			if tt.want.success {
				assert.Contains(t, w.Body.String(), "вход выполнен успешно")
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestCreateTask(t *testing.T) {
	tests := []struct {
		name    string
		request models.CreateTaskRequest
		userID  string
		want    struct {
			statusCode int
			success    bool
		}
		mockSetup func(*MockTaskRepository)
	}{
		{
			name: "successful task creation",
			request: models.CreateTaskRequest{
				Title:       "Test Task",
				Description: "Test Description",
			},
			userID: "user123",
			want: struct {
				statusCode int
				success    bool
			}{
				statusCode: 201,
				success:    true,
			},
			mockSetup: func(mockTaskRepo *MockTaskRepository) {
				mockTaskRepo.On("CreateTask", mock.Anything, mock.AnythingOfType("*models.Task")).Return(nil)
			},
		},
		{
			name: "invalid task data",
			request: models.CreateTaskRequest{
				Title:       "",
				Description: "Test Description",
			},
			userID: "user123",
			want: struct {
				statusCode int
				success    bool
			}{
				statusCode: 400,
				success:    false,
			},
			mockSetup: func(mockTaskRepo *MockTaskRepository) {
			},
		},
		{
			name: "database error",
			request: models.CreateTaskRequest{
				Title:       "Test Task",
				Description: "Test Description",
			},
			userID: "user123",
			want: struct {
				statusCode int
				success    bool
			}{
				statusCode: 500,
				success:    false,
			},
			mockSetup: func(mockTaskRepo *MockTaskRepository) {
				mockTaskRepo.On("CreateTask", mock.Anything, mock.AnythingOfType("*models.Task")).Return(errors.ErrInternalServer)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			mockRepo := &MockRepository{}
			mockTaskRepo := &MockTaskRepository{}
			tt.mockSetup(mockTaskRepo)

			api := NewTaskAPI(mockRepo, mockTaskRepo, &Config{})

			jsonData, _ := json.Marshal(tt.request)
			req, _ := http.NewRequest("POST", "/tasks", bytes.NewBuffer(jsonData))
			req.Header.Set("Content-Type", "application/json")
			req.AddCookie(&http.Cookie{
				Name:  "jwt_token",
				Value: generateTestToken(tt.userID),
			})

			w := httptest.NewRecorder()
			api.httpSrv.Handler.ServeHTTP(w, req)

			assert.Equal(t, tt.want.statusCode, w.Code)
			if tt.want.success {
				assert.Contains(t, w.Body.String(), "task")
			}

			mockTaskRepo.AssertExpectations(t)
		})
	}
}

func TestGetTasks(t *testing.T) {
	tests := []struct {
		name   string
		userID string
		want   struct {
			statusCode int
			success    bool
		}
		mockSetup func(*MockTaskRepository)
	}{
		{
			name:   "successful tasks retrieval",
			userID: "user123",
			want: struct {
				statusCode int
				success    bool
			}{
				statusCode: 200,
				success:    true,
			},
			mockSetup: func(mockTaskRepo *MockTaskRepository) {
				tasks := []models.Task{
					{
						ID:          "task1",
						Title:       "Task 1",
						Description: "Description 1",
						Status:      "new",
						UserID:      "user123",
					},
				}
				mockTaskRepo.On("GetTasks", mock.Anything, "user123").Return(tasks, nil)
			},
		},
		{
			name:   "database error",
			userID: "user123",
			want: struct {
				statusCode int
				success    bool
			}{
				statusCode: 500,
				success:    false,
			},
			mockSetup: func(mockTaskRepo *MockTaskRepository) {
				mockTaskRepo.On("GetTasks", mock.Anything, "user123").Return([]models.Task{}, errors.ErrInternalServer)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			mockRepo := &MockRepository{}
			mockTaskRepo := &MockTaskRepository{}
			tt.mockSetup(mockTaskRepo)

			api := NewTaskAPI(mockRepo, mockTaskRepo, &Config{})

			req, _ := http.NewRequest("GET", "/tasks", nil)
			req.AddCookie(&http.Cookie{
				Name:  "jwt_token",
				Value: generateTestToken(tt.userID),
			})

			w := httptest.NewRecorder()
			api.httpSrv.Handler.ServeHTTP(w, req)

			assert.Equal(t, tt.want.statusCode, w.Code)
			if tt.want.success {
				assert.Contains(t, w.Body.String(), "tasks")
			}

			mockTaskRepo.AssertExpectations(t)
		})
	}
}

func TestUpdateTask(t *testing.T) {
	tests := []struct {
		name    string
		taskID  string
		request models.UpdateTaskRequest
		userID  string
		want    struct {
			statusCode int
			success    bool
		}
		mockSetup func(*MockTaskRepository)
	}{
		{
			name:   "successful task update",
			taskID: "task123",
			request: models.UpdateTaskRequest{
				Title:       "Updated Task",
				Description: "Updated Description",
				Status:      "in_progress",
			},
			userID: "user123",
			want: struct {
				statusCode int
				success    bool
			}{
				statusCode: 200,
				success:    true,
			},
			mockSetup: func(mockTaskRepo *MockTaskRepository) {
				task := &models.Task{
					ID:          "task123",
					Title:       "Original Task",
					Description: "Original Description",
					Status:      "new",
					UserID:      "user123",
				}
				mockTaskRepo.On("GetTaskByID", mock.Anything, "task123").Return(task, nil)
				mockTaskRepo.On("UpdateTask", mock.Anything, "task123", mock.AnythingOfType("*models.Task")).Return(nil)
			},
		},
		{
			name:   "task not found",
			taskID: "nonexistent",
			request: models.UpdateTaskRequest{
				Title: "Updated Task",
			},
			userID: "user123",
			want: struct {
				statusCode int
				success    bool
			}{
				statusCode: 404,
				success:    false,
			},
			mockSetup: func(mockTaskRepo *MockTaskRepository) {
				mockTaskRepo.On("GetTaskByID", mock.Anything, "nonexistent").Return(nil, errors.ErrNotFound)
			},
		},
		{
			name:   "unauthorized access",
			taskID: "task123",
			request: models.UpdateTaskRequest{
				Title: "Updated Task",
			},
			userID: "user456",
			want: struct {
				statusCode int
				success    bool
			}{
				statusCode: 403,
				success:    false,
			},
			mockSetup: func(mockTaskRepo *MockTaskRepository) {
				task := &models.Task{
					ID:          "task123",
					Title:       "Original Task",
					Description: "Original Description",
					Status:      "new",
					UserID:      "user123",
				}
				mockTaskRepo.On("GetTaskByID", mock.Anything, "task123").Return(task, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			mockRepo := &MockRepository{}
			mockTaskRepo := &MockTaskRepository{}
			tt.mockSetup(mockTaskRepo)

			api := NewTaskAPI(mockRepo, mockTaskRepo, &Config{})

			jsonData, _ := json.Marshal(tt.request)
			req, _ := http.NewRequest("PUT", "/tasks/"+tt.taskID, bytes.NewBuffer(jsonData))
			req.Header.Set("Content-Type", "application/json")
			req.AddCookie(&http.Cookie{
				Name:  "jwt_token",
				Value: generateTestToken(tt.userID),
			})

			w := httptest.NewRecorder()
			api.httpSrv.Handler.ServeHTTP(w, req)

			assert.Equal(t, tt.want.statusCode, w.Code)
			if tt.want.success {
				assert.Contains(t, w.Body.String(), "task")
			}

			mockTaskRepo.AssertExpectations(t)
		})
	}
}

func TestDeleteTask(t *testing.T) {
	tests := []struct {
		name   string
		taskID string
		userID string
		want   struct {
			statusCode int
			success    bool
		}
		mockSetup func(*MockTaskRepository)
	}{
		{
			name:   "successful task deletion",
			taskID: "task123",
			userID: "user123",
			want: struct {
				statusCode int
				success    bool
			}{
				statusCode: 200,
				success:    true,
			},
			mockSetup: func(mockTaskRepo *MockTaskRepository) {
				task := &models.Task{
					ID:          "task123",
					Title:       "Test Task",
					Description: "Test Description",
					Status:      "new",
					UserID:      "user123",
				}
				mockTaskRepo.On("GetTaskByID", mock.Anything, "task123").Return(task, nil)
				mockTaskRepo.On("DeleteTask", mock.Anything, "task123").Return(nil)
				mockTaskRepo.On("EnqueueHardDelete", "task123").Return()
			},
		},
		{
			name:   "task not found",
			taskID: "nonexistent",
			userID: "user123",
			want: struct {
				statusCode int
				success    bool
			}{
				statusCode: 404,
				success:    false,
			},
			mockSetup: func(mockTaskRepo *MockTaskRepository) {
				mockTaskRepo.On("GetTaskByID", mock.Anything, "nonexistent").Return(nil, errors.ErrNotFound)
			},
		},
		{
			name:   "unauthorized access",
			taskID: "task123",
			userID: "user456",
			want: struct {
				statusCode int
				success    bool
			}{
				statusCode: 403,
				success:    false,
			},
			mockSetup: func(mockTaskRepo *MockTaskRepository) {
				task := &models.Task{
					ID:          "task123",
					Title:       "Test Task",
					Description: "Test Description",
					Status:      "new",
					UserID:      "user123",
				}
				mockTaskRepo.On("GetTaskByID", mock.Anything, "task123").Return(task, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			mockRepo := &MockRepository{}
			mockTaskRepo := &MockTaskRepository{}
			tt.mockSetup(mockTaskRepo)

			api := NewTaskAPI(mockRepo, mockTaskRepo, &Config{})

			req, _ := http.NewRequest("DELETE", "/tasks/"+tt.taskID, nil)
			req.AddCookie(&http.Cookie{
				Name:  "jwt_token",
				Value: generateTestToken(tt.userID),
			})

			w := httptest.NewRecorder()
			api.httpSrv.Handler.ServeHTTP(w, req)

			assert.Equal(t, tt.want.statusCode, w.Code)
			if tt.want.success {
				assert.Contains(t, w.Body.String(), "задача")
			}

			mockTaskRepo.AssertExpectations(t)
		})
	}
}

func generateTestToken(userID string) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(time.Hour * 24).Unix(),
	})
	tokenString, _ := token.SignedString([]byte("shouldbeinVaultsecret"))
	return tokenString
}

func TestServerErrorHandling(t *testing.T) {
	tests := []struct {
		name    string
		request interface{}
		method  string
		path    string
		want    struct {
			statusCode int
			hasError   bool
		}
		mockSetup func(*MockRepository, *MockTaskRepository)
	}{
		{
			name:    "invalid JSON in request",
			request: "invalid json",
			method:  "POST",
			path:    "/users/register",
			want: struct {
				statusCode int
				hasError   bool
			}{
				statusCode: 400,
				hasError:   true,
			},
			mockSetup: func(mockRepo *MockRepository, mockTaskRepo *MockTaskRepository) {
			},
		},
		{
			name: "missing required fields",
			request: map[string]interface{}{
				"username": "testuser",
			},
			method: "POST",
			path:   "/users/register",
			want: struct {
				statusCode int
				hasError   bool
			}{
				statusCode: 400,
				hasError:   true,
			},
			mockSetup: func(mockRepo *MockRepository, mockTaskRepo *MockTaskRepository) {
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			mockRepo := &MockRepository{}
			mockTaskRepo := &MockTaskRepository{}
			tt.mockSetup(mockRepo, mockTaskRepo)

			api := NewTaskAPI(mockRepo, mockTaskRepo, &Config{})

			var req *http.Request
			if tt.request == "invalid json" {
				req, _ = http.NewRequest(tt.method, tt.path, strings.NewReader("invalid json"))
			} else {
				jsonData, _ := json.Marshal(tt.request)
				req, _ = http.NewRequest(tt.method, tt.path, bytes.NewBuffer(jsonData))
			}
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			api.httpSrv.Handler.ServeHTTP(w, req)

			assert.Equal(t, tt.want.statusCode, w.Code)
			if tt.want.hasError {
				assert.Contains(t, w.Body.String(), "error")
			}
		})
	}
}

func TestServerMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockRepo := &MockRepository{}
	mockTaskRepo := &MockTaskRepository{}
	api := NewTaskAPI(mockRepo, mockTaskRepo, &Config{})

	req, _ := http.NewRequest("OPTIONS", "/users/register", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "POST")

	w := httptest.NewRecorder()
	api.httpSrv.Handler.ServeHTTP(w, req)

	assert.True(t, w.Code >= 200 && w.Code < 600, "Expected valid HTTP status, got %d", w.Code)
}

func TestServerRateLimiting(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockRepo := &MockRepository{}
	mockTaskRepo := &MockTaskRepository{}

	mockTaskRepo.On("GetTasks", mock.Anything, "user123").Return([]models.Task{}, nil)

	api := NewTaskAPI(mockRepo, mockTaskRepo, &Config{})

	for i := 0; i < 3; i++ {
		req, _ := http.NewRequest("GET", "/tasks", nil)
		req.AddCookie(&http.Cookie{Name: "jwt_token", Value: generateTestToken("user123")})

		w := httptest.NewRecorder()
		api.httpSrv.Handler.ServeHTTP(w, req)

		assert.True(t, w.Code >= 200 && w.Code < 600, "Expected valid HTTP status, got %d", w.Code)
	}
}

func TestServerGracefulShutdown(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockRepo := &MockRepository{}
	mockTaskRepo := &MockTaskRepository{}
	api := NewTaskAPI(mockRepo, mockTaskRepo, &Config{})

	assert.NotNil(t, api)
	assert.NotNil(t, api.httpSrv)
}

func BenchmarkLogin(b *testing.B) {
	gin.SetMode(gin.TestMode)
	mockRepo := &MockRepository{}
	mockTaskRepo := &MockTaskRepository{}

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	user := &models.User{
		ID:       "user123",
		Username: "testuser",
		Email:    "test@example.com",
		Password: string(hashedPassword),
		Role:     "user",
	}
	mockRepo.On("GetUserByUsername", "testuser").Return(user, nil)

	api := NewTaskAPI(mockRepo, mockTaskRepo, &Config{})

	loginRequest := models.LoginRequest{
		Username: "testuser",
		Password: "password123",
	}
	jsonData, _ := json.Marshal(loginRequest)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("POST", "/users/login", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		api.httpSrv.Handler.ServeHTTP(w, req)
	}
}

func BenchmarkRegister(b *testing.B) {
	gin.SetMode(gin.TestMode)
	mockRepo := &MockRepository{}
	mockTaskRepo := &MockTaskRepository{}

	mockRepo.On("GetUserByUsername", "testuser").Return(nil, errors.ErrUserNotFound)
	mockRepo.On("CreateUser", mock.AnythingOfType("*models.User")).Return(nil)

	api := NewTaskAPI(mockRepo, mockTaskRepo, &Config{})

	registerRequest := models.RegisterRequest{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
		Role:     "user",
	}
	jsonData, _ := json.Marshal(registerRequest)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("POST", "/users/register", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		api.httpSrv.Handler.ServeHTTP(w, req)
	}
}

func BenchmarkCreateTask(b *testing.B) {
	gin.SetMode(gin.TestMode)
	mockRepo := &MockRepository{}
	mockTaskRepo := &MockTaskRepository{}

	mockTaskRepo.On("CreateTask", mock.Anything, mock.AnythingOfType("*models.Task")).Return(nil)

	api := NewTaskAPI(mockRepo, mockTaskRepo, &Config{})

	createTaskRequest := models.CreateTaskRequest{
		Title:       "Test Task",
		Description: "Test Description",
	}
	jsonData, _ := json.Marshal(createTaskRequest)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("POST", "/tasks", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(&http.Cookie{
			Name:  "jwt_token",
			Value: generateTestToken("user123"),
		})

		w := httptest.NewRecorder()
		api.httpSrv.Handler.ServeHTTP(w, req)
	}
}

func BenchmarkGetTasks(b *testing.B) {
	gin.SetMode(gin.TestMode)
	mockRepo := &MockRepository{}
	mockTaskRepo := &MockTaskRepository{}

	tasks := []models.Task{
		{
			ID:          "task1",
			Title:       "Task 1",
			Description: "Description 1",
			Status:      "new",
			UserID:      "user123",
		},
		{
			ID:          "task2",
			Title:       "Task 2",
			Description: "Description 2",
			Status:      "in_progress",
			UserID:      "user123",
		},
	}
	mockTaskRepo.On("GetTasks", mock.Anything, "user123").Return(tasks, nil)

	api := NewTaskAPI(mockRepo, mockTaskRepo, &Config{})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("GET", "/tasks", nil)
		req.AddCookie(&http.Cookie{
			Name:  "jwt_token",
			Value: generateTestToken("user123"),
		})

		w := httptest.NewRecorder()
		api.httpSrv.Handler.ServeHTTP(w, req)
	}
}
