package main

import (
	"log"
	"os"

	"github.com/carbonable/leaderboard/internal/api"
	appdb "github.com/carbonable/leaderboard/internal/db"
	"github.com/carbonable/leaderboard/internal/indexer"
	"github.com/carbonable/leaderboard/internal/starknet"
)

func main() {
	network := os.Getenv("NETWORK")
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

	storage := indexer.NewPgStorage(db)
	api.Run(storage, db, rpc)
}
