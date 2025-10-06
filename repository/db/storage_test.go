package db

import (
	"context"
	"fmt"
	"log"
	"os"
	"project/internal/domain/models"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testDBConnStr = "postgres://shouldbeinVaultuser:shouldbeinVaultpassword@localhost:5432/tasks?sslmode=disable"

func setupTestDB(t *testing.T) *Storage {
	conn, err := pgx.Connect(context.Background(), testDBConnStr)
	if err != nil {
		t.Skipf("Skipping test: cannot connect to test database: %v", err)
		return nil
	}
	defer func() {
		if err := conn.Close(context.Background()); err != nil {
			log.Printf("Error closing connection: %v", err)
		}
	}()

	storage, err := NewStorage(testDBConnStr)
	require.NoError(t, err)
	require.NotNil(t, storage)

	return storage
}

func cleanupTestData(t *testing.T, storage *Storage) {
	ctx := context.Background()

	_, err := storage.conn.Exec(ctx, "DELETE FROM tasks")
	if err != nil {
		t.Logf("Warning: failed to cleanup tasks: %v", err)
	}

	_, err = storage.conn.Exec(ctx, "DELETE FROM users")
	if err != nil {
		t.Logf("Warning: failed to cleanup users: %v", err)
	}
}

func TestMain(m *testing.M) {
	conn, err := pgx.Connect(context.Background(), testDBConnStr)
	if err != nil {
		log.Printf("Cannot connect to test database: %v", err)
		os.Exit(1)
	}
	defer func() {
		if err := conn.Close(context.Background()); err != nil {
			log.Printf("Error closing connection: %v", err)
		}
	}()

	err = Migration(testDBConnStr, "../../migrations")
	if err != nil {
		log.Printf("Failed to run migrations: %v", err)
		os.Exit(1)
	}

	code := m.Run()
	os.Exit(code)
}

func TestNewStorage(t *testing.T) {
	tests := []struct {
		name        string
		connStr     string
		wantErr     bool
		wantStorage bool
	}{
		{
			name:        "valid connection string",
			connStr:     testDBConnStr,
			wantErr:     false,
			wantStorage: true,
		},
		{
			name:        "invalid connection string",
			connStr:     "invalid_connection",
			wantErr:     true,
			wantStorage: false,
		},
		{
			name:        "empty connection string",
			connStr:     "",
			wantErr:     true,
			wantStorage: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage, err := NewStorage(tt.connStr)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, storage)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, storage)
				if storage != nil {
					_ = storage.conn.Close(context.Background())
				}
			}
		})
	}
}

func TestStorageCreateTask(t *testing.T) {
	storage := setupTestDB(t)
	if storage == nil {
		return
	}
	defer func() {
		if err := storage.conn.Close(context.Background()); err != nil {
			t.Logf("Error closing connection: %v", err)
		}
	}()
	defer cleanupTestData(t, storage)

	user := &models.User{
		ID:       uuid.New().String(),
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
		Role:     "user",
	}
	err := storage.CreateUser(user)
	require.NoError(t, err)

	task := &models.Task{
		Title:       "Test Task",
		Description: "Test Description",
		Status:      "new",
		UserID:      user.ID,
	}

	err = storage.CreateTask(context.Background(), task)
	assert.NoError(t, err)
	assert.NotEmpty(t, task.ID)
	assert.False(t, task.Deleted)
}

func TestStorageGetTaskByID(t *testing.T) {
	storage := setupTestDB(t)
	if storage == nil {
		return
	}
	defer func() {
		if err := storage.conn.Close(context.Background()); err != nil {
			t.Logf("Error closing connection: %v", err)
		}
	}()
	defer cleanupTestData(t, storage)

	user := &models.User{
		ID:       uuid.New().String(),
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
		Role:     "user",
	}
	err := storage.CreateUser(user)
	require.NoError(t, err)

	task := &models.Task{
		Title:       "Test Task",
		Description: "Test Description",
		Status:      "new",
		UserID:      user.ID,
	}
	err = storage.CreateTask(context.Background(), task)
	require.NoError(t, err)

	retrievedTask, err := storage.GetTaskByID(context.Background(), task.ID)
	assert.NoError(t, err)
	assert.NotNil(t, retrievedTask)
	assert.Equal(t, task.ID, retrievedTask.ID)
	assert.Equal(t, task.Title, retrievedTask.Title)

	nonExistentTask, err := storage.GetTaskByID(context.Background(), uuid.New().String())
	assert.Error(t, err)
	assert.Nil(t, nonExistentTask)
}

func TestStorageGetTasks(t *testing.T) {
	storage := setupTestDB(t)
	if storage == nil {
		return
	}
	defer func() {
		if err := storage.conn.Close(context.Background()); err != nil {
			t.Logf("Error closing connection: %v", err)
		}
	}()
	defer cleanupTestData(t, storage)

	user := &models.User{
		ID:       uuid.New().String(),
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
		Role:     "user",
	}
	err := storage.CreateUser(user)
	require.NoError(t, err)

	task1 := &models.Task{
		Title:       "Task 1",
		Description: "Description 1",
		Status:      "new",
		UserID:      user.ID,
	}
	task2 := &models.Task{
		Title:       "Task 2",
		Description: "Description 2",
		Status:      "in_progress",
		UserID:      user.ID,
	}

	err = storage.CreateTask(context.Background(), task1)
	require.NoError(t, err)
	err = storage.CreateTask(context.Background(), task2)
	require.NoError(t, err)

	tasks, err := storage.GetTasks(context.Background(), user.ID)
	assert.NoError(t, err)
	assert.Len(t, tasks, 2)
}

func TestStorageUpdateTask(t *testing.T) {
	storage := setupTestDB(t)
	if storage == nil {
		return
	}
	defer func() {
		if err := storage.conn.Close(context.Background()); err != nil {
			t.Logf("Error closing connection: %v", err)
		}
	}()
	defer cleanupTestData(t, storage)

	user := &models.User{
		ID:       uuid.New().String(),
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
		Role:     "user",
	}
	err := storage.CreateUser(user)
	require.NoError(t, err)

	task := &models.Task{
		Title:       "Test Task",
		Description: "Test Description",
		Status:      "new",
		UserID:      user.ID,
	}
	err = storage.CreateTask(context.Background(), task)
	require.NoError(t, err)

	updatedTask := &models.Task{
		Title:       "Updated Task",
		Description: "Updated Description",
		Status:      "in_progress",
	}
	err = storage.UpdateTask(context.Background(), task.ID, updatedTask)
	assert.NoError(t, err)
}

func TestStorageDeleteTask(t *testing.T) {
	storage := setupTestDB(t)
	if storage == nil {
		return
	}
	defer func() {
		if err := storage.conn.Close(context.Background()); err != nil {
			t.Logf("Error closing connection: %v", err)
		}
	}()
	defer cleanupTestData(t, storage)

	user := &models.User{
		ID:       uuid.New().String(),
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
		Role:     "user",
	}
	err := storage.CreateUser(user)
	require.NoError(t, err)

	task := &models.Task{
		Title:       "Test Task",
		Description: "Test Description",
		Status:      "new",
		UserID:      user.ID,
	}
	err = storage.CreateTask(context.Background(), task)
	require.NoError(t, err)

	err = storage.DeleteTask(context.Background(), task.ID)
	assert.NoError(t, err)
}

func TestStorageCreateUser(t *testing.T) {
	storage := setupTestDB(t)
	if storage == nil {
		return
	}
	defer func() {
		if err := storage.conn.Close(context.Background()); err != nil {
			t.Logf("Error closing connection: %v", err)
		}
	}()
	defer cleanupTestData(t, storage)

	user := &models.User{
		ID:       uuid.New().String(),
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
		Role:     "user",
	}

	err := storage.CreateUser(user)
	assert.NoError(t, err)
}

func TestStorageGetUserByID(t *testing.T) {
	storage := setupTestDB(t)
	if storage == nil {
		return
	}
	defer func() {
		if err := storage.conn.Close(context.Background()); err != nil {
			t.Logf("Error closing connection: %v", err)
		}
	}()
	defer cleanupTestData(t, storage)

	user := &models.User{
		ID:       uuid.New().String(),
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
		Role:     "user",
	}
	err := storage.CreateUser(user)
	require.NoError(t, err)

	retrievedUser, err := storage.GetUserByID(user.ID)
	assert.NoError(t, err)
	assert.NotNil(t, retrievedUser)
	assert.Equal(t, user.ID, retrievedUser.ID)
	assert.Equal(t, user.Username, retrievedUser.Username)

	nonExistentUser, err := storage.GetUserByID(uuid.New().String())
	assert.Error(t, err)
	assert.Nil(t, nonExistentUser)
}

func TestStorageGetUserByUsername(t *testing.T) {
	storage := setupTestDB(t)
	if storage == nil {
		return
	}
	defer func() {
		if err := storage.conn.Close(context.Background()); err != nil {
			t.Logf("Error closing connection: %v", err)
		}
	}()
	defer cleanupTestData(t, storage)

	user := &models.User{
		ID:       uuid.New().String(),
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
		Role:     "user",
	}
	err := storage.CreateUser(user)
	require.NoError(t, err)

	retrievedUser, err := storage.GetUserByUsername(user.Username)
	assert.NoError(t, err)
	assert.NotNil(t, retrievedUser)
	assert.Equal(t, user.ID, retrievedUser.ID)
	assert.Equal(t, user.Username, retrievedUser.Username)

	nonExistentUser, err := storage.GetUserByUsername("nonexistent")
	assert.Error(t, err)
	assert.Nil(t, nonExistentUser)
}

func TestStorageUpdateUser(t *testing.T) {
	storage := setupTestDB(t)
	if storage == nil {
		return
	}
	defer func() {
		if err := storage.conn.Close(context.Background()); err != nil {
			t.Logf("Error closing connection: %v", err)
		}
	}()
	defer cleanupTestData(t, storage)

	user := &models.User{
		ID:       uuid.New().String(),
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
		Role:     "user",
	}
	err := storage.CreateUser(user)
	require.NoError(t, err)

	updatedUser := &models.User{
		Username: "updateduser",
		Email:    "updated@example.com",
		Password: "newpassword",
		Role:     "admin",
	}
	err = storage.UpdateUser(user.ID, updatedUser)
	assert.NoError(t, err)
}

func TestStorageDeleteUser(t *testing.T) {
	storage := setupTestDB(t)
	if storage == nil {
		return
	}
	defer func() {
		if err := storage.conn.Close(context.Background()); err != nil {
			t.Logf("Error closing connection: %v", err)
		}
	}()
	defer cleanupTestData(t, storage)

	user := &models.User{
		ID:       uuid.New().String(),
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
		Role:     "user",
	}
	err := storage.CreateUser(user)
	require.NoError(t, err)

	err = storage.DeleteUser(user.ID)
	assert.NoError(t, err)
}

func TestStorageEnqueueHardDelete(t *testing.T) {
	storage := setupTestDB(t)
	if storage == nil {
		return
	}
	defer func() {
		if err := storage.conn.Close(context.Background()); err != nil {
			t.Logf("Error closing connection: %v", err)
		}
	}()
	defer cleanupTestData(t, storage)

	assert.NotPanics(t, func() {
		storage.EnqueueHardDelete(uuid.New().String())
	})
}

func TestStorageTryEnqueueOrFlush(t *testing.T) {
	storage := setupTestDB(t)
	if storage == nil {
		return
	}
	defer func() {
		if err := storage.conn.Close(context.Background()); err != nil {
			t.Logf("Error closing connection: %v", err)
		}
	}()
	defer cleanupTestData(t, storage)

	assert.NotPanics(t, func() {
		storage.tryEnqueueOrFlush()
	})
}

func TestStorageDrainDeleteQueue(t *testing.T) {
	storage := setupTestDB(t)
	if storage == nil {
		return
	}
	defer func() {
		if err := storage.conn.Close(context.Background()); err != nil {
			t.Logf("Error closing connection: %v", err)
		}
	}()
	defer cleanupTestData(t, storage)

	assert.NotPanics(t, func() {
		storage.drainDeleteQueue()
	})
}

func TestStorageHardDeleteAllFlagged(t *testing.T) {
	storage := setupTestDB(t)
	if storage == nil {
		return
	}
	defer func() {
		if err := storage.conn.Close(context.Background()); err != nil {
			t.Logf("Error closing connection: %v", err)
		}
	}()
	defer cleanupTestData(t, storage)

	user := &models.User{
		ID:       uuid.New().String(),
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
		Role:     "user",
	}
	err := storage.CreateUser(user)
	require.NoError(t, err)

	task := &models.Task{
		Title:       "Test Task",
		Description: "Test Description",
		Status:      "new",
		UserID:      user.ID,
	}
	err = storage.CreateTask(context.Background(), task)
	require.NoError(t, err)

	err = storage.DeleteTask(context.Background(), task.ID)
	require.NoError(t, err)

	count, err := storage.hardDeleteAllFlagged(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, int64(1), count)
}

func TestStorageIntegration(t *testing.T) {
	storage := setupTestDB(t)
	if storage == nil {
		return
	}
	defer func() {
		if err := storage.conn.Close(context.Background()); err != nil {
			t.Logf("Error closing connection: %v", err)
		}
	}()
	defer cleanupTestData(t, storage)

	user := &models.User{
		ID:       uuid.New().String(),
		Username: "integrationuser",
		Email:    "integration@example.com",
		Password: "password123",
		Role:     "user",
	}
	err := storage.CreateUser(user)
	require.NoError(t, err)

	task := &models.Task{
		Title:       "Integration Task",
		Description: "Integration Description",
		Status:      "new",
		UserID:      user.ID,
	}
	err = storage.CreateTask(context.Background(), task)
	require.NoError(t, err)

	retrievedTask, err := storage.GetTaskByID(context.Background(), task.ID)
	require.NoError(t, err)
	assert.Equal(t, task.Title, retrievedTask.Title)

	tasks, err := storage.GetTasks(context.Background(), user.ID)
	require.NoError(t, err)
	assert.Len(t, tasks, 1)

	task.Title = "Updated Integration Task"
	task.Status = "in_progress"
	err = storage.UpdateTask(context.Background(), task.ID, task)
	require.NoError(t, err)

	updatedTask, err := storage.GetTaskByID(context.Background(), task.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated Integration Task", updatedTask.Title)
	assert.Equal(t, "in_progress", updatedTask.Status)

	err = storage.DeleteTask(context.Background(), task.ID)
	require.NoError(t, err)

	retrievedUser, err := storage.GetUserByID(user.ID)
	require.NoError(t, err)
	assert.Equal(t, user.Username, retrievedUser.Username)

	retrievedUserByUsername, err := storage.GetUserByUsername(user.Username)
	require.NoError(t, err)
	assert.Equal(t, user.ID, retrievedUserByUsername.ID)

	user.Username = "updatedintegrationuser"
	user.Email = "updated@example.com"
	err = storage.UpdateUser(user.ID, user)
	require.NoError(t, err)

	updatedUser, err := storage.GetUserByID(user.ID)
	require.NoError(t, err)
	assert.Equal(t, "updatedintegrationuser", updatedUser.Username)
	assert.Equal(t, "updated@example.com", updatedUser.Email)

	err = storage.DeleteUser(user.ID)
	require.NoError(t, err)

	deletedUser, err := storage.GetUserByID(user.ID)
	assert.Error(t, err)
	assert.Nil(t, deletedUser)
}

func TestStorageEdgeCases(t *testing.T) {
	storage := setupTestDB(t)
	if storage == nil {
		return
	}
	defer func() {
		if err := storage.conn.Close(context.Background()); err != nil {
			t.Logf("Error closing connection: %v", err)
		}
	}()
	defer cleanupTestData(t, storage)

	user1 := &models.User{
		ID:       uuid.New().String(),
		Username: "duplicateuser",
		Email:    "user1@example.com",
		Password: "password123",
		Role:     "user",
	}
	err := storage.CreateUser(user1)
	require.NoError(t, err)

	user2 := &models.User{
		ID:       uuid.New().String(),
		Username: "duplicateuser",
		Email:    "user2@example.com",
		Password: "password456",
		Role:     "user",
	}
	err = storage.CreateUser(user2)
	assert.Error(t, err)

	user3 := &models.User{
		ID:       uuid.New().String(),
		Username: "differentuser",
		Email:    "user1@example.com",
		Password: "password789",
		Role:     "user",
	}
	err = storage.CreateUser(user3)
	assert.Error(t, err)
}

func TestStorageConcurrency(t *testing.T) {
	storage := setupTestDB(t)
	if storage == nil {
		return
	}
	defer func() {
		if err := storage.conn.Close(context.Background()); err != nil {
			t.Logf("Error closing connection: %v", err)
		}
	}()
	defer cleanupTestData(t, storage)

	user := &models.User{
		ID:       uuid.New().String(),
		Username: "concurrentuser",
		Email:    "concurrent@example.com",
		Password: "password123",
		Role:     "user",
	}
	err := storage.CreateUser(user)
	require.NoError(t, err)

	taskCount := 5
	for i := 0; i < taskCount; i++ {
		task := &models.Task{
			Title:       fmt.Sprintf("Concurrent Task %d", i),
			Description: fmt.Sprintf("Concurrent Description %d", i),
			Status:      "new",
			UserID:      user.ID,
		}
		err := storage.CreateTask(context.Background(), task)
		assert.NoError(t, err)
	}

	tasks, err := storage.GetTasks(context.Background(), user.ID)
	assert.NoError(t, err)
	assert.Len(t, tasks, taskCount)
}

func TestStorageInvalidData(t *testing.T) {
	storage := setupTestDB(t)
	if storage == nil {
		return
	}
	defer func() {
		if err := storage.conn.Close(context.Background()); err != nil {
			t.Logf("Error closing connection: %v", err)
		}
	}()
	defer cleanupTestData(t, storage)

	task := &models.Task{
		Title:       "Invalid Task",
		Description: "Invalid Description",
		Status:      "new",
		UserID:      "invalid-user-id",
	}
	err := storage.CreateTask(context.Background(), task)
	assert.Error(t, err)

	nonExistentTask := &models.Task{
		Title:       "Non-existent Task",
		Description: "Non-existent Description",
		Status:      "new",
	}
	err = storage.UpdateTask(context.Background(), "non-existent-id", nonExistentTask)
	assert.Error(t, err)

	err = storage.DeleteTask(context.Background(), "non-existent-id")
	assert.Error(t, err)
}

func TestStorageConnectionErrors(t *testing.T) {
	invalidStorage, err := NewStorage("invalid_connection_string")
	assert.Error(t, err)
	assert.Nil(t, invalidStorage)

	emptyStorage, err := NewStorage("")
	assert.Error(t, err)
	assert.Nil(t, emptyStorage)
}

func TestMigrationErrors(t *testing.T) {
	err := Migration("invalid_dsn", "../../migrations")
	assert.Error(t, err)

	err = Migration(testDBConnStr, "invalid_path")
	assert.Error(t, err)
}
