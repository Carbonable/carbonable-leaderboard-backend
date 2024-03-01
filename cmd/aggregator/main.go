package main

import (
	"context"
	"time"

	appdb "github.com/carbonable/leaderboard/internal/db"
	"github.com/carbonable/leaderboard/internal/leaderboard"
	"github.com/charmbracelet/log"
)

func main() {
	log.Info("Starting leaderboard aggregator")

	db, err := appdb.GetDbConnection()
	if err != nil {
		log.Fatalf("failed to get db connection: %v", err)
		return
	}

	aggregator := leaderboard.NewPgAggregrator(db)
	for {
		go aggregator.Run(context.Background())
		time.Sleep(1 * time.Minute)
	}
}
