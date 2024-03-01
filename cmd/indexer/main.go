package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/carbonable/leaderboard/internal/config"
	appdb "github.com/carbonable/leaderboard/internal/db"
	"github.com/carbonable/leaderboard/internal/indexer"
	"github.com/carbonable/leaderboard/internal/starknet"
	"github.com/carbonable/leaderboard/internal/subscriber"
	_ "github.com/lib/pq"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
)

func main() {
	fmt.Println("Carbonable leaderboard indexer")
	network := os.Getenv("NETWORK")
	cfg, err := config.FromYamlFile(fmt.Sprintf("contracts.%s.yaml", network))
	if err != nil {
		log.Fatalf("failed to get config from file: %v", err)
	}

	var rpc starknet.StarknetRpcClient
	switch network {
	case "goerli":
		rpc = starknet.GoerliJsonRpcStarknetClient()
	case "sepolia":
		rpc = starknet.SepoliaJsonRpcStarknetClient()
	case "mainnet":
		rpc = starknet.MainnetJsonRpcStarknetClient()
	default:
		rpc = starknet.SepoliaJsonRpcStarknetClient()
	}

	db, err := appdb.GetDbConnection()
	if err != nil {
		log.Fatalf("failed to get db connection: %v", err)
		return
	}

	indexerErr := make(chan error)

	opts := &server.Options{}

	// Initialize new server with options
	ns, err := server.NewServer(opts)
	if err != nil {
		panic(err)
	}

	// Start the server via goroutine
	go ns.Start()

	// Wait for server to be ready for connections
	if !ns.ReadyForConnections(4 * time.Second) {
		panic("not ready for connection")
	}

	// Connect to server
	nc, err := nats.Connect(ns.ClientURL())
	if err != nil {
		panic(err)
	}

	storage := indexer.NewPgStorage(db)

	if err = subscriber.RegisterSubscribers(subscriber.NewSubscriberArgs(nc, db, storage, cfg, rpc)); err != nil {
		panic(err)
	}

	client := starknet.NewFeederGatewayClient(os.Getenv("FEEDER_GATEWAY"))
	go indexer.Run(cfg, storage, nc, client, indexerErr)

	select {
	case err := <-indexerErr:
		log.Fatalf("indexer failed: %v", err)
	}
	ns.WaitForShutdown()
}
