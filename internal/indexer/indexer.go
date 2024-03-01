package indexer

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/carbonable/leaderboard/internal/config"
	"github.com/carbonable/leaderboard/internal/starknet"
	"github.com/charmbracelet/log"
	"github.com/nats-io/nats.go"
)

// Base config given to indexer service
type IndexerConfig struct {
	Gateway string
}

// Main run loop function for indexer service
// this is spinned up as a background task
// Sync all blocks from chain and stores them locally
// Additionnaly save contract config to save some time for next run
func Run(cfg *config.Config, storage Storage, nc *nats.Conn, client *starknet.FeederGatewayClient, errCh chan<- error) {
	wg := sync.WaitGroup{}
	for _, c := range cfg.Contracts {
		idx := NewIndexer(c, storage, nc, client)
		go idx.Run(cfg.StartBlock)
		wg.Add(1)
	}
	wg.Wait()
}

type Indexer interface {
	Index(*starknet.GetBlockResponse) error
	Run() error
}

type EventIndexer struct {
	storage  Storage
	nats     *nats.Conn
	msgch    chan *starknet.GetBlockResponse
	client   *starknet.FeederGatewayClient
	contract config.Contract
}

func (i *EventIndexer) Index(block *starknet.GetBlockResponse) error {
	address := i.contract.Address

	go i.indexTransaction(address, block)
	go i.indexEvent(address, block)

	i.saveContractIndexLatestBlock(address, block.BlockNumber)

	return nil
}

func (i *EventIndexer) Run(startBlock uint64) error {
	contractIdx, _ := i.getContractIdx(i.contract.Address, startBlock)

	go i.replayBlocks(contractIdx.Blocks)

	block := contractIdx.LatestBlock
	if startBlock > block {
		block = startBlock
	}
	log.Info("Running indexer for contract", "address", i.contract.Address, "block", block)
	go i.start(block)
	for {
		msg := <-i.msgch
		err := i.Index(msg)
		if err != nil {
			log.Error("failed to index block", "error", err)
		}
	}
}

func (i *EventIndexer) replayBlocks(blocks []uint64) {
	for _, b := range blocks {
		resp, err := i.fetchBlock(b)
		if err != nil {
			// error will often be some timeout
			// majority of time block not found happens we block has not beed indexed yet
			time.Sleep(5 * time.Second)
			continue
		}

		i.msgch <- resp
	}
}

func (i *EventIndexer) start(block uint64) {
	var retries int
	for {
		resp, err := i.fetchBlock(block)
		if err != nil {
			if retries > 5 {
				i.syncBlock(block)
				retries = 0
			}
			// error will often be some timeout
			// majority of time block not found happens we block has not beed indexed yet
			time.Sleep(10 * time.Second)
			retries++
			continue
		}

		i.msgch <- resp
		block++
	}
}

func (i *EventIndexer) fetchBlock(blockNumber uint64) (*starknet.GetBlockResponse, error) {
	key := []byte(fmt.Sprintf("BLOCK#%d", blockNumber))
	if i.storage.Has(key) {
		block := i.storage.Get(key)
		buf := bytes.NewBuffer(block)
		decoder := gob.NewDecoder(buf)
		var resp starknet.GetBlockResponse
		err := decoder.Decode(&resp)
		if err != nil {
			log.Error(fmt.Sprintf("failed to decode block %s", err))
			return &resp, err
		}
		return &resp, nil
	}
	return nil, errors.New("block not found")
}

func (i *EventIndexer) indexTransaction(address string, block *starknet.GetBlockResponse) {
	for _, tx := range block.Transactions {
		if starknet.EnsureStarkFelt(tx.SenderAddress) != address {
			continue
		}
		var buf bytes.Buffer
		encoder := gob.NewEncoder(&buf)
		err := encoder.Encode(tx)
		if err != nil {
			log.Error("failed to encode event", "error", err)
		}

		if err := i.storage.Set([]byte(fmt.Sprintf("%s#TX#%s", address, tx.TransactionHash)), buf.Bytes()); err != nil {
			log.Error("failed to store event", "error", err)
		}
		log.Info("Indexing tx for address", "address", address, "tx", tx.TransactionHash)

		i.saveContractIndexInteresstingBlock(address, block.BlockNumber)
	}
}

func (i *EventIndexer) indexEvent(address string, block *starknet.GetBlockResponse) {
	for _, tx := range block.TransactionReceipts {
		for eventIdx, event := range tx.Events {
			if starknet.EnsureStarkFelt(event.FromAddress) != address {
				continue
			}

			var buf bytes.Buffer
			encoder := gob.NewEncoder(&buf)

			// Aggregating event_id to event
			eventId := fmt.Sprintf("%s_%d", tx.TransactionHash, eventIdx)
			event.EventId = eventId
			event.RecordedAt = time.Unix(int64(block.Timestamp), 0)

			err := encoder.Encode(event)
			if err != nil {
				log.Error("failed to encode event", "error", err)
			}

			if err := i.storage.Set([]byte(fmt.Sprintf("%s#EVENT#%s", address, eventId)), buf.Bytes()); err != nil {
				log.Error("failed to store event", "error", err)
			}
			if err := i.storage.Set([]byte(fmt.Sprintf("EVENT#%s", eventId)), buf.Bytes()); err != nil {
				log.Error("failed to store event", "error", err)
			}
			i.nats.Publish("event:published", []byte(eventId))
			log.Info("Indexing event for address", "address", address, "eventId", eventId)

			i.saveContractIndexInteresstingBlock(address, block.BlockNumber)
		}
	}
}

func (i *EventIndexer) getContractIdx(address string, block uint64) (*ContractIndex, []byte) {
	contractIdxKey := []byte(fmt.Sprintf("IDX#%s", address))

	contractIdx := NewContractIndex(block)
	if i.storage.Has(contractIdxKey) {
		idx := i.storage.Get(contractIdxKey)
		err := contractIdx.Decode(idx)
		if err != nil {
			log.Error("failed to decode contract index", "error", err, "contract", address)
		}
	}
	return contractIdx, contractIdxKey
}

func (i *EventIndexer) saveContractIndexInteresstingBlock(address string, block uint64) {
	idx, key := i.getContractIdx(address, block)

	idx.AddBlock(block)

	buf, err := idx.Encode()
	if err != nil {
		log.Error("failed to encode contract index", "error", err, "contract", address)
	}
	if err := i.storage.Set(key, buf.Bytes()); err != nil {
		log.Error("failed to store contract index", "error", err, "contract", address)
	}
}

func (i *EventIndexer) saveContractIndexLatestBlock(address string, block uint64) {
	idx, key := i.getContractIdx(address, block)

	idx.SetLatestBlock(block)

	buf, err := idx.Encode()
	if err != nil {
		log.Error("failed to encode contract index", "error", err, "contract", address)
	}
	if err := i.storage.Set(key, buf.Bytes()); err != nil {
		log.Error("failed to store contract index", "error", err, "contract", address)
	}
}

func (i *EventIndexer) syncBlock(block uint64) {
	resp, err := i.client.GetBlock(block)
	if err != nil {
		log.Error("failed to re-fetch block")
		return
	}

	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	if err := encoder.Encode(resp); err != nil {
		log.Error(fmt.Sprintf("failed to encode block %s", err))
	}

	if err := i.storage.Set([]byte(fmt.Sprintf("BLOCK#%d", resp.BlockNumber)), buf.Bytes()); err != nil {
		log.Error(err)
	}
}

func NewIndexer(contract config.Contract, storage Storage, nc *nats.Conn, client *starknet.FeederGatewayClient) *EventIndexer {
	return &EventIndexer{
		contract: contract,
		storage:  storage,
		nats:     nc,
		msgch:    make(chan *starknet.GetBlockResponse),
		client:   client,
	}
}

type ContractIndex struct {
	Blocks      []uint64
	LatestBlock uint64
}

func NewContractIndex(startBlock uint64) *ContractIndex {
	return &ContractIndex{
		LatestBlock: startBlock,
		Blocks:      []uint64{},
	}
}

func (c *ContractIndex) SetLatestBlock(block uint64) {
	c.LatestBlock = block
}

func (c *ContractIndex) AddBlock(block uint64) {
	if !slices.Contains(c.Blocks, block) {
		c.Blocks = append(c.Blocks, block)
	}
}

func (c *ContractIndex) Encode() (bytes.Buffer, error) {
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	err := encoder.Encode(c)
	if err != nil {
		return buf, err
	}

	return buf, nil
}

func (c *ContractIndex) Decode(buf []byte) error {
	decoder := gob.NewDecoder(bytes.NewBuffer(buf))
	err := decoder.Decode(c)
	if err != nil {
		return err
	}

	return nil
}
