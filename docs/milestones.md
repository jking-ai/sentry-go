# Milestones: Sentry-Go

## Phase 1: Foundations & Triage Graph (Goal: Automated Log/Commit Fetching)
- [x] **1.1: Project Scaffolding:** Go module init, folder structure, and basic HTTP server.
- [x] **1.2: Alert Receiver:** Implement `/api/v1/alerts` to parse Pub/Sub messages.
- [x] **1.3: GCP Client Library Integration:** Implement the `Log Collector` node using `cloud.google.com/go/logging`.
- [x] **1.4: GitHub API Integration:** Implement the `Commit Hunter` node to fetch recent commits.
- [x] **1.5: ADK Graph Definition:** Define the workflow graph connecting collectors to the reasoning node.

## Phase 2: AI Reasoning & Briefing (Goal: Gemini-Powered Root Cause Analysis)
- [x] **2.1: Prompt Engineering:** Design the "Triage Prompt" that combines logs, commits, and metrics.
- [x] **2.2: Gemini Integration:** Use Go ADK to send context to `gemini-3.1-flash-lite-preview`.
- [x] **2.3: Root Cause Deduction:** Implement logic to identify the most likely "breaking change."
- [x] **2.4: Briefing Formatter:** Create a Markdown/Slack-ready summary of findings.
- [x] **2.5: Metrics Integration:** Wire the `Metric Analyst` node into the triage prompt for CPU/memory correlation.

## Phase 3: Remediation & Human-in-the-Loop (Goal: Actionable Fixes)
- [x] **3.1: Remediation Engine:** Agent produces structured JSON with `remediation_suggestion` and `remediation_command` fields.
- [x] **3.2: Approval API:** Implement `POST /api/v1/incidents/{id}/actions` for human approval/rejection.
- [x] **3.3: Approval Persistence:** Store approval status, comment, and timestamp in Firestore.
- [x] **3.4: Circuit Breakers & Retry:** Implement `internal/resilience` package with exponential-backoff retry and circuit breaker for GCP/GitHub API calls.
- [ ] **3.5: Cloud Run Deployment:** Deploy to GCP and wire up to a live Pub/Sub topic.

## Phase 4: Validation & Demo (Goal: Portfolio Ready)
- [x] **4.1: Unit Tests:** Model, handler, agent (JSON parsing), and resilience package tests.
- [x] **4.2: Documentation Finalization:** Complete README, Architecture, and API docs.
- [ ] **4.3: Demo Script:** Create a video or walkthrough showing a 60-second incident triage.
- [ ] **4.4: End-to-End Integration Test:** Scripted scenarios (e.g., "The Bad Commit") to verify triage accuracy against a live environment.