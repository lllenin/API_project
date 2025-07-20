package storage

import (
	"context"
	"project/internal/domain/errors"
	"project/internal/domain/models"

	"github.com/google/uuid"
)

type Storage struct {
	users map[string]models.User
	tasks map[string]models.Task
}

func NewStorage() *Storage {
	return &Storage{
		users: make(map[string]models.User),
		tasks: make(map[string]models.Task),
	}
}

func (s *Storage) GetUserByID(id string) (*models.User, error) {
	user, exists := s.users[id]
	if !exists {
		return nil, errors.ErrUserNotFound
	}
	return &user, nil
}

func (s *Storage) GetUserByUsername(username string) (*models.User, error) {
	for _, user := range s.users {
		if user.Username == username {
			return &user, nil
		}
	}
	return nil, errors.ErrUserNotFound
}

func (s *Storage) CreateUser(user *models.User) error {
	for _, existingUser := range s.users {
		if existingUser.Username == user.Username {
			return errors.ErrUserAlreadyExists
		}
	}
	userID := uuid.New().String()
	user.ID = userID
	s.users[userID] = *user
	return nil
}

func (s *Storage) UpdateUser(id string, user *models.User) error {
	if _, exists := s.users[id]; !exists {
		return errors.ErrUserNotFound
	}
	s.users[id] = *user
	return nil
}

func (s *Storage) DeleteUser(id string) error {
	if _, exists := s.users[id]; !exists {
		return errors.ErrUserNotFound
	}
	delete(s.users, id)
	return nil
}

func (s *Storage) CreateTask(ctx context.Context, task *models.Task) error {
	return s.CreateTaskNoCtx(task)
}

func (s *Storage) GetTaskByID(ctx context.Context, id string) (*models.Task, error) {
	return s.GetTaskByIDNoCtx(id)
}

func (s *Storage) GetTasks(ctx context.Context, userID string) ([]models.Task, error) {
	return s.GetTasksByUserIDNoCtx(userID)
}

func (s *Storage) UpdateTask(ctx context.Context, id string, task *models.Task) error {
	return s.UpdateTaskNoCtx(id, task)
}

func (s *Storage) DeleteTask(ctx context.Context, id string) error {
	return s.DeleteTaskNoCtx(id)
}

func (s *Storage) CreateTaskNoCtx(task *models.Task) error {
	id := uuid.New().String()
	task.ID = id
	s.tasks[id] = *task
	return nil
}

func (s *Storage) GetTaskByIDNoCtx(id string) (*models.Task, error) {
	task, exists := s.tasks[id]
	if !exists {
		return nil, errors.ErrNotFound
	}
	return &task, nil
}

func (s *Storage) GetTasksByUserIDNoCtx(userID string) ([]models.Task, error) {
	var tasks []models.Task
	for _, t := range s.tasks {
		if t.UserID == userID {
			tasks = append(tasks, t)
		}
	}
	return tasks, nil
}

func (s *Storage) UpdateTaskNoCtx(id string, task *models.Task) error {
	if _, exists := s.tasks[id]; !exists {
		return errors.ErrNotFound
	}
	task.ID = id
	s.tasks[id] = *task
	return nil
}

func (s *Storage) DeleteTaskNoCtx(id string) error {
	if _, exists := s.tasks[id]; !exists {
		return errors.ErrNotFound
	}
	delete(s.tasks, id)
	return nil
}
