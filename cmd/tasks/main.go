package main

import (
	"log"
	"project/internal/server"
	db "project/repository/db"
	inmemory "project/repository/inmemory"
)

func main() {
	log.Println("Сервис задач запускается...")

	cfg := server.ReadConfig()

	migratePath := "/app/migrations"
	if err := db.Migration(cfg.DBStr, migratePath); err != nil {
		log.Fatalf("[ERROR] Ошибка применения миграции: %v", err)
	}
	log.Println("[SUCCESS] Миграции применены успешно.")

	var userRepo server.Repository
	var taskRepo server.TaskRepository

	dbStorage, err := db.NewStorage(cfg.DBStr)
	if err != nil {
		log.Println("[WARN] Не удалось подключиться к БД, переключаюсь на inmemory хранилище:", err)
		inmem := inmemory.NewStorage()
		userRepo = inmem
		taskRepo = inmem
	} else {
		userRepo = dbStorage
		taskRepo = dbStorage
	}

	api := server.NewTaskAPI(userRepo, taskRepo)
	if api == nil {
		log.Fatal("[ERROR] Не удалось создать API сервер")
	}

	log.Println("Сервер запущен на :8080")
	log.Fatal(api.Start())
}
