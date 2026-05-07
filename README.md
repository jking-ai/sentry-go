# Sentry-Go

**Autonomous Incident Remediation Agent built with Go ADK v2.0.**

---

## One-line Summary
Sentry-Go is an AI-powered SRE assistant that autonomously triages cloud incidents by correlating logs, commits, and metrics to provide instant root-cause analysis and remediation suggestions.

## Problem Statement
On-call engineers often spend the first 30 minutes of an incident manually gathering data: checking logs, searching through recent GitHub commits, and correlating metrics. This "manual toil" increases Mean Time to Resolution (MTTR) and engineer burnout. Sentry-Go automates this triage phase, providing a "warmed up" incident report the moment the alert fires.

## Target User Persona
**Name:** Jordan, Senior Site Reliability Engineer (SRE)
- Responsible for high-availability cloud services.
- Tired of "triage fatigue" from high-volume alerts.
- Wants a tool that gives them the "why" before they even open their laptop.
- Values deterministic workflows over black-box AI responses.

## Skills and Engineering Patterns Showcased
| Pattern | Description |
|---------|-------------|
| **Graph-Based Workflows** | Implementing deterministic agent logic using Go ADK v2.0 Graphs. |
| **Deterministic Routing** | Controlling agent flow via Go code for reliable, repeatable triage steps. |
| **GCP Integration** | Utilizing Cloud Logging, Cloud Monitoring, and Firestore APIs at scale. |
| **Cloud-Native Go** | Building high-concurrency, memory-efficient services for incident response. |
| **Human-in-the-Loop** | Implementing approval gates for remediation actions. |
| **Resilience Patterns** | Circuit breakers and exponential-backoff retry for external API calls. |
| **Structured LLM Output** | Prompting Gemini for JSON responses with root cause + remediation. |

## Success Criteria
1. **Automated Triage:** When a simulated alert fires (via Pub/Sub), the agent successfully fetches relevant logs, commits, and metrics within 60 seconds.
2. **Root Cause Analysis:** The agent correctly identifies a "breaking change" (e.g., a specific commit or config change) in 80% of test scenarios.
3. **Remediation Suggestion:** The agent provides a valid, actionable remediation command (e.g., `gcloud compute instances reset`).
4. **Human Approval:** Engineers can approve or reject remediation via the dashboard or API.
5. **Deployable:** Runs on Google Cloud Run and responds to Pub/Sub triggers.
6. **Portfolio-Ready:** Complete with architecture docs, API specs, unit tests, and a scripted demo.

## Documentation
- [Architecture](docs/architecture.md)
- [API Contracts](docs/api-contracts.md)
- [Milestones](docs/milestones.md)
- [Production Deployment](docs/production-deployment.md)

## Quick Start

### Prerequisites
- Go 1.26+
- GCP project with Cloud Logging, Monitoring, Vertex AI, and Firestore APIs enabled
- GitHub personal access token (optional; for commit correlation)
- Node.js 22+ (for frontend)

### Backend
```bash
export GCP_PROJECT_ID=your-project-id
export GITHUB_TOKEN=ghp_...          # optional
export GITHUB_REPO=owner/repo        # optional

go run cmd/main.go
```

### Frontend
```bash
cd frontend
cp .env.example .env                # set PUBLIC_API_URL
npm install
npm run dev
```

### Docker
```bash
docker build -t sentry-go .
docker run -p 8080:8080 -e GCP_PROJECT_ID=your-project-id sentry-go
```

## Running Tests
```bash
go test ./internal/... -v
```