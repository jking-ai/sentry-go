# Production Deployment: Sentry-Go

## GCP Resource Inventory

| Resource | Name | Description |
|----------|------|-------------|
| Project | `<your-gcp-project-id>` | Primary host project |
| Service Account | `sentry-go-sa` | Identity for the Cloud Run service |
| Pub/Sub Topic | `sentry-go-alerts` | Topic for receiving incident alerts |
| Cloud Run | `sentry-go` | (To be deployed) Containerized Go service |

## Enabled APIs
- `run.googleapis.com` (Cloud Run)
- `pubsub.googleapis.com` (Pub/Sub)
- `logging.googleapis.com` (Cloud Logging)
- `monitoring.googleapis.com` (Cloud Monitoring)
- `aiplatform.googleapis.com` (Vertex AI)

## IAM Roles Assigned to `sentry-go-sa`
- `roles/logging.viewer`: To fetch error logs for triage.
- `roles/monitoring.viewer`: To fetch metrics for correlation.
- `roles/aiplatform.user`: To call Gemini 3.1 Flash-Lite via Vertex AI.
- `roles/run.viewer`: To inspect Cloud Run service state.

## Deployment Instructions

### 1. Build & Push Image
```bash
# Set your artifact registry path
PROJECT_ID=<your-gcp-project-id>
IMAGE_URL=us-central1-docker.pkg.dev/$PROJECT_ID/apps/sentry-go:latest
docker build -t $IMAGE_URL .
docker push $IMAGE_URL
```

### 2. Deploy to Cloud Run
```bash
gcloud run deploy sentry-go \
    --image $IMAGE_URL \
    --service-account sentry-go-sa@$PROJECT_ID.iam.gserviceaccount.com \
    --region us-central1 \
    --allow-unauthenticated \
    --set-env-vars="GCP_PROJECT_ID=$PROJECT_ID"
```

### 3. Configure Pub/Sub Push
```bash
gcloud pubsub subscriptions create sentry-go-sub \
    --topic sentry-go-alerts \
    --push-endpoint=https://<your-cloud-run-url>/api/v1/alerts
```
