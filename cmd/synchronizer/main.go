package main

import (
	"fmt"
	"log"
	"os"

	appdb "github.com/carbonable/leaderboard/internal/db"
	"github.com/carbonable/leaderboard/internal/starknet"

	"github.com/carbonable/leaderboard/internal/config"
	"github.com/carbonable/leaderboard/internal/indexer"
	"github.com/carbonable/leaderboard/internal/synchronizer"
	_ "github.com/lib/pq"
)

func main() {
	fmt.Println("Carbonable synchronizer")
	network := os.Getenv("NETWORK")
	cfg, err := config.FromYamlFile(fmt.Sprintf("contracts.%s.yaml", network))
	if err != nil {
		log.Fatalf("failed to get config from file: %v", err)
	}

	indexerErr := make(chan error)

	db, err := appdb.GetDbConnection()
	if err != nil {
		log.Fatal("failed to acquire db connection", "err", err)
	}
	client := starknet.NewFeederGatewayClient(os.Getenv("FEEDER_GATEWAY"))
	storage := indexer.NewPgStorage(db)

	go synchronizer.Run(cfg, client, storage, indexerErr)

	select {
	case err := <-indexerErr:
		log.Fatalf("indexer failed: %v", err)
	}
}
