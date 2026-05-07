package github

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	gh "github.com/google/go-github/v60/github"
	"github.com/jrk-ai-labs/sentry-go/internal/resilience"
)

type Client struct {
	GHClient *gh.Client
	CB       *resilience.CircuitBreaker
}

func NewClient(ctx context.Context) (*Client, error) {
	token := os.Getenv("GITHUB_TOKEN")
	var ghClient *gh.Client
	if token == "" {
		ghClient = gh.NewClient(nil)
		log.Println("[GitHub] No GITHUB_TOKEN set; using unauthenticated client (rate-limited)")
	} else {
		ghClient = gh.NewClient(nil).WithAuthToken(token)
	}

	cb := resilience.NewCircuitBreaker(5, 30*time.Second,
		resilience.WithStateChangeHook(func(from, to resilience.State) {
			log.Printf("[CircuitBreaker] GitHub: %v → %v", stateName(from), stateName(to))
		}),
	)

	return &Client{GHClient: ghClient, CB: cb}, nil
}

func (c *Client) FetchRecentCommits(ctx context.Context, owner, repo string, count int) (string, error) {
	var result string
	err := c.CB.Execute(ctx, func() error {
		fetched, fetchErr := resilience.Retry(ctx, resilience.DefaultMaxRetries, resilience.DefaultBaseDelay, resilience.DefaultMaxDelay, func() (string, error) {
			return fetchRecentCommitsOnce(ctx, c, owner, repo, count)
		})
		result = fetched
		return fetchErr
	})
	return result, err
}

func fetchRecentCommitsOnce(ctx context.Context, c *Client, owner, repo string, count int) (string, error) {
	opts := &gh.CommitsListOptions{
		ListOptions: gh.ListOptions{PerPage: count},
	}
	commits, resp, err := c.GHClient.Repositories.ListCommits(ctx, owner, repo, opts)
	if err != nil {
		if resp != nil && resp.StatusCode == 404 {
			return "Repository not found or access denied.", nil
		}
		return "", fmt.Errorf("failed to list commits: %v", err)
	}

	summary := fmt.Sprintf("Recent Commits for %s/%s:\n", owner, repo)
	for _, commit := range commits {
		msg := commit.GetCommit().GetMessage()
		if len(msg) > 100 {
			msg = msg[:97] + "..."
		}

		authorName := "Unknown"
		if author := commit.GetCommit().GetAuthor(); author != nil {
			authorName = author.GetName()
			if authorName == "" {
				authorName = author.GetEmail()
			}
		}

		summary += fmt.Sprintf("- [%s] %s: %s\n",
			commit.GetSHA()[:7],
			authorName,
			msg)
	}

	return summary, nil
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