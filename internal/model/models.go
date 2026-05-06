package model

import "time"

// PubSubMessage is the wrapper for the push payload
type PubSubMessage struct {
	Message struct {
		Data []byte `json:"data"`
	} `json:"message"`
}

// Alert represents the inner Cloud Monitoring schema
type Alert struct {
	Incident IncidentDetails `json:"incident"`
}

type IncidentDetails struct {
	IncidentID    string    `json:"incident_id"`
	ResourceID    string    `json:"resource_id"`
	ResourceName  string    `json:"resource_name"`
	State         string    `json:"state"`
	StartedAt     int64     `json:"started_at"`
	PolicyName    string    `json:"policy_name"`
	ConditionName string    `json:"condition_name"`
	URL           string    `json:"url"`
}

// Report represents the final triage analysis
type Report struct {
	IncidentID   string             `json:"incident_id" firestore:"incident_id"`
	Status       string             `json:"status" firestore:"status"`
	Summary      string             `json:"summary" firestore:"summary"`
	RootCause    string             `json:"root_cause" firestore:"root_cause"`
	Logs         []string           `json:"logs" firestore:"logs"`
	Commits      []Commit           `json:"commits" firestore:"commits"`
	Remediation  RemediationDetails `json:"remediation" firestore:"remediation"`
	TriageTime   time.Duration      `json:"triage_time" firestore:"triage_time"`
}

type Commit struct {
	SHA     string `json:"sha"`
	Author  string `json:"author"`
	Message string `json:"message"`
	URL     string `json:"url"`
}

type RemediationDetails struct {
	Suggestion string `json:"suggestion"`
	Command    string `json:"command"`
}
