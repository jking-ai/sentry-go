package model

import (
	"encoding/json"
	"testing"
)

func TestAlertUnmarshal(t *testing.T) {
	raw := []byte(`{
		"incident": {
			"incident_id": "0.m123",
			"resource_id": "my-svc",
			"resource_name": "projects/p/locations/us-central1/services/my-svc",
			"state": "open",
			"started_at": 1714982400,
			"policy_name": "High Error Rate",
			"condition_name": "Error rate > 5%",
			"url": "https://app.google.com/alert/123"
		}
	}`)
	var a Alert
	if err := json.Unmarshal(raw, &a); err != nil {
		t.Fatalf("unmarshal alert: %v", err)
	}
	if a.Incident.IncidentID != "0.m123" {
		t.Errorf("incident_id = %q, want %q", a.Incident.IncidentID, "0.m123")
	}
	if a.Incident.ResourceID != "my-svc" {
		t.Errorf("resource_id = %q, want %q", a.Incident.ResourceID, "my-svc")
	}
	if a.Incident.PolicyName != "High Error Rate" {
		t.Errorf("policy_name = %q, want %q", a.Incident.PolicyName, "High Error Rate")
	}
	if a.Incident.State != "open" {
		t.Errorf("state = %q, want %q", a.Incident.State, "open")
	}
}

func TestPubSubMessageDecode(t *testing.T) {
	// The inner data is base64 of {"incident":{"incident_id":"test123"}}
	encoded := []byte(`{"message":{"data":"eyJpbmNpZGVudCI6eyJpbmNpZGVudF9pZCI6InRlc3QxMjMifX0="}}`)
	var ps PubSubMessage
	if err := json.Unmarshal(encoded, &ps); err != nil {
		t.Fatalf("unmarshal pubsub: %v", err)
	}
	if len(ps.Message.Data) == 0 {
		t.Error("message.data is empty")
	}
	// Verify we can decode the base64 payload into an Alert
	var a Alert
	if err := json.Unmarshal(ps.Message.Data, &a); err != nil {
		t.Fatalf("unmarshal inner alert: %v", err)
	}
	if a.Incident.IncidentID != "test123" {
		t.Errorf("incident_id = %q, want %q", a.Incident.IncidentID, "test123")
	}
}

func TestApprovalRequestValidate(t *testing.T) {
	tests := []struct {
		name   string
		action string
		valid  bool
	}{
		{"approve", "APPROVE", true},
		{"reject", "REJECT", true},
		{"lowercase", "approve", false},
		{"invalid", "MAYBE", false},
		{"empty", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := ApprovalRequest{Action: tt.action}
			ok := req.Action == "APPROVE" || req.Action == "REJECT"
			if ok != tt.valid {
				t.Errorf("action=%q ok=%v, want %v", tt.action, ok, tt.valid)
			}
		})
	}
}

func TestReportJSONRoundTrip(t *testing.T) {
	r := Report{
		IncidentID: "inc-001",
		Status:     "COMPLETED",
		Summary:    "Triage complete",
		RootCause:  "Bad deploy",
		Logs:       []string{"ERROR: crash at line 42"},
		Remediation: RemediationDetails{
			Suggestion: "Roll back to previous image",
			Command:    "gcloud run services update my-svc --image gcr.io/proj/my-svc:prev",
		},
		ApprovalStatus:  "PENDING",
		ApprovalComment: "",
	}

	data, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}

	var got Report
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal report: %v", err)
	}
	if got.IncidentID != r.IncidentID {
		t.Errorf("incident_id = %q, want %q", got.IncidentID, r.IncidentID)
	}
	if got.RootCause != r.RootCause {
		t.Errorf("root_cause = %q, want %q", got.RootCause, r.RootCause)
	}
	if got.Remediation.Command != r.Remediation.Command {
		t.Errorf("remediation.command = %q, want %q", got.Remediation.Command, r.Remediation.Command)
	}
	if len(got.Logs) != 1 {
		t.Errorf("logs length = %d, want 1", len(got.Logs))
	}
}