package graph

import (
	"github.com/carbonable/leaderboard/internal/indexer"
	"github.com/carbonable/leaderboard/internal/starknet"
	"gorm.io/gorm"
)

//go:generate go run github.com/99designs/gqlgen generate

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

type Resolver struct {
	storage indexer.Storage
	db      *gorm.DB
	rpc     starknet.StarknetRpcClient
}

func NewGraphResolver(storage indexer.Storage, db *gorm.DB, rpc starknet.StarknetRpcClient) *Resolver {
	return &Resolver{
		storage: storage,
		db:      db,
		rpc:     rpc,
	}
}
