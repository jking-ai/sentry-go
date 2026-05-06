# API Contracts: Sentry-Go

## Alert Ingestion (Internal/PubSub)

### POST `/api/v1/alerts`
Receiver for Cloud Monitoring Pub/Sub notifications.

**Request Body (Cloud Monitoring Schema):**
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

### GET `/api/v1/incidents/{id}`
Retrieve the current status and findings of an incident triage.

**Response:**
```json
{
  "id": "0.m123456789",
  "status": "COMPLETED",
  "findings": {
    "summary": "High error rate likely caused by recent commit.",
    "root_cause": "NullPointerException in UserProfileService.java introduced in commit 8f2a1b",
    "logs": ["... ERROR: null pointer at line 42 ..."],
    "commits": [
      {
        "sha": "8f2a1b",
        "author": "dev-alpha",
        "message": "Refactor user profile loading"
      }
    ],
    "remediation": {
      "suggestion": "Roll back to commit 7e1b2a or reset instance.",
      "command": "gcloud run services update my-service --image gcr.io/my-project/my-service:7e1b2a"
    }
  }
}
```

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

**Response:**
- `200 OK`: Action executed.
- `403 Forbidden`: User not authorized to approve.

---

## Health Check

### GET `/health`
Check service and ADK readiness.
