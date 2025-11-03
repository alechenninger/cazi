package main

import (
	"fmt"
	"log"
	"net/http"

	"widgets-service/application"
	"widgets-service/infrastructure"
	"widgets-service/presentation"
)

func main() {
	// Infrastructure layer
	repo := infrastructure.NewInMemoryWidgetRepository()
	authz := infrastructure.NewLocalAuthz()

	// Application layer
	widgetService := application.NewWidgetService(repo, authz)

	// Presentation layer
	handler := presentation.NewWidgetHandler(widgetService)

	// Setup HTTP routes
	http.HandleFunc("/widgets", handler.CreateWidget)
	http.HandleFunc("/widgets/", handler.GetWidget)

	// Start server
	port := 8080
	log.Printf("Starting widgets service on port %d", port)
	if err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
