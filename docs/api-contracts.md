# API Contracts: Sentry-Go

## Alert Ingestion (Internal/PubSub)

### POST `/api/v1/alerts`
Receiver for Cloud Monitoring Pub/Sub notifications.

**Request Body (Cloud Monitoring Schema):**
```json
{
  "message": {
    "data": "<base64-encoded alert JSON>"
  }
}
```

**Decoded Alert:**
```json
{
  "incident": {
    "incident_id": "0.m123456789",
    "resource_id": "my-service-instance",
    "resource_name": "projects/my-project/locations/us-central1/services/my-service",
    "state": "open",
    "started_at": 1714982400,
    "policy_name": "High Error Rate",
    "condition_name": "Error rate > 5%"
  }
}
```

**Response:**
- `202 Accepted`: Workflow started.
- `400 Bad Request`: Invalid alert schema.

---

## Agent Status & Reporting

### GET `/api/v1/incidents`
List the 20 most recent incident reports ordered by triage time (descending).

**Response:**
```json
[
  {
    "incident_id": "0.m123456789",
    "status": "COMPLETED",
    "summary": "Triage analysis completed.",
    "root_cause": "NullPointerException in UserProfileService.java introduced in commit 8f2a1b triggered a cascade of 500s across the us-central1 revision. CPU utilization spiked to 95% seconds before the alert fired.",
    "logs": ["... ERROR: null pointer at line 42 ..."],
    "commits": [],
    "remediation": {
      "suggestion": "Roll back to the previous Cloud Run revision to restore service.",
      "command": "gcloud run services update my-service --image gcr.io/my-project/my-service:7e1b2a"
    },
    "triage_time": 4200000000,
    "approval_status": "PENDING",
    "approval_comment": "",
    "approved_at": "0001-01-01T00:00:00Z"
  }
]
```

---

### GET `/api/v1/incidents/{id}`
Retrieve a single incident report by ID.

**Response:** Same shape as a single item from the list above.

- `200 OK`: Report found.
- `404 Not Found`: Incident ID does not exist.

---

## Human-in-the-Loop Actions

### POST `/api/v1/incidents/{id}/actions`
Submit an approval or rejection for a remediation command.

**Request Body:**
```json
{
  "action": "APPROVE",
  "comment": "Proceed with rollback."
}
```

**Action values:** `APPROVE` or `REJECT` (case-sensitive).

**Response:**
- `200 OK`: Action recorded. Returns the updated report.
- `400 Bad Request`: Invalid action or malformed body.
- `404 Not Found`: Incident ID does not exist.
- `500 Internal Server Error`: Failed to update.

**Response body (updated report):**
```json
{
  "incident_id": "0.m123456789",
  "status": "COMPLETED",
  "root_cause": "...",
  "remediation": { "suggestion": "...", "command": "..." },
  "approval_status": "APPROVED",
  "approval_comment": "Proceed with rollback.",
  "approved_at": "2025-05-06T20:00:00Z"
}
```

---

## Health Check

### GET `/health`
Check service readiness.

**Response:**
- `200 OK`: Body contains `OK`.