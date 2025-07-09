package main

import (
	"log"
	"project/internal/server"
	storage "project/repository/inmemory"
)

func main() {
	log.Println("To-Do API Service starting...")

	db := storage.NewStorage()
	api := server.NewLibraryAPI(db)
	if api == nil {
		log.Fatal("Failed to create API server")
	}

	log.Println("Server is running on :8080")
	log.Fatal(api.Start())
}
