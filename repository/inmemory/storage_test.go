package storage

import (
	"project/internal/domain/models"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewStorage(t *testing.T) {
	tests := []struct {
		name string
		want struct {
			notNil     bool
			hasUsers   bool
			hasTasks   bool
			emptyUsers bool
			emptyTasks bool
		}
	}{
		{
			name: "create new storage instance",
			want: struct {
				notNil     bool
				hasUsers   bool
				hasTasks   bool
				emptyUsers bool
				emptyTasks bool
			}{
				notNil:     true,
				hasUsers:   true,
				hasTasks:   true,
				emptyUsers: true,
				emptyTasks: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := NewStorage()

			assert.Equal(t, tt.want.notNil, storage != nil)
			assert.Equal(t, tt.want.hasUsers, storage.users != nil)
			assert.Equal(t, tt.want.hasTasks, storage.tasks != nil)
			assert.Equal(t, tt.want.emptyUsers, len(storage.users) == 0)
			assert.Equal(t, tt.want.emptyTasks, len(storage.tasks) == 0)
		})
	}
}

func TestStorageCreateUser(t *testing.T) {
	tests := []struct {
		name string
		user *models.User
		want struct {
			error bool
		}
		setup func(*Storage)
	}{
		{
			name: "successful user creation",
			user: &models.User{
				Username: "testuser",
				Email:    "test@example.com",
				Password: "password123",
				Role:     "user",
			},
			want: struct {
				error bool
			}{
				error: false,
			},
			setup: func(s *Storage) {
			},
		},
		{
			name: "duplicate username",
			user: &models.User{
				Username: "testuser",
				Email:    "test2@example.com",
				Password: "password456",
				Role:     "user",
			},
			want: struct {
				error bool
			}{
				error: true,
			},
			setup: func(s *Storage) {
				s.users["user1"] = models.User{
					ID:       "user1",
					Username: "testuser",
					Email:    "test@example.com",
					Password: "password123",
					Role:     "user",
				}
			},
		},
		{
			name: "duplicate email",
			user: &models.User{
				Username: "testuser2",
				Email:    "test@example.com",
				Password: "password456",
				Role:     "user",
			},
			want: struct {
				error bool
			}{
				error: false,
			},
			setup: func(s *Storage) {
				s.users["existinguser"] = models.User{
					ID:       "user1",
					Username: "existinguser",
					Email:    "test@example.com",
					Password: "password123",
					Role:     "user",
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := NewStorage()
			tt.setup(storage)

			err := storage.CreateUser(tt.user)

			if tt.want.error {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, tt.user.ID)
			}
		})
	}
}

func TestStorageGetUserByID(t *testing.T) {
	tests := []struct {
		name   string
		userID string
		want   struct {
			error bool
			found bool
		}
		setup func(*Storage)
	}{
		{
			name:   "successful user retrieval",
			userID: "user1",
			want: struct {
				error bool
				found bool
			}{
				error: false,
				found: true,
			},
			setup: func(s *Storage) {
				s.users["user1"] = models.User{
					ID:       "user1",
					Username: "testuser",
					Email:    "test@example.com",
					Password: "password123",
					Role:     "user",
				}
			},
		},
		{
			name:   "user not found",
			userID: "nonexistent",
			want: struct {
				error bool
				found bool
			}{
				error: true,
				found: false,
			},
			setup: func(s *Storage) {
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := NewStorage()
			tt.setup(storage)

			user, err := storage.GetUserByID(tt.userID)

			if tt.want.error {
				assert.Error(t, err)
				assert.Nil(t, user)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, user)
				assert.Equal(t, tt.userID, user.ID)
			}
		})
	}
}

func TestStorageGetUserByUsername(t *testing.T) {
	tests := []struct {
		name     string
		username string
		want     struct {
			error bool
			found bool
		}
		setup func(*Storage)
	}{
		{
			name:     "successful user retrieval",
			username: "testuser",
			want: struct {
				error bool
				found bool
			}{
				error: false,
				found: true,
			},
			setup: func(s *Storage) {
				s.users["user1"] = models.User{
					ID:       "user1",
					Username: "testuser",
					Email:    "test@example.com",
					Password: "password123",
					Role:     "user",
				}
			},
		},
		{
			name:     "user not found",
			username: "nonexistent",
			want: struct {
				error bool
				found bool
			}{
				error: true,
				found: false,
			},
			setup: func(s *Storage) {
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := NewStorage()
			tt.setup(storage)

			user, err := storage.GetUserByUsername(tt.username)

			if tt.want.error {
				assert.Error(t, err)
				assert.Nil(t, user)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, user)
				assert.Equal(t, tt.username, user.Username)
			}
		})
	}
}

func TestStorageUpdateUser(t *testing.T) {
	tests := []struct {
		name   string
		userID string
		user   *models.User
		want   struct {
			error bool
		}
		setup func(*Storage)
	}{
		{
			name:   "successful user update",
			userID: "user1",
			user: &models.User{
				Username: "updateduser",
				Email:    "updated@example.com",
				Password: "newpassword",
				Role:     "admin",
			},
			want: struct {
				error bool
			}{
				error: false,
			},
			setup: func(s *Storage) {
				s.users["user1"] = models.User{
					ID:       "user1",
					Username: "testuser",
					Email:    "test@example.com",
					Password: "password123",
					Role:     "user",
				}
			},
		},
		{
			name:   "user not found",
			userID: "nonexistent",
			user: &models.User{
				Username: "updateduser",
				Email:    "updated@example.com",
				Password: "newpassword",
				Role:     "admin",
			},
			want: struct {
				error bool
			}{
				error: true,
			},
			setup: func(s *Storage) {
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := NewStorage()
			tt.setup(storage)

			err := storage.UpdateUser(tt.userID, tt.user)

			if tt.want.error {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestStorageDeleteUser(t *testing.T) {
	tests := []struct {
		name   string
		userID string
		want   struct {
			error bool
		}
		setup func(*Storage)
	}{
		{
			name:   "successful user deletion",
			userID: "user1",
			want: struct {
				error bool
			}{
				error: false,
			},
			setup: func(s *Storage) {
				s.users["user1"] = models.User{
					ID:       "user1",
					Username: "testuser",
					Email:    "test@example.com",
					Password: "password123",
					Role:     "user",
				}
			},
		},
		{
			name:   "user not found",
			userID: "nonexistent",
			want: struct {
				error bool
			}{
				error: true,
			},
			setup: func(s *Storage) {
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := NewStorage()
			tt.setup(storage)

			err := storage.DeleteUser(tt.userID)

			if tt.want.error {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestStorageCreateTaskNoCtx(t *testing.T) {
	tests := []struct {
		name string
		task *models.Task
		want struct {
			error bool
		}
		setup func(*Storage)
	}{
		{
			name: "successful task creation",
			task: &models.Task{
				Title:       "Test Task",
				Description: "Test Description",
				Status:      "new",
				UserID:      "user1",
			},
			want: struct {
				error bool
			}{
				error: false,
			},
			setup: func(s *Storage) {
			},
		},
		{
			name: "task with empty title",
			task: &models.Task{
				Title:       "",
				Description: "Test Description",
				Status:      "new",
				UserID:      "user1",
			},
			want: struct {
				error bool
			}{
				error: false,
			},
			setup: func(s *Storage) {
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := NewStorage()
			tt.setup(storage)

			err := storage.CreateTaskNoCtx(tt.task)

			if tt.want.error {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, tt.task.ID)
				assert.False(t, tt.task.Deleted)
			}
		})
	}
}

func TestStorageGetTaskByIDNoCtx(t *testing.T) {
	tests := []struct {
		name   string
		taskID string
		want   struct {
			error bool
			found bool
		}
		setup func(*Storage)
	}{
		{
			name:   "successful task retrieval",
			taskID: "task1",
			want: struct {
				error bool
				found bool
			}{
				error: false,
				found: true,
			},
			setup: func(s *Storage) {
				s.tasks["task1"] = models.Task{
					ID:          "task1",
					Title:       "Test Task",
					Description: "Test Description",
					Status:      "new",
					UserID:      "user1",
					Deleted:     false,
				}
			},
		},
		{
			name:   "task not found",
			taskID: "nonexistent",
			want: struct {
				error bool
				found bool
			}{
				error: true,
				found: false,
			},
			setup: func(s *Storage) {
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := NewStorage()
			tt.setup(storage)

			task, err := storage.GetTaskByIDNoCtx(tt.taskID)

			if tt.want.error {
				assert.Error(t, err)
				assert.Nil(t, task)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, task)
				assert.Equal(t, tt.taskID, task.ID)
			}
		})
	}
}

func TestStorageGetTasksByUserIDNoCtx(t *testing.T) {
	tests := []struct {
		name   string
		userID string
		want   struct {
			error bool
			count int
		}
		setup func(*Storage)
	}{
		{
			name:   "successful tasks retrieval",
			userID: "user1",
			want: struct {
				error bool
				count int
			}{
				error: false,
				count: 2,
			},
			setup: func(s *Storage) {
				s.tasks["task1"] = models.Task{
					ID:          "task1",
					Title:       "Task 1",
					Description: "Description 1",
					Status:      "new",
					UserID:      "user1",
					Deleted:     false,
				}
				s.tasks["task2"] = models.Task{
					ID:          "task2",
					Title:       "Task 2",
					Description: "Description 2",
					Status:      "in_progress",
					UserID:      "user1",
					Deleted:     false,
				}
			},
		},
		{
			name:   "no tasks found",
			userID: "user2",
			want: struct {
				error bool
				count int
			}{
				error: false,
				count: 0,
			},
			setup: func(s *Storage) {
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := NewStorage()
			tt.setup(storage)

			tasks, err := storage.GetTasksByUserIDNoCtx(tt.userID)

			if tt.want.error {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, tasks, tt.want.count)
			}
		})
	}
}

func TestStorageUpdateTaskNoCtx(t *testing.T) {
	tests := []struct {
		name   string
		taskID string
		task   *models.Task
		want   struct {
			error bool
		}
		setup func(*Storage)
	}{
		{
			name:   "successful task update",
			taskID: "task1",
			task: &models.Task{
				Title:       "Updated Task",
				Description: "Updated Description",
				Status:      "in_progress",
			},
			want: struct {
				error bool
			}{
				error: false,
			},
			setup: func(s *Storage) {
				s.tasks["task1"] = models.Task{
					ID:          "task1",
					Title:       "Original Task",
					Description: "Original Description",
					Status:      "new",
					UserID:      "user1",
					Deleted:     false,
				}
			},
		},
		{
			name:   "task not found",
			taskID: "nonexistent",
			task: &models.Task{
				Title: "Updated Task",
			},
			want: struct {
				error bool
			}{
				error: true,
			},
			setup: func(s *Storage) {
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := NewStorage()
			tt.setup(storage)

			err := storage.UpdateTaskNoCtx(tt.taskID, tt.task)

			if tt.want.error {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestStorageDeleteTaskNoCtx(t *testing.T) {
	tests := []struct {
		name   string
		taskID string
		want   struct {
			error bool
		}
		setup func(*Storage)
	}{
		{
			name:   "successful task deletion",
			taskID: "task1",
			want: struct {
				error bool
			}{
				error: false,
			},
			setup: func(s *Storage) {
				s.tasks["task1"] = models.Task{
					ID:          "task1",
					Title:       "Test Task",
					Description: "Test Description",
					Status:      "new",
					UserID:      "user1",
					Deleted:     false,
				}
			},
		},
		{
			name:   "task not found",
			taskID: "nonexistent",
			want: struct {
				error bool
			}{
				error: true,
			},
			setup: func(s *Storage) {
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := NewStorage()
			tt.setup(storage)

			err := storage.DeleteTaskNoCtx(tt.taskID)

			if tt.want.error {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
