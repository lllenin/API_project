package db

import (
    "context"
    "log"
    "project/internal/domain/errors"
    "project/internal/domain/models"
    "time"

    "github.com/google/uuid"
    "github.com/jackc/pgx/v5"
)

type Storage struct {
    conn                  *pgx.Conn
    prepCreateTask        string
    prepGetTaskByID       string
    prepGetTasks          string
    prepUpdateTask        string
    prepDeleteTask        string
    prepCreateUser        string
    prepGetUserByID       string
    prepGetUserByUsername string
    prepUpdateUser        string
    prepDeleteUser        string
    deleteQueue           chan struct{}
}

func NewStorage(connStr string) (*Storage, error) {
    ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
    defer cancel()
    conn, err := pgx.Connect(ctx, connStr)
    if err != nil {
        log.Println("[ERROR] Не удалось подключиться к базе данных:", err)
        return nil, err
    }

    s := &Storage{
        conn:                  conn,
        prepCreateTask:        `INSERT INTO tasks (id, title, description, status, user_id) VALUES ($1, $2, $3, $4, $5)`,
        prepGetTaskByID:       `SELECT id, title, description, status, user_id, deleted FROM tasks WHERE id = $1`,
        prepGetTasks:          `SELECT id, title, description, status, user_id FROM tasks WHERE user_id = $1 AND deleted = false`,
        prepUpdateTask:        `UPDATE tasks SET title = $1, description = $2, status = $3 WHERE id = $4`,
        prepDeleteTask:        `UPDATE tasks SET deleted = true WHERE id = $1 AND deleted = false`,
        prepCreateUser:        `INSERT INTO users (id, username, email, password, role) VALUES ($1, $2, $3, $4, $5)`,
        prepGetUserByID:       `SELECT id, username, email, password, role FROM users WHERE id = $1`,
        prepGetUserByUsername: `SELECT id, username, email, password, role FROM users WHERE username = $1`,
        prepUpdateUser:        `UPDATE users SET username = $1, email = $2, password = $3, role = $4 WHERE id = $5`,
        prepDeleteUser:        `DELETE FROM users WHERE id = $1`,
        deleteQueue:           make(chan struct{}, 10),
    }
    log.Println("[SUCCESS] Соединение с базой данных установлено успешно")
    return s, nil
}

func (s *Storage) CreateTask(ctx context.Context, task *models.Task) error {
    ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
    defer cancel()
    id := uuid.New().String()
    task.ID = id
    task.Deleted = false
    stmt, err := s.conn.Prepare(ctx, "create_task", s.prepCreateTask)
    if err != nil {
        log.Println("[ERROR] Не удалось подготовить запрос на создание задачи:", err)
        return err
    }
    _, err = s.conn.Exec(ctx, stmt.Name, task.ID, task.Title, task.Description, task.Status, task.UserID)
    if err != nil {
        log.Println("[ERROR] Не удалось создать задачу:", err)
        return errors.ErrConflict
    }
    log.Println("[SUCCESS] Задача успешно создана:", task.ID)
    return nil
}

func (s *Storage) GetTaskByID(ctx context.Context, id string) (*models.Task, error) {
    ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
    defer cancel()
    stmt, err := s.conn.Prepare(ctx, "get_task_by_id", s.prepGetTaskByID)
    if err != nil {
        log.Println("[ERROR] Не удалось подготовить запрос на получение задачи по ID:", err)
        return nil, err
    }
    row := s.conn.QueryRow(ctx, stmt.Name, id)
    task := &models.Task{}
    if err := row.Scan(&task.ID, &task.Title, &task.Description, &task.Status, &task.UserID, &task.Deleted); err != nil {
        if err == pgx.ErrNoRows {
            log.Println("[ERROR] Задача не найдена:", id)
            return nil, errors.ErrNotFound
        }
        log.Println("[ERROR] Ошибка при получении задачи:", err)
        return nil, err
    }
    log.Println("[SUCCESS] Задача найдена:", id)
    return task, nil
}

func (s *Storage) GetTasks(ctx context.Context, userID string) ([]models.Task, error) {
    ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
    defer cancel()
    stmt, err := s.conn.Prepare(ctx, "get_tasks", s.prepGetTasks)
    if err != nil {
        log.Println("[ERROR] Не удалось подготовить запрос на получение всех задач:", err)
        return nil, err
    }
    rows, err := s.conn.Query(ctx, stmt.Name, userID)
    if err != nil {
        log.Println("[ERROR] Не удалось получить задачи:", err)
        return nil, err
    }
    defer rows.Close()

    tasks := []models.Task{}
    for rows.Next() {
        task := models.Task{}
        if err := rows.Scan(&task.ID, &task.Title, &task.Description, &task.Status, &task.UserID); err != nil {
            log.Println("[ERROR] Ошибка при чтении задач:", err)
            return nil, err
        }
        tasks = append(tasks, task)
    }
    log.Println("[SUCCESS] Получено задач:", len(tasks))
    return tasks, nil
}

func (s *Storage) UpdateTask(ctx context.Context, id string, task *models.Task) error {
    ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
    defer cancel()
    stmt, err := s.conn.Prepare(ctx, "update_task", s.prepUpdateTask)
    if err != nil {
        log.Println("[ERROR] Не удалось подготовить запрос на обновление задачи:", err)
        return err
    }
    ct, err := s.conn.Exec(ctx, stmt.Name, task.Title, task.Description, task.Status, id)
    if err != nil {
        log.Println("[ERROR] Не удалось обновить задачу:", err)
        return err
    }
    if ct.RowsAffected() == 0 {
        log.Println("[ERROR] Задача для обновления не найдена:", id)
        return errors.ErrNotFound
    }
    log.Println("[SUCCESS] Задача успешно обновлена:", id)
    return nil
}

func (s *Storage) DeleteTask(ctx context.Context, id string) error {
    ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
    defer cancel()
    stmt, err := s.conn.Prepare(ctx, "delete_task_soft", s.prepDeleteTask)
    if err != nil {
        log.Println("[ERROR] Не удалось подготовить запрос на пометку задачи как удалённой:", err)
        return err
    }
    ct, err := s.conn.Exec(ctx, stmt.Name, id)
    if err != nil {
        log.Println("[ERROR] Не удалось пометить задачу как удалённую:", err)
        return err
    }
    if ct.RowsAffected() == 0 {
        log.Println("[ERROR] Задача для удаления не найдена:", id)
        return errors.ErrNotFound
    }
    log.Println("[SUCCESS] Задача помечена как удалённая:", id)
    s.tryEnqueueOrFlush()
    return nil
}

func (s *Storage) CreateUser(user *models.User) error {
    ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
    defer cancel()
    stmt, err := s.conn.Prepare(ctx, "create_user", s.prepCreateUser)
    if err != nil {
        log.Println("[ERROR] Не удалось подготовить запрос на создание пользователя:", err)
        return err
    }
    _, err = s.conn.Exec(ctx, stmt.Name, user.ID, user.Username, user.Email, user.Password, user.Role)
    if err != nil {
        log.Println("[ERROR] Не удалось создать пользователя:", err)
        return errors.ErrUserAlreadyExists
    }
    log.Println("[SUCCESS] Пользователь успешно создан:", user.ID)
    return nil
}

func (s *Storage) GetUserByID(id string) (*models.User, error) {
    ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
    defer cancel()
    stmt, err := s.conn.Prepare(ctx, "get_user_by_id", s.prepGetUserByID)
    if err != nil {
        log.Println("[ERROR] Не удалось подготовить запрос на получение пользователя по ID:", err)
        return nil, err
    }
    row := s.conn.QueryRow(ctx, stmt.Name, id)
    user := &models.User{}
    if err := row.Scan(&user.ID, &user.Username, &user.Email, &user.Password, &user.Role); err != nil {
        if err == pgx.ErrNoRows {
            log.Println("[ERROR] Пользователь не найден:", id)
            return nil, errors.ErrUserNotFound
        }
        log.Println("[ERROR] Ошибка при получении пользователя:", err)
        return nil, err
    }
    log.Println("[SUCCESS] Пользователь найден:", id)
    return user, nil
}

func (s *Storage) GetUserByUsername(username string) (*models.User, error) {
    ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
    defer cancel()
    stmt, err := s.conn.Prepare(ctx, "get_user_by_username", s.prepGetUserByUsername)
    if err != nil {
        log.Println("[ERROR] Не удалось подготовить запрос на получение пользователя по имени:", err)
        return nil, err
    }
    row := s.conn.QueryRow(ctx, stmt.Name, username)
    user := &models.User{}
    if err := row.Scan(&user.ID, &user.Username, &user.Email, &user.Password, &user.Role); err != nil {
        if err == pgx.ErrNoRows {
            log.Println("[ERROR] Пользователь не найден:", username)
            return nil, errors.ErrUserNotFound
        }
        log.Println("[ERROR] Ошибка при получении пользователя:", err)
        return nil, err
    }
    log.Println("[SUCCESS] Пользователь найден:", username)
    return user, nil
}

func (s *Storage) UpdateUser(id string, user *models.User) error {
    ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
    defer cancel()
    stmt, err := s.conn.Prepare(ctx, "update_user", s.prepUpdateUser)
    if err != nil {
        log.Println("[ERROR] Не удалось подготовить запрос на обновление пользователя:", err)
        return err
    }
    ct, err := s.conn.Exec(ctx, stmt.Name, user.Username, user.Email, user.Password, user.Role, id)
    if err != nil {
        log.Println("[ERROR] Не удалось обновить пользователя:", err)
        return err
    }
    if ct.RowsAffected() == 0 {
        log.Println("[ERROR] Пользователь для обновления не найден:", id)
        return errors.ErrUserNotFound
    }
    log.Println("[SUCCESS] Пользователь успешно обновлен:", id)
    return nil
}

func (s *Storage) DeleteUser(id string) error {
    ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
    defer cancel()
    stmt, err := s.conn.Prepare(ctx, "delete_user", s.prepDeleteUser)
    if err != nil {
        log.Println("[ERROR] Не удалось подготовить запрос на удаление пользователя:", err)
        return err
    }
    ct, err := s.conn.Exec(ctx, stmt.Name, id)
    if err != nil {
        log.Println("[ERROR] Не удалось удалить пользователя:", err)
        return err
    }
    if ct.RowsAffected() == 0 {
        log.Println("[ERROR] Пользователь для удаления не найден:", id)
        return errors.ErrUserNotFound
    }
    log.Println("[SUCCESS] Пользователь успешно удален:", id)
    return nil
}

func (s *Storage) EnqueueHardDelete(_ string) {
    s.tryEnqueueOrFlush()
}

func (s *Storage) tryEnqueueOrFlush() {
    if s.deleteQueue == nil {
        return
    }
    select {
    case s.deleteQueue <- struct{}{}:
    default:
        s.drainDeleteQueue()
        if affected, err := s.hardDeleteAllFlagged(context.Background()); err != nil {
            log.Println("[ERROR] Ошибка при удалении задач с признаком deleted:", err)
        } else if affected > 0 {
            log.Println("[SUCCESS] Жёстко удалено задач:", affected)
        }
    }
}

func (s *Storage) drainDeleteQueue() {
    if s.deleteQueue == nil {
        return
    }
    for {
        select {
        case <-s.deleteQueue:
        default:
            return
        }
    }
}

func (s *Storage) hardDeleteAllFlagged(ctx context.Context) (int64, error) {
    c, cancel := context.WithTimeout(ctx, 15*time.Second)
    defer cancel()
    tx, err := s.conn.Begin(c)
    if err != nil {
        return 0, err
    }
    ct, err := tx.Exec(c, `DELETE FROM tasks WHERE deleted = true`)
    if err != nil {
        _ = tx.Rollback(c)
        return 0, err
    }
    if err := tx.Commit(c); err != nil {
        return 0, err
    }
    return ct.RowsAffected(), nil
}

