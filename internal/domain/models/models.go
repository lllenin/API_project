package models

type User struct {
	ID       string `json:"id" validate:"uuid"`
	Username string `json:"username" validate:"required,min=3,max=50,alphanum"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8,max=100,alphanum"`
	Role     string `json:"role" validate:"omitempty,oneof=user admin moderator"`
}

type LoginRequest struct {
	Username string `json:"username" validate:"required,min=3,max=50"`
	Password string `json:"password" validate:"required,min=6"`
}

type RegisterRequest struct {
	Username string `json:"username" validate:"required,min=3,max=50,alphanum"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6,max=100"`
	Role     string `json:"role" validate:"omitempty,oneof=user admin moderator"`
}

type UpdateUserRequest struct {
	Username string `json:"username" validate:"omitempty,min=3,max=50,alphanum"`
	Email    string `json:"email" validate:"omitempty,email"`
	Password string `json:"password" validate:"omitempty,min=6,max=100"`
	Role     string `json:"role" validate:"omitempty,oneof=user admin moderator"`
}

type Task struct {
	ID          string `json:"id" validate:"omitempty,uuid"`
	Title       string `json:"title" validate:"required,min=1,max=100"`
	Description string `json:"description" validate:"omitempty,max=500"`
	Status      string `json:"status" validate:"required,oneof=new in_progress done"`
}

type CreateTaskRequest struct {
	Title       string `json:"title" validate:"required,min=1,max=100"`
	Description string `json:"description" validate:"omitempty,max=500"`
}

type UpdateTaskRequest struct {
	Title       string `json:"title" validate:"omitempty,min=1,max=100"`
	Description string `json:"description" validate:"omitempty,max=500"`
	Status      string `json:"status" validate:"omitempty,oneof=new in_progress done"`
}
