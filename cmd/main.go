package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/jrk-ai-labs/sentry-go/internal/agent"
	"github.com/jrk-ai-labs/sentry-go/internal/handler"
)

func main() {
	ctx := context.Background()
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	projectID := os.Getenv("GCP_PROJECT_ID")
	if projectID == "" {
		log.Fatal("GCP_PROJECT_ID environment variable is required")
	}

	triageSvc, err := agent.NewTriageService(ctx, projectID)
	if err != nil {
		log.Fatalf("Failed to initialize triage service: %v", err)
	}

	alertHandler := handler.NewAlertHandler(triageSvc)

	// Routing
	http.HandleFunc("/api/v1/alerts", alertHandler.HandleAlert)
	http.HandleFunc("/api/v1/incidents", alertHandler.HandleGetReports)
	http.HandleFunc("/health", handler.HandleHealth)

	log.Printf("Sentry-Go (Autonomous Triage Agent) starting on port %s in project %s", port, projectID)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
