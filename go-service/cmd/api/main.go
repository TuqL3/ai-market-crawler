package main

import (
	"context"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	pb "github.com/lukas/ai-aggregator/go-service/gen/aggregator/v1"
	"github.com/lukas/ai-aggregator/go-service/internal/config"
	"github.com/lukas/ai-aggregator/go-service/internal/grpcclient"
	"github.com/lukas/ai-aggregator/go-service/internal/models"
	"github.com/lukas/ai-aggregator/go-service/internal/store"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	db, err := store.New(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()
	log.Println("Go → Postgres ✓")

	err = db.DB.AutoMigrate(
		&models.RawProblem{},
		&models.ClassifiedProblem{},
		&models.ProblemCluster{},
		&models.TrendSnapshot{},
		&models.ChatSession{},
		&models.ChatMessage{},
		&models.CrawlJob{},
	)
	if err != nil {
		log.Fatalf("Failed to auto migrate: %v", err)
	}
	log.Println("Database migrated ✓")

	ctx := context.Background()
	grpcClient, err := grpcclient.New(ctx, cfg.PythonGRPCAddr)
	if err != nil {
		log.Fatalf("Failed to connect to Python service: %v", err)
	}
	defer grpcClient.Close()

	resp, err := grpcClient.Analysis.ClassifyProblems(ctx, &pb.ClassifyProblemsRequest{
		Problems: []*pb.ProblemInput{
			{Id: "test-1", Title: "Test problem", Body: "This is a test"},
		},
	})
	if err != nil {
		log.Printf("Warning: gRPC test call failed: %v", err)
	} else {
		log.Printf("Go → Python gRPC ✓ (got %d classifications)", len(resp.Classifications))
	}

	r := gin.Default()

	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	addr := ":" + cfg.APIPort
	log.Printf("API server starting on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
