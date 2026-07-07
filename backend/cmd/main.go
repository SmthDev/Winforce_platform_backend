package main

import (
	"log"
	"os"

	"platform/backend/internal/routes"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	r := routes.NewRouter()

	log.Printf("starting server on :%s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
