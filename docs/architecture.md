# Architecture: Sentry-Go

## System Overview
Sentry-Go is a cloud-native incident response agent written in Go. It uses the **Google Agent Development Kit (ADK) v2.0** to orchestrate a graph-based triage workflow. The system is designed to be triggered by GCP Pub/Sub alerts and provides an automated "Incident Brief" to engineers.

## Tech Stack
- **Language:** Go 1.23+
- **Agent Framework:** Go ADK v2.0
- **LLM:** Gemini 2.0 Flash (via Vertex AI)
- **Deployment:** Google Cloud Run
- **Eventing:** Cloud Pub/Sub
- **Observability:** Cloud Logging, Cloud Monitoring
- **Version Control:** GitHub API (for commit correlation)

## Core Components

### 1. Alert Receiver (Entry Point)
A lightweight HTTP handler running on Cloud Run that accepts Pub/Sub push notifications. It parses the common alert schema and extracts the resource ID, project, and alert condition.

### 2. Triage Graph (ADK Workflow)
The heart of the system, defined as a directed acyclic graph (DAG) using ADK's `workflow` package.
- **Node: Log Collector:** Fetches the last 5 minutes of "ERROR" level logs for the affected resource.
- **Node: Commit Hunter:** Queries GitHub/GitLab for the latest commits to the service's repository.
- **Node: Metric Analyst:** Retrieves 15 minutes of relevant metrics (CPU, Memory, Request Rate) leading up to the incident.
- **Node: Reasoner (Gemini):** Analyzes the logs + commits + metrics to hypothesize the root cause.
- **Node: Remediation Engine:** Suggests a `gcloud` or `terraform` command to fix the issue.

### 3. Human-in-the-Loop Interface
A simple CLI or Slack integration where the agent posts its findings and waits for an engineer to signal `APPROVE` for any remediation actions.

## Data Flow
1. **Incident Occurs:** Cloud Monitoring fires an alert.
2. **Event Trigger:** Pub/Sub pushes the alert to Sentry-Go.
3. **Workflow Execution:**
   - Agent starts the Triage Graph.
   - Concurrent calls to Cloud Logging and GitHub.
   - Context is bundled into a prompt for Gemini.
4. **Briefing:** Agent posts the summary to Slack/Logs.
5. **Remediation:** If approved, agent executes the suggested fix via GCP SDK.

## Security & Reliability
- **Least Privilege:** Cloud Run service account limited to `logging.viewer`, `monitoring.viewer`, and specific resource-level `admin` roles.
- **Circuit Breakers:** Uses standard Go patterns to prevent cascading failures if GitHub or GCP APIs are throttled.
- **Deterministic Paths:** Critical triage steps are hard-coded in the graph; the LLM is only used for *reasoning* and *correlation*, not for control flow.
