package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RawProblem struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	Source        string    `gorm:"type:varchar(20);not null"`
	SourceID      string    `gorm:"type:varchar(255);not null"`
	URL           string    `gorm:"type:text;not null"`
	Title         string    `gorm:"type:text;not null"`
	Body          string    `gorm:"type:text"`
	Tags          []string  `gorm:"type:text[]"`
	Score         int       `gorm:"default:0"`
	AnswerCount   int       `gorm:"default:0"`
	CommentCount  int       `gorm:"default:0"`
	SourceCreated time.Time `gorm:"not null"`
	CrawledAt     time.Time `gorm:"not null;default:now()"`
}

func (r *RawProblem) BeforeCreate(tx *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}

type ClassifiedProblem struct {
	ID            uuid.UUID  `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	RawProblemID  uuid.UUID  `gorm:"type:uuid;not null;uniqueIndex"`
	RawProblem    RawProblem `gorm:"foreignKey:RawProblemID"`
	Category      string     `gorm:"type:varchar(100);not null"`
	Subcategories []string   `gorm:"type:text[]"`
	Confidence    float32    `gorm:"not null"`
	ClassifiedAt  time.Time  `gorm:"not null;default:now()"`
}

func (c *ClassifiedProblem) BeforeCreate(tx *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return nil
}

type ProblemCluster struct {
	ID              uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	Label           string    `gorm:"type:varchar(255);not null"`
	Summary         string    `gorm:"type:text"`
	KeyThemes       []string  `gorm:"type:text[]"`
	CommonSolutions []string  `gorm:"type:text[]"`
	CohesionScore   float32
	ProblemCount    int          `gorm:"default:0"`
	Members         []RawProblem `gorm:"many2many:cluster_members"`
	CreatedAt       time.Time    `gorm:"not null;default:now()"`
	UpdatedAt       time.Time    `gorm:"not null;default:now()"`
}

func (p *ProblemCluster) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}

type TrendSnapshot struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	ClusterID    uuid.UUID `gorm:"type:uuid"`
	Label        string    `gorm:"type:varchar(255)"`
	ProblemCount int
	GrowthRate   float32
	WindowStart  time.Time `gorm:"not null"`
	WindowEnd    time.Time `gorm:"not null"`
	SnapshotAt   time.Time `gorm:"not null;default:now()"`
}

func (t *TrendSnapshot) BeforeCreate(tx *gorm.DB) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	return nil
}

type ChatSession struct {
	ID        uuid.UUID     `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	Messages  []ChatMessage `gorm:"foreignKey:SessionID"`
	CreatedAt time.Time     `gorm:"not null;default:now()"`
	UpdatedAt time.Time     `gorm:"not null;default:now()"`
}

func (c *ChatSession) BeforeCreate(tx *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return nil
}

type ChatMessage struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	SessionID uuid.UUID `gorm:"type:uuid;not null"`
	Role      string    `gorm:"type:varchar(20);not null"`
	Content   string    `gorm:"type:text;not null"`
	Sources   string    `gorm:"type:jsonb"`
	CreatedAt time.Time `gorm:"not null;default:now()"`
}

func (c *ChatMessage) BeforeCreate(tx *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return nil
}

type CrawlJob struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	Source       string    `gorm:"type:varchar(20);not null"`
	Status       string    `gorm:"type:varchar(20);not null;default:'pending'"`
	ItemsCrawled int       `gorm:"default:0"`
	ErrorMessage string    `gorm:"type:text"`
	StartedAt    *time.Time
	CompletedAt  *time.Time
	CreatedAt    time.Time `gorm:"not null;default:now()"`
}

func (c *CrawlJob) BeforeCreate(tx *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return nil
}
