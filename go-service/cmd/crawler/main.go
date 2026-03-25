package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/go-github/v84/github"
	"github.com/lukas/ai-aggregator/go-service/internal/config"
	"github.com/lukas/ai-aggregator/go-service/internal/crawler"
	"github.com/lukas/ai-aggregator/go-service/internal/scheduler"
	"github.com/lukas/ai-aggregator/go-service/internal/store"
	"golang.org/x/oauth2"
	"golang.org/x/time/rate"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	db, err := store.New(cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	ts := oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: cfg.GitHubToken,
	})

	httpClient := oauth2.NewClient(context.Background(), ts)
	ghClient := github.NewClient(httpClient)

	limiter := rate.NewLimiter(rate.Every(time.Second), 5)
	githubCrawler := crawler.NewGithubCrawler(ghClient, limiter)

	crawlers := []crawler.Crawler{githubCrawler}
	s := scheduler.NewScheduler(db, crawlers)
	s.Start()
	log.Println("starting crawlers")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("shutting down gracefully")
	s.Stop()

}
