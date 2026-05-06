# Milestones: Sentry-Go

## Phase 1: Foundations & Triage Graph (Goal: Automated Log/Commit Fetching)
- [ ] **1.1: Project Scaffolding:** Go module init, folder structure, and basic HTTP server.
- [ ] **1.2: Alert Receiver:** Implement `/api/v1/alerts` to parse Pub/Sub messages.
- [ ] **1.3: GCP Client Library Integration:** Implement the `Log Collector` node using `cloud.google.com/go/logging`.
- [ ] **1.4: GitHub API Integration:** Implement the `Commit Hunter` node to fetch recent commits.
- [ ] **1.5: ADK Graph Definition:** Define the workflow graph connecting collectors to the reasoning node.

## Phase 2: AI Reasoning & Briefing (Goal: Gemini-Powered Root Cause Analysis)
- [ ] **2.1: Prompt Engineering:** Design the "Triage Prompt" that combines logs, commits, and metrics.
- [ ] **2.2: Gemini Integration:** Use Go ADK to send context to `gemini-2.0-flash`.
- [ ] **2.3: Root Cause Deduction:** Implement logic to identify the most likely "breaking change."
- [ ] **2.4: Briefing Formatter:** Create a Markdown/Slack-ready summary of findings.

## Phase 3: Remediation & Human-in-the-Loop (Goal: Actionable Fixes)
- [ ] **3.1: Remediation Engine:** Agent suggests specific `gcloud` commands based on root cause.
- [ ] **3.2: Approval API:** Implement `/api/v1/incidents/{id}/actions` for human approval.
- [ ] **3.3: Execution Node:** Implement the final node in the graph that runs the approved command.
- [ ] **3.4: Cloud Run Deployment:** Deploy to GCP and wire up to a live Pub/Sub topic.

## Phase 4: Validation & Demo (Goal: Portfolio Ready)
- [ ] **4.1: End-to-End Tests:** Scripted scenarios (e.g., "The Bad Commit") to verify triage accuracy.
- [ ] **4.2: Documentation Finalization:** Complete README, Architecture, and API docs.
- [ ] **4.3: Demo Script:** Create a video or walkthrough showing a 60-second incident triage.
