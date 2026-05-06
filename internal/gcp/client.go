package gcp

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/logging/logadmin"
	"google.golang.org/api/iterator"
)

type Client struct {
	ProjectID string
	LogAdmin  *logadmin.Client
}

func NewClient(ctx context.Context, projectID string) (*Client, error) {
	admin, err := logadmin.NewClient(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to create log admin client: %v", err)
	}

	return &Client{
		ProjectID: projectID,
		LogAdmin:  admin,
	}, nil
}

func (c *Client) FetchErrorLogs(ctx context.Context, resourceID string, lookback time.Duration) ([]string, error) {
	filter := fmt.Sprintf(`resource.labels.service_name="%s" AND severity>=ERROR AND timestamp >= "%s"`,
		resourceID, time.Now().Add(-lookback).Format(time.RFC3339))

	it := c.LogAdmin.Entries(ctx, logadmin.Filter(filter))
	
	var logs []string
	for {
		entry, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to fetch log entry: %v", err)
		}
		
		payload, ok := entry.Payload.(string)
		if !ok {
			payload = fmt.Sprintf("%v", entry.Payload)
		}
		logs = append(logs, fmt.Sprintf("[%s] %s: %s", 
			entry.Timestamp.Format("15:04:05"), 
			entry.Severity, 
			payload))
		
		if len(logs) >= 50 {
			break
		}
	}

	return logs, nil
}
