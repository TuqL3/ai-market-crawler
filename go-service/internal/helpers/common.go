package helpers

import (
	"fmt"

	"github.com/google/go-github/v84/github"
	"github.com/lukas/ai-aggregator/go-service/internal/models"
)

func MapIssueToProblem(issue *github.Issue) models.RawProblem {
	var tags []string
	for _, label := range issue.Labels {
		tags = append(tags, label.GetName())
	}

	return models.RawProblem{
		Source:        "github",
		SourceID:      fmt.Sprintf("%d", issue.GetID()),
		Title:         issue.GetTitle(),
		Body:          issue.GetBody(),
		URL:           issue.GetHTMLURL(),
		Score:         issue.GetComments(),
		CommentCount:  issue.GetComments(),
		SourceCreated: issue.GetCreatedAt().Time,
		Tags:          tags,
	}
}
