package gcp

import (
	"context"
	"fmt"
	"time"

	monitoring "cloud.google.com/go/monitoring/apiv3/v2"
	"cloud.google.com/go/monitoring/apiv3/v2/monitoringpb"
	"github.com/jrk-ai-labs/sentry-go/internal/resilience"
	"google.golang.org/api/iterator"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (c *Client) FetchMetrics(ctx context.Context, resourceID string, lookback time.Duration) (string, error) {
	var result string

	err := c.MetricsCB.Execute(ctx, func() error {
		// Create the MetricClient once per FetchMetrics call (not per retry attempt).
		metricClient, clientErr := monitoring.NewMetricClient(ctx)
		if clientErr != nil {
			return clientErr
		}
		defer metricClient.Close()

		fetched, fetchErr := resilience.Retry(ctx, resilience.DefaultMaxRetries, resilience.DefaultBaseDelay, resilience.DefaultMaxDelay, func() (string, error) {
			return listTimeSeries(ctx, metricClient, c.ProjectID, resourceID, lookback)
		})
		result = fetched
		return fetchErr
	})

	return result, err
}

// listTimeSeries performs a single attempt to fetch CPU utilization metrics.
func listTimeSeries(ctx context.Context, metricClient *monitoring.MetricClient, projectID, resourceID string, lookback time.Duration) (string, error) {
	filter := fmt.Sprintf(`resource.type="cloud_run_revision" AND resource.labels.service_name="%s" AND metric.type="run.googleapis.com/container/cpu/utilizations"`, resourceID)

	req := &monitoringpb.ListTimeSeriesRequest{
		Name:   "projects/" + projectID,
		Filter: filter,
		Interval: &monitoringpb.TimeInterval{
			StartTime: timestamppb.New(time.Now().Add(-lookback)),
			EndTime:   timestamppb.New(time.Now()),
		},
	}

	it := metricClient.ListTimeSeries(ctx, req)

	summary := "Metric: CPU Utilization\n"
	for {
		resp, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return "", err
		}

		for _, point := range resp.Points {
			summary += fmt.Sprintf("[%s] Value: %.2f\n",
				time.Unix(point.Interval.EndTime.Seconds, 0).Format("15:04:05"),
				point.Value.GetDoubleValue())
		}
	}

	return summary, nil
}