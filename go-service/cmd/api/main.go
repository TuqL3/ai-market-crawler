package main

import (
	"log"

	"github.com/lukas/ai-aggregator/go-service/internal/api"
	"github.com/lukas/ai-aggregator/go-service/internal/config"
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

	r, err := api.NewRouter(db)
	if err != nil {
		log.Fatalf("Failed to create router: %v", err)
	}

	addr := ":" + cfg.APIPort
	log.Printf("API server starting on %s", addr)
	log.Printf("GraphQL:    http://localhost%s/graphql", addr)
	log.Printf("Playground: http://localhost%s/playground", addr)

	if err := r.Run(addr); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
