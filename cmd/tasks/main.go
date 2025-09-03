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

func main() {
	log.Println("Запуск сервиса задач...")

	cfg := server.ReadConfig()

	migratePath := cfg.MigratePath
	if err := db.Migration(cfg.DBStr, migratePath); err != nil {
		log.Fatalf("[ERROR] Ошибка применения миграций: %v", err)
	}
	log.Println("[SUCCESS] Миграции применены успешно")

	var userRepo server.Repository
	var taskRepo server.TaskRepository

	dbStorage, err := db.NewStorage(cfg.DBStr)
	if err != nil {
		log.Println("[WARN] Не удалось подключиться к БД, используем память:", err)
		inmem := inmemory.NewStorage()
		userRepo = inmem
		taskRepo = inmem
	} else {
		userRepo = dbStorage
		taskRepo = dbStorage
	}

	api := server.NewTaskAPI(userRepo, taskRepo)
	if api == nil {
		log.Fatal("[ERROR] Не удалось инициализировать API")
	}

	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	serverErr := make(chan error, 1)
	go func() {
		log.Printf("Сервис запущен на %s:%d", cfg.Addr, cfg.Port)
		if err := api.Start(); err != nil {
			serverErr <- err
		}
	}()

	select {
	case sig := <-sigChan:
		log.Printf("[INFO] Получен сигнал %v, начинаем graceful shutdown...", sig)

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer shutdownCancel()

		if err := api.Shutdown(shutdownCtx); err != nil {
			log.Printf("[ERROR] Ошибка при graceful shutdown: %v", err)
		} else {
			log.Println("[SUCCESS] Graceful shutdown выполнен успешно")
		}

	case err := <-serverErr:
		log.Printf("[ERROR] Ошибка сервера: %v", err)
		cancel()
	}

	log.Println("Сервис завершен")
}
