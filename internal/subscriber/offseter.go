package subscriber

import (
	"github.com/carbonable/leaderboard/internal/indexer"
	"github.com/carbonable/leaderboard/internal/leaderboard"
	"github.com/carbonable/leaderboard/internal/starknet"
	"github.com/charmbracelet/log"
	"github.com/nats-io/nats.go"
	"gorm.io/gorm"
)

// Handling offseter `Withdraw` event
func OffseterWithdrawSubscriber(storage indexer.Storage, db *gorm.DB, rpc starknet.StarknetRpcClient) nats.MsgHandler {
	return func(m *nats.Msg) {
		event, err := decodeEvent("offseter:withdraw", storage.Get([]byte("EVENT#"+string(m.Data))))
		if err != nil {
			return
		}

		log.Info("offseter:withdraw", "event", event)

		metadata := getMetadataFromEvent(rpc, event.FromAddress)
		data := map[string]string{
			"address": event.Data[0],
			"value":   event.Data[1],
			"time":    event.Data[3],
		}

		evt := leaderboard.DomainEventFromStarknetEvent(event, "offseter:withdraw", event.Data[0], data, metadata)
		db.Create(&evt)
	}
}

// Handling offseter `Deposit` event
func OffseterDepositSubscriber(storage indexer.Storage, db *gorm.DB, rpc starknet.StarknetRpcClient) nats.MsgHandler {
	return func(m *nats.Msg) {
		event, err := decodeEvent("offseter:deposit", storage.Get([]byte("EVENT#"+string(m.Data))))
		if err != nil {
			return
		}

		log.Info("offseter:deposit", "event", event)

		metadata := getMetadataFromEvent(rpc, event.FromAddress)
		data := map[string]string{
			"address": event.Data[0],
			"value":   event.Data[1],
			"time":    event.Data[3],
		}

		evt := leaderboard.DomainEventFromStarknetEvent(event, "offseter:deposit", event.Data[0], data, metadata)
		db.Create(&evt)
	}
}

// Handling offseter `Claim` event
func OffseterClaimSubscriber(storage indexer.Storage, db *gorm.DB, rpc starknet.StarknetRpcClient) nats.MsgHandler {
	return func(m *nats.Msg) {
		event, err := decodeEvent("offseter:claim", storage.Get([]byte("EVENT#"+string(m.Data))))
		if err != nil {
			return
		}

		log.Info("offseter:claim", "event", event)

		metadata := getMetadataFromEvent(rpc, event.FromAddress)
		data := map[string]string{
			"address": event.Data[0],
			"amount":  event.Data[1],
			"time":    event.Data[3],
		}

		evt := leaderboard.DomainEventFromStarknetEvent(event, "offseter:claim", event.Data[0], data, metadata)
		db.Create(&evt)
	}
}

func RegisterOffseterSubscribers(args *SubscriberArgs) error {
	if _, err := args.nc.Subscribe("offseter:withdraw", OffseterWithdrawSubscriber(args.storage, args.db, args.rpc)); err != nil {
		return err
	}
	if _, err := args.nc.Subscribe("offseter:deposit", OffseterDepositSubscriber(args.storage, args.db, args.rpc)); err != nil {
		return err
	}
	if _, err := args.nc.Subscribe("offseter:claim", OffseterClaimSubscriber(args.storage, args.db, args.rpc)); err != nil {
		return err
	}

	return nil
}
