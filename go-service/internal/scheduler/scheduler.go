package scheduler

import (
	"context"
	"log"
	"time"

	"github.com/lukas/ai-aggregator/go-service/internal/crawler"
	"github.com/lukas/ai-aggregator/go-service/internal/store"
	"github.com/robfig/cron/v3"
)

type Scheduler struct {
	cron     *cron.Cron
	crawlers []crawler.Crawler
	store    *store.Store
}

func NewScheduler(store *store.Store, crawlers []crawler.Crawler) *Scheduler {
	return &Scheduler{
		cron:     cron.New(),
		crawlers: crawlers,
		store:    store,
	}
}

func (s *Scheduler) Start() {
	for _, c := range s.crawlers {
		c := c
		_, err := s.cron.AddFunc("@every 30m", func() {
			s.runCrawler(context.Background(), c)
		})
		if err != nil {
			return
		}
	}
	s.cron.Start()
}

func (s *Scheduler) runCrawler(ctx context.Context, c crawler.Crawler) {
	crawlJob, err := s.store.CreateCrawlJob(ctx, c.Source())
	if err != nil {
		log.Printf("Error creating crawl job: %v", err)
		return
	}
	since := time.Now().Add(-24 * time.Hour)
	rawProblem, err := c.Crawl(ctx, since)
	if err != nil {
		err := s.store.FailCrawlJob(ctx, crawlJob.ID, err.Error())
		if err != nil {
			return
		}
		return
	}
	num, err := s.store.UpsertProblems(ctx, rawProblem)

	if err != nil {
		log.Printf("Error upserting problems: %v", err)
		return
	}
	err = s.store.CompleteCrawlJob(ctx, crawlJob.ID, num)
	if err != nil {
		return
	}
}

func (s *Scheduler) Stop() {
	s.cron.Stop()
}
