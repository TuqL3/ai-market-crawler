package crawler

import (
	"context"
	"time"

	"github.com/lukas/ai-aggregator/go-service/internal/models"
)

type Crawler interface {
	Crawl(ctx context.Context, since time.Time) ([]models.RawProblem, error)
	Source() string
}
