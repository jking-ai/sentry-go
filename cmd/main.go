package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"

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

	// Single-incident and approval routes require path-based routing.
	// /api/v1/incidents/{id}        → GET a single incident
	// /api/v1/incidents/{id}/actions → POST approval/rejection
	http.HandleFunc("/api/v1/incidents/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if strings.HasSuffix(path, "/actions") {
			alertHandler.HandleIncidentAction(w, r)
			return
		}
		// Exact match on /api/v1/incidents (no trailing content) → list
		// Otherwise treat as /api/v1/incidents/{id}
		if path == "/api/v1/incidents/" || path == "/api/v1/incidents" {
			alertHandler.HandleGetReports(w, r)
			return
		}
		alertHandler.HandleGetIncident(w, r)
	})

	log.Printf("Sentry-Go (Autonomous Triage Agent) starting on port %s in project %s", port, projectID)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}