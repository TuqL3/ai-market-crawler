package store

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lukas/ai-aggregator/go-service/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

type Store struct {
	DB *gorm.DB
}

func New(databaseURL string) (*Store, error) {
	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying db: %w", err)
	}
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &Store{DB: db}, nil
}

func (s *Store) Close() {
	sqlDB, err := s.DB.DB()
	if err == nil {
		sqlDB.Close()
	}
}

func (s *Store) UpsertProblems(ctx context.Context, problems []models.RawProblem) (int, error) {
	result := s.DB.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "source"},
			{Name: "source_id"},
		},
		DoUpdates: clause.AssignmentColumns([]string{
			"title", "body", "tags", "score",
			"answer_count", "comment_count",
			"source_created", "crawled_at",
		}),
	}).Create(&problems)
	return int(result.RowsAffected), result.Error
}

func (s *Store) CreateCrawlJob(ctx context.Context, source string) (*models.CrawlJob, error) {
	now := time.Now()
	crawlJob := models.CrawlJob{
		Source:    source,
		Status:    "running",
		StartedAt: &now,
	}

	result := s.DB.WithContext(ctx).Create(&crawlJob)
	if result.Error != nil {
		return nil, result.Error
	}
	return &crawlJob, nil
}

func (s *Store) CompleteCrawlJob(ctx context.Context, jobID uuid.UUID, itemsCrawled int) error {
	now := time.Now()
	data := models.CrawlJob{
		Status:       "completed",
		ItemsCrawled: itemsCrawled,
		CompletedAt:  &now,
	}
	err := s.DB.WithContext(ctx).Where("id = ?", jobID).Updates(&data).Error
	if err != nil {
		return err
	}
	return nil
}

func (s *Store) FailCrawlJob(ctx context.Context, jobID uuid.UUID, errMsg string) error {
	now := time.Now()
	data := models.CrawlJob{
		Status:       "failed",
		ErrorMessage: errMsg,
		CompletedAt:  &now,
	}
	err := s.DB.WithContext(ctx).Where("id = ?", jobID).Updates(&data).Error
	if err != nil {
		return err
	}
	return nil
}
