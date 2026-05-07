package handler

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"path"
	"strings"

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

func corsHeaders(w http.ResponseWriter, methods string) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", methods)
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

// GET /api/v1/incidents — list all reports
func (h *AlertHandler) HandleGetReports(w http.ResponseWriter, r *http.Request) {
	corsHeaders(w, "GET, OPTIONS")
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
	writeJSON(w, http.StatusOK, reports)
}

// GET /api/v1/incidents/{id} — single report
func (h *AlertHandler) HandleGetIncident(w http.ResponseWriter, r *http.Request) {
	corsHeaders(w, "GET, OPTIONS")
	if r.Method == http.MethodOptions {
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := path.Base(r.URL.Path)
	report, err := h.TriageService.Firestore.GetReport(r.Context(), id)
	if err != nil {
		log.Printf("Failed to fetch incident %s: %v", id, err)
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "incident not found"})
		return
	}
	writeJSON(w, http.StatusOK, report)
}

// POST /api/v1/incidents/{id}/actions — approve or reject remediation
func (h *AlertHandler) HandleIncidentAction(w http.ResponseWriter, r *http.Request) {
	corsHeaders(w, "POST, OPTIONS")
	if r.Method == http.MethodOptions {
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Expect path like /api/v1/incidents/{id}/actions
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v1/incidents/"), "/")
	if len(parts) != 2 || parts[1] != "actions" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid path; expected /api/v1/incidents/{id}/actions"})
		return
	}
	incidentID := parts[0]

	var req model.ApprovalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}

	req.Action = strings.ToUpper(req.Action)
	if req.Action != "APPROVE" && req.Action != "REJECT" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "action must be APPROVE or REJECT"})
		return
	}

	// Verify the incident exists
	if _, err := h.TriageService.Firestore.GetReport(r.Context(), incidentID); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "incident not found"})
		return
	}

	report, err := h.TriageService.Firestore.UpdateApproval(r.Context(), incidentID, req.Action, req.Comment)
	if err != nil {
		log.Printf("Failed to update approval for %s: %v", incidentID, err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to update approval"})
		return
	}

	log.Printf("[Approval] Incident %s: action=%s comment=%q", incidentID, req.Action, req.Comment)
	writeJSON(w, http.StatusOK, report)
}

// POST /api/v1/alerts — ingest Pub/Sub alert
func (h *AlertHandler) HandleAlert(w http.ResponseWriter, r *http.Request) {
	corsHeaders(w, "POST, OPTIONS")
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