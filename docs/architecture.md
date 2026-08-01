# Architecture: Sentry-Go

## System Overview
Sentry-Go is a cloud-native incident response agent written in Go. It uses the **Google Agent Development Kit (ADK) v2.0** to orchestrate a graph-based triage workflow. The system is designed to be triggered by GCP Pub/Sub alerts and provides an automated "Incident Brief" — including root cause, remediation suggestion, and an executable fix command — to engineers.

## Tech Stack
- **Language:** Go 1.26+
- **Agent Framework:** Go ADK v2.0 (`google.golang.org/adk`)
- **LLM:** Gemini 3.1 Flash-Lite (via Vertex AI)
- **Deployment:** Google Cloud Run
- **Eventing:** Cloud Pub/Sub
- **Observability:** Cloud Logging, Cloud Monitoring
- **Version Control:** GitHub API (for commit correlation)
- **Persistence:** Firestore (incident reports, approval state)
- **Frontend:** Astro + Tailwind CSS (hosted on Firebase Hosting)

## Core Components

### 1. Alert Receiver (Entry Point)
A lightweight HTTP handler running on Cloud Run that accepts Pub/Sub push notifications. It parses the common alert schema and extracts the resource ID, project, and alert condition.

### 2. Triage Service (ADK Workflow Orchestration)
The heart of the system. When an alert arrives, the `TriageService` concurrently dispatches three data collectors:
- **Log Collector:** Fetches the last 15 minutes of `ERROR`-level logs for the affected resource via Cloud Logging.
- **Commit Hunter:** Queries GitHub for the latest 10 commits to the service's repository.
- **Metric Analyst:** Retrieves 15 minutes of CPU utilization time series via Cloud Monitoring.

Once all collectors return, their output is combined into a structured prompt and sent through the **ADK Runner** to the **Gemini Reasoner**, which produces a JSON response containing:
- `root_cause`: 2–4 sentence analysis of the most likely cause.
- `remediation_suggestion`: Description of the recommended fix.
- `remediation_command`: A `gcloud` or `terraform` command to execute the fix.

The response is parsed and persisted to Firestore as an incident `Report`.

### 3. Human-in-the-Loop Approval API
After triage completes, engineers can review the remediation suggestion via the dashboard and either **APPROVE** or **REJECT** the proposed command:
- `GET /api/v1/incidents/{id}` — Retrieve a single incident report.
- `POST /api/v1/incidents/{id}/actions` — Submit approval/rejection with a comment.

Approval state is persisted in Firestore alongside the report.

### 4. Frontend Dashboard
An Astro + Tailwind CSS single-page app hosted on Firebase Hosting. Features:
- Live incident feed with auto-refresh (30s).
- Root cause and remediation display per incident card.
- Approve/Reject buttons for pending remediations.
- "Trigger Simulation" button to fire a demo alert.
- Average triage time stats.

### 5. Resilience Layer (`internal/resilience`)
Two reliability patterns protect the system from cascading failures:
- **Exponential Backoff Retry:** All external API calls (GCP Logging, GCP Monitoring, GitHub) are wrapped in `Retry[T]()` with configurable max attempts, base delay, max delay, and full jitter.
- **Circuit Breaker:** Each external dependency has a dedicated `CircuitBreaker` that trips open after consecutive failures and probes for recovery after a cooldown period. State transitions are logged for observability.

## Data Flow
1. **Incident Occurs:** Cloud Monitoring fires an alert.
2. **Event Trigger:** Pub/Sub pushes the alert to Sentry-Go.
3. **Workflow Execution (concurrent):**
   - Agent starts the Triage Graph.
   - Concurrent calls to Cloud Logging, GitHub, and Cloud Monitoring (each with retry + circuit breaker).
   - Context is bundled into a structured JSON prompt for Gemini.
4. **Gemini Reasoning:** The LLM returns structured JSON with root_cause, remediation_suggestion, and remediation_command.
5. **Briefing:** Report is saved to Firestore and surfaced on the dashboard.
6. **Approval:** Engineer reviews and approves/rejects the remediation via the dashboard or API.

## Security & Reliability
- **Least Privilege:** Cloud Run service account limited to `logging.viewer`, `monitoring.viewer`, and specific resource-level `admin` roles.
- **Circuit Breakers:** Each external API client has a circuit breaker (threshold: 5 failures, cooldown: 30s) to prevent cascading failures when upstream services are degraded.
- **Retry with Backoff:** Full-jitter exponential backoff (3 retries, 500ms base, 10s max) on all external calls.
- **Deterministic Paths:** Critical triage steps are hard-coded in Go; the LLM is only used for *reasoning* and *correlation*, not for control flow.
- **XSS Protection:** Frontend escapes all server-provided strings before rendering.