package handler

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/jrk-ai-labs/sentry-go/internal/agent"
	"github.com/jrk-ai-labs/sentry-go/internal/model"
)

type AlertHandler struct {
	TriageService *agent.TriageService
}

func NewAlertHandler(svc *agent.TriageService) *AlertHandler {
	return &AlertHandler{
		TriageService: svc,
	}
}

func (h *AlertHandler) HandleGetReports(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	if r.Method == http.MethodOptions {
		return
	}

	reports, err := h.TriageService.Firestore.GetReports(r.Context())
	if err != nil {
		log.Printf("Failed to fetch reports from Firestore: %v", err)
		http.Error(w, "Failed to fetch reports", http.StatusInternalServerError)
		return
	}
	if reports == nil {
		reports = []model.Report{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(reports)
}

func (h *AlertHandler) HandleAlert(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == http.MethodOptions {
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var push model.PubSubMessage
	if err := json.NewDecoder(r.Body).Decode(&push); err != nil {
		log.Printf("Error decoding push message: %v", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	var alert model.Alert
	if err := json.Unmarshal(push.Message.Data, &alert); err != nil {
		log.Printf("Error unmarshaling alert data: %v", err)
		w.WriteHeader(http.StatusOK)
		return
	}

	if alert.Incident.IncidentID == "" {
		log.Printf("Received malformed alert data: %s", string(push.Message.Data))
		w.WriteHeader(http.StatusOK)
		return
	}

	log.Printf("Triage started for incident: %s (%s)", alert.Incident.IncidentID, alert.Incident.PolicyName)

	go h.TriageService.StartTriage(context.Background(), alert)
	
	w.WriteHeader(http.StatusAccepted)
}

func HandleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
