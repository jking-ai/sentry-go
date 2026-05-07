package gcp

import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"
	"github.com/jrk-ai-labs/sentry-go/internal/model"
)

type FirestoreClient struct {
	Client *firestore.Client
}

func NewFirestoreClient(ctx context.Context, projectID string) (*FirestoreClient, error) {
	client, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to create firestore client: %v", err)
	}
	return &FirestoreClient{Client: client}, nil
}

func (c *FirestoreClient) SaveReport(ctx context.Context, report model.Report) error {
	_, err := c.Client.Collection("incidents").Doc(report.IncidentID).Set(ctx, report)
	return err
}

func (c *FirestoreClient) GetReports(ctx context.Context) ([]model.Report, error) {
	var reports []model.Report
	docs, err := c.Client.Collection("incidents").OrderBy("triage_time", firestore.Desc).Limit(20).Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}
	for _, doc := range docs {
		var r model.Report
		if err := doc.DataTo(&r); err != nil {
			continue
		}
		reports = append(reports, r)
	}
	return reports, nil
}

// GetReport fetches a single incident report by ID.
func (c *FirestoreClient) GetReport(ctx context.Context, incidentID string) (*model.Report, error) {
	doc, err := c.Client.Collection("incidents").Doc(incidentID).Get(ctx)
	if err != nil {
		return nil, err
	}
	var r model.Report
	if err := doc.DataTo(&r); err != nil {
		return nil, err
	}
	return &r, nil
}

// UpdateApproval sets the approval status on an incident. The action must be
// "APPROVE" or "REJECT". Returns the updated report or an error.
func (c *FirestoreClient) UpdateApproval(ctx context.Context, incidentID, action, comment string) (*model.Report, error) {
	ref := c.Client.Collection("incidents").Doc(incidentID)

	var status string
	switch action {
	case "APPROVE":
		status = "APPROVED"
	case "REJECT":
		status = "REJECTED"
	default:
		return nil, fmt.Errorf("invalid action %q: must be APPROVE or REJECT", action)
	}

	updates := []firestore.Update{
		{Path: "approval_status", Value: status},
		{Path: "approval_comment", Value: comment},
		{Path: "approved_at", Value: firestore.ServerTimestamp},
	}
	if _, err := ref.Update(ctx, updates); err != nil {
		return nil, err
	}

	// Read back the updated document
	doc, err := ref.Get(ctx)
	if err != nil {
		return nil, err
	}
	var r model.Report
	if err := doc.DataTo(&r); err != nil {
		return nil, err
	}
	return &r, nil
}