package main

import (
	"flag"

	infradb "github.com/carbonable/leaderboard/internal/db"
	"github.com/carbonable/leaderboard/internal/indexer"
	"github.com/carbonable/leaderboard/internal/leaderboard"
	"github.com/charmbracelet/log"
)

func main() {
	fresh := flag.Bool("fresh", false, "drop all tables before migration")
	flag.Parse()
	log.Info("Starting application migration")

	db, err := infradb.GetDbConnection()
	if err != nil {
		log.Fatalf("failed to get db connection: %v", err)
		return
	}

	if *fresh {
		log.Info("Dropping all tables")
		_ = db.Migrator().DropTable(&leaderboard.DomainEvent{}, &leaderboard.LeaderboardLine{}, &leaderboard.MinterBuyValue{})
	}

	_ = db.AutoMigrate(&leaderboard.DomainEvent{}, &leaderboard.LeaderboardLine{}, &leaderboard.MinterBuyValue{}, &indexer.KVStore{})

	log.Info("Migration done !")
}
