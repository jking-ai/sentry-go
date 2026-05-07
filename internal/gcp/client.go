package gcp

import (
	"context"
	"fmt"
	"log"
	"time"

	"cloud.google.com/go/logging/logadmin"
	"github.com/jrk-ai-labs/sentry-go/internal/resilience"
	"google.golang.org/api/iterator"
)

type Client struct {
	ProjectID string
	LogAdmin  *logadmin.Client
	LogCB     *resilience.CircuitBreaker
	MetricsCB *resilience.CircuitBreaker
}

func NewClient(ctx context.Context, projectID string) (*Client, error) {
	admin, err := logadmin.NewClient(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to create log admin client: %v", err)
	}

	logCB := resilience.NewCircuitBreaker(5, 30*time.Second,
		resilience.WithStateChangeHook(func(from, to resilience.State) {
			log.Printf("[CircuitBreaker] Logging: %v → %v", stateName(from), stateName(to))
		}),
	)
	metricsCB := resilience.NewCircuitBreaker(5, 30*time.Second,
		resilience.WithStateChangeHook(func(from, to resilience.State) {
			log.Printf("[CircuitBreaker] Monitoring: %v → %v", stateName(from), stateName(to))
		}),
	)

	return &Client{
		ProjectID: projectID,
		LogAdmin:  admin,
		LogCB:     logCB,
		MetricsCB: metricsCB,
	}, nil
}

func (c *Client) FetchErrorLogs(ctx context.Context, resourceID string, lookback time.Duration) ([]string, error) {
	var result []string
	err := c.LogCB.Execute(ctx, func() error {
		fetched, fetchErr := resilience.Retry(ctx, resilience.DefaultMaxRetries, resilience.DefaultBaseDelay, resilience.DefaultMaxDelay, func() ([]string, error) {
			return fetchErrorLogsOnce(ctx, c, resourceID, lookback)
		})
		result = fetched
		return fetchErr
	})
	return result, err
}

func fetchErrorLogsOnce(ctx context.Context, c *Client, resourceID string, lookback time.Duration) ([]string, error) {
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

func stateName(s resilience.State) string {
	switch s {
	case resilience.Closed:
		return "CLOSED"
	case resilience.Open:
		return "OPEN"
	case resilience.HalfOpen:
		return "HALF_OPEN"
	default:
		return "UNKNOWN"
	}
}