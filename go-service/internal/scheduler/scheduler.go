package scheduler

import (
	"context"
	"log"
	"time"

	"github.com/google/uuid"
	pb "github.com/lukas/ai-aggregator/go-service/gen/aggregator/v1"
	"github.com/lukas/ai-aggregator/go-service/internal/crawler"
	"github.com/lukas/ai-aggregator/go-service/internal/grpcclient"
	"github.com/lukas/ai-aggregator/go-service/internal/models"
	"github.com/lukas/ai-aggregator/go-service/internal/store"
	"github.com/robfig/cron/v3"
)

type Scheduler struct {
	cron       *cron.Cron
	crawlers   []crawler.Crawler
	store      *store.Store
	grpcClient *grpcclient.Client
}

func NewScheduler(store *store.Store, crawlers []crawler.Crawler, grpcClient *grpcclient.Client) *Scheduler {
	return &Scheduler{
		cron:       cron.New(),
		crawlers:   crawlers,
		store:      store,
		grpcClient: grpcClient,
	}
}

func (s *Scheduler) Start() {
	for _, c := range s.crawlers {
		c := c
		_, err := s.cron.AddFunc("@every 30m", func() {
			s.runCrawler(context.Background(), c)
		})
		if err != nil {
			log.Printf("Error adding crawler cron: %v", err)
		}
	}

	_, err := s.cron.AddFunc("@every 6h", func() {
		s.runAnalytics(context.Background())
	})
	if err != nil {
		log.Printf("Error adding analytics cron: %v", err)
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
	rawProblems, err := c.Crawl(ctx, since)
	if err != nil {
		s.store.FailCrawlJob(ctx, crawlJob.ID, err.Error())
		return
	}

	num, err := s.store.UpsertProblems(ctx, rawProblems)
	if err != nil {
		log.Printf("Error upserting problems: %v", err)
		return
	}

	err = s.store.CompleteCrawlJob(ctx, crawlJob.ID, num)
	if err != nil {
		log.Printf("Error completing crawl job: %v", err)
	}

	s.runPostCrawlPipeline(ctx)
}

func (s *Scheduler) runPostCrawlPipeline(ctx context.Context) {
	if s.grpcClient == nil {
		log.Println("gRPC client not available, skipping AI pipeline")
		return
	}

	s.classifyNewProblems(ctx)

	s.embedNewProblems(ctx)
}

func (s *Scheduler) classifyNewProblems(ctx context.Context) {
	problems, err := s.store.GetUnclassifiedProblems(ctx, 100)
	if err != nil {
		log.Printf("Error getting unclassified problems: %v", err)
		return
	}
	if len(problems) == 0 {
		return
	}

	inputs := make([]*pb.ProblemInput, len(problems))
	for i, p := range problems {
		inputs[i] = &pb.ProblemInput{
			Id:    p.ID.String(),
			Title: p.Title,
			Body:  p.Body,
			Tags:  p.Tags,
		}
	}

	resp, err := s.grpcClient.Analysis.ClassifyProblems(ctx, &pb.ClassifyProblemsRequest{
		Problems: inputs,
	})
	if err != nil {
		log.Printf("Error calling ClassifyProblems: %v", err)
		return
	}

	classified := make([]models.ClassifiedProblem, 0, len(resp.Classifications))
	for _, c := range resp.Classifications {
		pid, err := parseUUID(c.ProblemId)
		if err != nil {
			continue
		}
		classified = append(classified, models.ClassifiedProblem{
			RawProblemID:  pid,
			Category:      c.Category,
			Subcategories: c.Subcategories,
			Confidence:    c.Confidence,
			ClassifiedAt:  time.Now(),
		})
	}

	if err := s.store.SaveClassifications(ctx, classified); err != nil {
		log.Printf("Error saving classifications: %v", err)
		return
	}

	log.Printf("Classified %d problems", len(classified))
}

func (s *Scheduler) embedNewProblems(ctx context.Context) {
	problems, err := s.store.GetUnembeddedProblems(ctx, 100)
	if err != nil {
		log.Printf("Error getting unembedded problems: %v", err)
		return
	}
	if len(problems) == 0 {
		return
	}

	inputs := make([]*pb.ProblemInput, len(problems))
	for i, p := range problems {
		inputs[i] = &pb.ProblemInput{
			Id:    p.ID.String(),
			Title: p.Title,
			Body:  p.Body,
			Tags:  p.Tags,
		}
	}

	_, err = s.grpcClient.Analysis.EmbedProblems(ctx, &pb.EmbedProblemsRequest{
		Problems: inputs,
	})
	if err != nil {
		log.Printf("Error calling EmbedProblems: %v", err)
		return
	}

	log.Printf("Embedded %d problems", len(problems))
}

func (s *Scheduler) runAnalytics(ctx context.Context) {
	if s.grpcClient == nil {
		return
	}

	clusterResp, err := s.grpcClient.Analysis.ClusterProblems(ctx, &pb.ClusterProblemsRequest{
		MinClusterSize: 5,
	})
	if err != nil {
		log.Printf("Error calling ClusterProblems: %v", err)
	} else {
		log.Printf("Created %d clusters", len(clusterResp.Clusters))
	}

	trendResp, err := s.grpcClient.Analysis.DetectTrends(ctx, &pb.DetectTrendsRequest{
		WindowDays: 7,
	})
	if err != nil {
		log.Printf("Error calling DetectTrends: %v", err)
	} else {
		log.Printf("Detected %d trends", len(trendResp.Trends))
	}
}

func (s *Scheduler) Stop() {
	s.cron.Stop()
}

func parseUUID(s string) (uuid.UUID, error) {
	return uuid.Parse(s)
}
