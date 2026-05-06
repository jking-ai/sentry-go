package agent

import (
	"context"
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

	m, err := gemini.NewModel(ctx, "gemini-2.5-flash", &genai.ClientConfig{
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
		Description: "Expert SRE that analyzes logs, metrics, and commits to find root causes.",
		Instruction: `You are an expert Site Reliability Engineer producing a one-shot incident triage report for a dashboard card.

Strict output rules (apply to every response):
- Output ONLY the root-cause analysis. Do not greet, acknowledge, or ask the user any questions.
- Never request additional data.
- Respond in plain prose, 2-4 sentences. No markdown headers, no bullet lists, no code fences.
- Lead with the most likely root cause, then briefly cite the supporting signal.

The user prompt will declare a Mode: LIVE or SIMULATION.

LIVE mode:
- Analyze the provided telemetry. Cite the specific log line, commit, or condition that supports your hypothesis.
- If logs and commits are empty, state the most likely cause categories (recent deploy regression, resource saturation, dependency/upstream outage, configuration drift) and explicitly note the data was unavailable.

SIMULATION mode:
- Produce a concrete, plausible root cause for demo purposes. Invent realistic specifics: a short commit SHA, a believable error type or signal name, a timestamp, a pod or revision identifier, etc. Match the resource and policy in the prompt.
- Write as if you had analyzed real telemetry. Do NOT mention that data was unavailable, that you are simulating, that details are invented, or qualify with "hypothetical" / "likely" wording about data origin.
- Still obey the 2-4 sentence plain-prose limit.`,
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
		Firestore:    fsClient,
		GitHubClient: ghClient,
		Runner:       r,
	}, nil
}

func (s *TriageService) StartTriage(ctx context.Context, alert model.Alert) {
	log.Printf("[Triage] Starting workflow for incident %s", alert.Incident.IncidentID)
	startTime := time.Now()

	logs, err := s.GCPClient.FetchErrorLogs(ctx, alert.Incident.ResourceID, 15*time.Minute)
	if err != nil {
		log.Printf("[Triage] Error fetching logs: %v", err)
	}

	commitsSummary := "No commit data available."
	ghRepo := os.Getenv("GITHUB_REPO")
	if ghRepo != "" && strings.Contains(ghRepo, "/") {
		parts := strings.Split(ghRepo, "/")
		summary, err := s.GitHubClient.FetchRecentCommits(ctx, parts[0], parts[1], 10)
		if err != nil {
			log.Printf("[Triage] Error fetching commits: %v", err)
		} else {
			commitsSummary = summary
		}
	}

	logsSection := "(no error logs found in the lookback window)"
	if len(logs) > 0 {
		logsSection = strings.Join(logs, "\n")
	}

	mode := "LIVE"
	if strings.Contains(strings.ToLower(alert.Incident.PolicyName), "simulation") {
		mode = "SIMULATION"
	}

	prompt := fmt.Sprintf(`Mode: %s
Produce a root-cause hypothesis for the incident below. Respond with the analysis only — no preamble, no questions.

Incident Policy: %s
Condition: %s
Resource: %s

Recent Error Logs (last 15m):
%s

Recent Commits:
%s`, mode, alert.Incident.PolicyName, alert.Incident.ConditionName, alert.Incident.ResourceID, logsSection, commitsSummary)

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

	report := model.Report{
		IncidentID: alert.Incident.IncidentID,
		Status:     "COMPLETED",
		Summary:    "Triage analysis completed.",
		RootCause:  strings.TrimSpace(finalResponse),
		Logs:       logs,
		TriageTime: time.Since(startTime),
	}

	log.Printf("[Triage] Report generated for %s", report.IncidentID)

	if err := s.Firestore.SaveReport(ctx, report); err != nil {
		log.Printf("[Triage] Error saving report to Firestore: %v", err)
	}
}
