package github

import (
	"context"
	"fmt"
	"os"

	"github.com/google/go-github/v60/github"
)

type Client struct {
	GHClient *github.Client
}

func NewClient(ctx context.Context) (*Client, error) {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		// Log warning but return client for public repos
		return &Client{GHClient: github.NewClient(nil)}, nil
	}

	client := github.NewClient(nil).WithAuthToken(token)
	return &Client{GHClient: client}, nil
}

func (c *Client) FetchRecentCommits(ctx context.Context, owner, repo string, count int) (string, error) {
	opts := &github.CommitsListOptions{
		ListOptions: github.ListOptions{PerPage: count},
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
