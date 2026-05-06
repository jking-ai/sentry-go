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
| **GCP Integration** | Utilizing Cloud Logging and Cloud Monitoring APIs at scale. |
| **Cloud-Native Go** | Building high-concurrency, memory-efficient services for incident response. |
| **Human-in-the-Loop** | Implementing approval gates for remediation actions. |

## Success Criteria
1. **Automated Triage:** When a simulated alert fires (via Pub/Sub), the agent successfully fetches relevant logs and recent commits within 60 seconds.
2. **Root Cause Analysis:** The agent correctly identifies a "breaking change" (e.g., a specific commit or config change) in 80% of test scenarios.
3. **Remediation Suggestion:** The agent provides a valid, actionable remediation command (e.g., `gcloud compute instances reset`).
4. **Deployable:** Runs on Google Cloud Run and responds to Pub/Sub triggers.
5. **Portfolio-Ready:** Complete with architecture docs, API specs, and a scripted demo.

## Documentation
- [Architecture](docs/architecture.md)
- [API Contracts](docs/api-contracts.md)
- [Milestones](docs/milestones.md)
