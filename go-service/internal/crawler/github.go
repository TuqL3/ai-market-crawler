package crawler

import (
	"context"
	"log"
	"time"

	"github.com/google/go-github/v84/github"
	"github.com/lukas/ai-aggregator/go-service/internal/helpers"
	"github.com/lukas/ai-aggregator/go-service/internal/models"
	"golang.org/x/time/rate"
)

type GithubCrawler struct {
	client  *github.Client
	limiter *rate.Limiter
}

func (g GithubCrawler) Crawl(ctx context.Context, since time.Time) ([]models.RawProblem, error) {
	var results []models.RawProblem
	seen := make(map[string]bool)
	queries := []string{
		"label:bug state:open",
		"label:\"help wanted\" state:open",
	}
	for _, query := range queries {
		page := 1
		for {
			if err := g.limiter.Wait(ctx); err != nil {
				return nil, err
			}
			log.Printf("[GitHubCrawler] query=%s page=%d", query, page)
			res, _, err := g.client.Search.Issues(ctx, query, &github.SearchOptions{
				Sort:  "updated",
				Order: "desc",
				ListOptions: github.ListOptions{
					PerPage: 100,
					Page:    page,
				},
			})
			if err != nil {
				return nil, err
			}
			if len(res.Issues) == 0 {
				log.Printf("[GitHubCrawler] query=%s page=%d no more issues", query, page)
				break
			}
			stop := false
			for _, issue := range res.Issues {
				if issue.GetUpdatedAt().Before(since) {
					stop = true
					break
				}
				problem := helpers.MapIssueToProblem(issue)
				if seen[problem.SourceID] {
					continue
				}
				seen[problem.SourceID] = true
				results = append(results, problem)
			}
			log.Printf("[GitHubCrawler] query=%s page=%d fetched=%d total=%d",
				query, page, len(res.Issues), len(results),
			)
			if stop {
				log.Printf("[GitHubCrawler] query=%s stop due to since condition", query)
				break
			}
			page++
		}
	}
	log.Printf("[GitHubCrawler] total results=%d", len(results))
	return results, nil
}

func (g GithubCrawler) Source() string {
	return "github"
}

func NewGithubCrawler(client *github.Client, limiter *rate.Limiter) Crawler {
	return &GithubCrawler{
		client:  client,
		limiter: limiter,
	}
}
