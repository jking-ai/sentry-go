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
