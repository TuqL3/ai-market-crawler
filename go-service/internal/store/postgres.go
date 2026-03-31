package store

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lukas/ai-aggregator/go-service/internal/helpers"
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

func (s *Store) GetUnclassifiedProblems(ctx context.Context, limit int) ([]models.RawProblem, error) {
	var problems []models.RawProblem
	err := s.DB.WithContext(ctx).
		Where("id NOT IN (SELECT raw_problem_id FROM classified_problems)").
		Order("crawled_at DESC").
		Limit(limit).
		Find(&problems).Error
	return problems, err
}

func (s *Store) SaveClassifications(ctx context.Context, classifications []models.ClassifiedProblem) error {
	if len(classifications) == 0 {
		return nil
	}
	result := s.DB.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "raw_problem_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"category", "subcategories", "confidence", "classified_at"}),
	}).Create(&classifications)
	return result.Error
}

func (s *Store) GetUnembeddedProblems(ctx context.Context, limit int) ([]models.RawProblem, error) {
	var problems []models.RawProblem
	err := s.DB.WithContext(ctx).
		Where("id NOT IN (SELECT raw_problem_id FROM problem_embeddings)").
		Order("crawled_at DESC").
		Limit(limit).
		Find(&problems).Error
	return problems, err
}

func (s *Store) GetProblems(ctx context.Context, filter map[string]interface{}, page, pageSize int) ([]models.RawProblem, int64, error) {
	var problems []models.RawProblem
	var total int64

	query := s.DB.WithContext(ctx).Model(&models.RawProblem{})

	if source, ok := filter["source"].(string); ok && source != "" {
		query = query.Where("source = ?", source)
	}
	if category, ok := filter["category"].(string); ok && category != "" {
		query = query.Where("id IN (SELECT raw_problem_id FROM classified_problem WHERE category = ?)", category)
	}
	if tags, ok := filter["tags"].([]interface{}); ok && len(tags) > 0 {
		tagStrings := make([]string, len(tags))
		for i, t := range tags {
			tagStrings[i] = t.(string)
		}
		query = query.Where("tags && ?", fmt.Sprintf("{%s}", helpers.JoinStrings(tagStrings)))
	}
	if dateForm, ok := filter["dateFrom"].(string); ok && dateForm != "" {
		query = query.Where("source_created >= ?", dateForm)
	}
	if dateTo, ok := filter["dateTo"].(string); ok && dateTo != "" {
		query = query.Where("source_created <= ?", dateTo)
	}
	if minScore, ok := filter["minScore"].(int); ok && minScore > 0 {
		query = query.Where("score >= ?", minScore)
	}

	query.Count(&total)
	offset := (page - 1) * pageSize
	err := query.Order("crawled_at DESC").Offset(offset).Limit(pageSize).Find(&problems).Error
	return problems, total, err
}

func (s *Store) GetProblemByID(ctx context.Context, id string) (*models.RawProblem, error) {
	var problem models.RawProblem
	err := s.DB.WithContext(ctx).Where("id = ?", id).First(&problem).Error
	if err != nil {
		return nil, err
	}
	return &problem, nil
}

func (s *Store) GetClusters(ctx context.Context, page, pageSize int) ([]models.ProblemCluster, int64, error) {
	var clusters []models.ProblemCluster
	var total int64
	s.DB.WithContext(ctx).Model(&models.ProblemCluster{}).Count(&total)

	offset := (page - 1) * pageSize
	err := s.DB.WithContext(ctx).Order("updated_at DESC").Offset(offset).Limit(pageSize).Find(&clusters).Error
	return clusters, total, err
}

func (s *Store) GetClusterByID(ctx context.Context, id string) (*models.ProblemCluster, error) {
	var cluster models.ProblemCluster
	err := s.DB.WithContext(ctx).Preload("Members").Where("id = ?", id).First(&cluster).Error
	if err != nil {
		return nil, err
	}
	return &cluster, nil
}

func (s *Store) GetTrends(ctx context.Context, windowDays int) ([]models.TrendSnapshot, error) {
	var trends []models.TrendSnapshot
	since := time.Now().AddDate(0, 0, -windowDays)
	err := s.DB.WithContext(ctx).
		Where("snapshot_at >= ?", since).
		Order("growth_rate DESC").
		Find(&trends).Error
	return trends, err
}

func (s *Store) GetCategories(ctx context.Context) ([]map[string]interface{}, error) {
	var results []map[string]interface{}
	err := s.DB.WithContext(ctx).
		Model(&models.ClassifiedProblem{}).
		Select("category, COUNT(*) as count").
		Group("category").
		Order("count DESC").
		Find(&results).Error
	return results, err
}

func (s *Store) CreateChatSession(ctx context.Context) (*models.ChatSession, error) {
	session := models.ChatSession{}
	err := s.DB.WithContext(ctx).Create(&session).Error
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func (s *Store) CreateChatMessage(ctx context.Context, sessionID string, role, content string) (*models.ChatMessage, error) {
	uid, err := uuid.Parse(sessionID)
	if err != nil {
		return nil, fmt.Errorf("invalid session id: %w", err)
	}
	msg := models.ChatMessage{
		SessionID: uid,
		Role:      role,
		Content:   content,
	}
	if err := s.DB.WithContext(ctx).Create(&msg).Error; err != nil {
		return nil, err
	}
	return &msg, nil
}

func (s *Store) GetChatHistory(ctx context.Context, sessionID string) ([]models.ChatMessage, error) {
	var messages []models.ChatMessage
	err := s.DB.WithContext(ctx).
		Where("session_id = ?", sessionID).
		Order("created_at ASC").
		Find(&messages).Error
	return messages, err
}
