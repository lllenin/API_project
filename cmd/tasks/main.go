package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"project/internal/server"
	db "project/repository/db"
	inmemory "project/repository/inmemory"
	"syscall"
	"time"
)

func InitializeRepositories(cfg *server.Config) (server.Repository, server.TaskRepository, error) {
	dbStorage, err := db.NewStorage(cfg.DBStr)
	if err != nil {
		log.Println("[WARN] Не удалось подключиться к БД, используем память:", err)
		inmem := inmemory.NewStorage()
		return inmem, inmem, nil
	}
	return dbStorage, dbStorage, nil
}

func RunMigrations(cfg *server.Config) error {
	migratePath := cfg.MigratePath
	if err := db.Migration(cfg.DBStr, migratePath); err != nil {
		return err
	}
	log.Println("[SUCCESS] Миграции применены успешно")
	return nil
}

type TaskAPIInterface interface {
	Start() error
	Shutdown(ctx context.Context) error
}

func StartServer(api TaskAPIInterface, cfg *server.Config) (chan os.Signal, chan error) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	serverErr := make(chan error, 1)
	go func() {
		log.Printf("Сервис запущен на %s:%d", cfg.Addr, cfg.Port)
		if err := api.Start(); err != nil {
			serverErr <- err
		}
	}()

	return sigChan, serverErr
}

func HandleShutdown(api TaskAPIInterface, sig os.Signal) error {
	log.Printf("[INFO] Получен сигнал %v, начинаем graceful shutdown...", sig)

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := api.Shutdown(shutdownCtx); err != nil {
		log.Printf("[ERROR] Ошибка при graceful shutdown: %v", err)
		return err
	}
	log.Println("[SUCCESS] Graceful shutdown выполнен успешно")
	return nil
}

func main() {
	log.Println("Запуск сервиса задач...")

	cfg := server.ReadConfig()

	if err := RunMigrations(cfg); err != nil {
		log.Fatalf("[ERROR] Ошибка применения миграций: %v", err)
	}

	userRepo, taskRepo, err := InitializeRepositories(cfg)
	if err != nil {
		log.Fatal("[ERROR] Не удалось инициализировать репозитории:", err)
	}

	api := server.NewTaskAPI(userRepo, taskRepo)
	if api == nil {
		log.Fatal("[ERROR] Не удалось инициализировать API")
	}

	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan, serverErr := StartServer(api, cfg)

	select {
	case sig := <-sigChan:
		HandleShutdown(api, sig)

	case err := <-serverErr:
		log.Printf("[ERROR] Ошибка сервера: %v", err)
		cancel()
	}

	log.Println("Сервис завершен")
}
