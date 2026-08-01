package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/jrk-ai-labs/sentry-go/internal/gcp"
	"github.com/jrk-ai-labs/sentry-go/internal/github"
	"github.com/jrk-ai-labs/sentry-go/internal/model"
	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/model/gemini"
	"google.golang.org/adk/runner"
	"google.golang.org/adk/session"
	"google.golang.org/genai"
)

type TriageService struct {
	ProjectID    string
	GCPClient    *gcp.Client
	Firestore    *gcp.FirestoreClient
	GitHubClient *github.Client
	Runner       *runner.Runner
}

func NewTriageService(ctx context.Context, projectID string) (*TriageService, error) {
	gcpClient, err := gcp.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}

	fsClient, err := gcp.NewFirestoreClient(ctx, projectID)
	if err != nil {
		return nil, err
	}

	ghClient, err := github.NewClient(ctx)
	if err != nil {
		return nil, err
	}

	m, err := gemini.NewModel(ctx, "gemini-3.1-flash-lite-preview", &genai.ClientConfig{
		Project:  projectID,
		Location: "us-central1",
		Backend:  genai.BackendVertexAI,
	})
	if err != nil {
		return nil, err
	}

	reasoner, err := llmagent.New(llmagent.Config{
		Name:        "sentry-reasoner",
		Model:       m,
		Description: "Expert SRE that analyzes logs, metrics, and commits to find root causes and suggest remediation.",
		Instruction: `You are an expert Site Reliability Engineer producing a structured incident triage report for a dashboard.

Strict output rules (apply to every response):
- Output ONLY valid JSON with no additional text, markdown, or code fences.
- The JSON must have exactly these keys:
  - "root_cause": 2-4 sentences of plain prose identifying the most likely root cause and the supporting signal (log line, commit, or metric anomaly).
  - "remediation_suggestion": 1-2 sentences describing the recommended fix.
  - "remediation_command": a single executable gcloud or terraform command string (no backticks, no explanation).

The user prompt will declare a Mode: LIVE or SIMULATION.

LIVE mode:
- Analyze the provided telemetry. Cite the specific log line, commit, or metric condition that supports your hypothesis.
- If logs and commits are empty, state the most likely cause categories (recent deploy regression, resource saturation, dependency/upstream outage, configuration drift) and explicitly note the data was unavailable.

SIMULATION mode:
- Produce a concrete, plausible root cause for demo purposes. Invent realistic specifics: a short commit SHA, a believable error type or signal name, a timestamp, a pod or revision identifier, etc. Match the resource and policy in the prompt.
- Write as if you had analyzed real telemetry. Do NOT mention that data was unavailable, that you are simulating, that details are invented, or qualify with "hypothetical" / "likely" wording about data origin.
- The remediation_command should be a realistic gcloud command matching the scenario.`,
	})
	if err != nil {
		return nil, err
	}

	r, err := runner.New(runner.Config{
		AppName:           "Sentry-Go",
		Agent:             reasoner,
		SessionService:    session.InMemoryService(),
		AutoCreateSession: true,
	})
	if err != nil {
		return nil, err
	}

	return &TriageService{
		ProjectID:    projectID,
		GCPClient:    gcpClient,
		Firestore:     fsClient,
		GitHubClient: ghClient,
		Runner:       r,
	}, nil
}

func (s *TriageService) StartTriage(ctx context.Context, alert model.Alert) {
	log.Printf("[Triage] Starting workflow for incident %s", alert.Incident.IncidentID)
	startTime := time.Now()

	// --- Collect data concurrently ---
	type logsResult struct {
		logs []string
		err  error
	}
	type commitsResult struct {
		summary string
		err     error
	}
	type metricsResult struct {
		summary string
		err     error
	}

	logsCh := make(chan logsResult, 1)
	commitsCh := make(chan commitsResult, 1)
	metricsCh := make(chan metricsResult, 1)

	go func() {
		logs, err := s.GCPClient.FetchErrorLogs(ctx, alert.Incident.ResourceID, 15*time.Minute)
		logsCh <- logsResult{logs, err}
	}()

	go func() {
		summary := "No commit data available."
		ghRepo := os.Getenv("GITHUB_REPO")
		if ghRepo != "" && strings.Contains(ghRepo, "/") {
			parts := strings.Split(ghRepo, "/")
			s, err := s.GitHubClient.FetchRecentCommits(ctx, parts[0], parts[1], 10)
			if err != nil {
				log.Printf("[Triage] Error fetching commits: %v", err)
				commitsCh <- commitsResult{summary, err}
				return
			}
			summary = s
		}
		commitsCh <- commitsResult{summary, nil}
	}()

	go func() {
		summary, err := s.GCPClient.FetchMetrics(ctx, alert.Incident.ResourceID, 15*time.Minute)
		if err != nil {
			log.Printf("[Triage] Error fetching metrics: %v", err)
		}
		if summary == "" {
			summary = "(no metrics available)"
		}
		metricsCh <- metricsResult{summary, err}
	}()

	lr := <-logsCh
	cr := <-commitsCh
	mr := <-metricsCh

	if lr.err != nil {
		log.Printf("[Triage] Error fetching logs: %v", lr.err)
	}
	if mr.err != nil {
		log.Printf("[Triage] Error fetching metrics: %v", mr.err)
	}

	logsSection := "(no error logs found in the lookback window)"
	if len(lr.logs) > 0 {
		logsSection = strings.Join(lr.logs, "\n")
	}

	metricsSection := "(no metrics data available)"
	if mr.summary != "" && mr.summary != "(no metrics available)" {
		metricsSection = mr.summary
	}

	mode := "LIVE"
	if strings.Contains(strings.ToLower(alert.Incident.PolicyName), "simulation") {
		mode = "SIMULATION"
	}

	prompt := fmt.Sprintf(`Mode: %s
Produce a root-cause hypothesis and remediation for the incident below. Output ONLY valid JSON with keys: root_cause, remediation_suggestion, remediation_command.

Incident Policy: %s
Condition: %s
Resource: %s

Recent Error Logs (last 15m):
%s

Recent Commits:
%s

Resource Metrics (last 15m):
%s`, mode, alert.Incident.PolicyName, alert.Incident.ConditionName, alert.Incident.ResourceID, logsSection, cr.summary, metricsSection)

	events := s.Runner.Run(ctx, "system", alert.Incident.IncidentID, &genai.Content{
		Parts: []*genai.Part{{Text: prompt}},
	}, agent.RunConfig{})

	var finalResponse string
	for event, err := range events {
		if err != nil {
			log.Printf("[Triage] Error during agent run: %v", err)
			return
		}
		if event.Content != nil && event.Content.Role == "model" {
			for _, part := range event.Content.Parts {
				finalResponse += part.Text
			}
		}
	}

	// Parse structured JSON from Gemini
	var analysis struct {
		RootCause             string `json:"root_cause"`
		RemediationSuggestion string `json:"remediation_suggestion"`
		RemediationCommand    string `json:"remediation_command"`
	}

	cleaned := cleanJSONResponse(finalResponse)
	report := model.Report{
		IncidentID: alert.Incident.IncidentID,
		Status:     "COMPLETED",
		Summary:    "Triage analysis completed.",
		RootCause:  strings.TrimSpace(finalResponse), // fallback
		Logs:       lr.logs,
		TriageTime: time.Since(startTime),
	}

	if err := json.Unmarshal([]byte(cleaned), &analysis); err == nil {
		report.RootCause = strings.TrimSpace(analysis.RootCause)
		report.Remediation = model.RemediationDetails{
			Suggestion: strings.TrimSpace(analysis.RemediationSuggestion),
			Command:    strings.TrimSpace(analysis.RemediationCommand),
		}
	} else {
		log.Printf("[Triage] Could not parse structured JSON from Gemini, using raw response as root cause: %v", err)
		report.Remediation = model.RemediationDetails{
			Suggestion: "Manual review required — agent could not produce structured remediation.",
			Command:    "",
		}
	}

	log.Printf("[Triage] Report generated for %s (root cause: %s)", report.IncidentID, truncate(report.RootCause, 80))

	if err := s.Firestore.SaveReport(ctx, report); err != nil {
		log.Printf("[Triage] Error saving report to Firestore: %v", err)
	}
}

// cleanJSONResponse strips markdown code fences and surrounding whitespace
// that LLMs sometimes wrap around JSON output.
func cleanJSONResponse(s string) string {
	s = strings.TrimSpace(s)
	// Remove ```json ... ``` wrappers
	if strings.HasPrefix(s, "```json") {
		s = strings.TrimPrefix(s, "```json")
		s = strings.TrimSuffix(s, "```")
		s = strings.TrimSpace(s)
	} else if strings.HasPrefix(s, "```") {
		s = strings.TrimPrefix(s, "```")
		s = strings.TrimSuffix(s, "```")
		s = strings.TrimSpace(s)
	}
	return s
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}