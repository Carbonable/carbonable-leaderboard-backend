package subscriber

import (
	"github.com/carbonable/leaderboard/internal/indexer"
	"github.com/carbonable/leaderboard/internal/leaderboard"
	"github.com/carbonable/leaderboard/internal/starknet"
	"github.com/charmbracelet/log"
	"github.com/nats-io/nats.go"
	"gorm.io/gorm"
)

// Handling yielder `Withdraw` event
func YielderWithdrawSubscriber(storage indexer.Storage, db *gorm.DB, rpc starknet.StarknetRpcClient) nats.MsgHandler {
	return func(m *nats.Msg) {
		event, err := decodeEvent("yielder:withdraw", storage.Get([]byte("EVENT#"+string(m.Data))))
		if err != nil {
			return
		}

		log.Info("yielder:withdraw", "event", event)

		metadata := getMetadataFromEvent(rpc, event.FromAddress)
		data := map[string]string{
			"address": event.Data[0],
			"value":   event.Data[1],
			"time":    event.Data[3],
		}

		evt := leaderboard.DomainEventFromStarknetEvent(event, "yielder:withdraw", event.Data[0], data, metadata)
		db.Create(&evt)
	}
}

// Handling yielder `Deposit` event
func YielderDepositSubscriber(storage indexer.Storage, db *gorm.DB, rpc starknet.StarknetRpcClient) nats.MsgHandler {
	return func(m *nats.Msg) {
		event, err := decodeEvent("yielder:deposit", storage.Get([]byte("EVENT#"+string(m.Data))))
		if err != nil {
			return
		}

		log.Info("yielder:deposit", "event", event)

		metadata := getMetadataFromEvent(rpc, event.FromAddress)
		data := map[string]string{
			"address": event.Data[0],
			"value":   event.Data[1],
			"time":    event.Data[3],
		}

		evt := leaderboard.DomainEventFromStarknetEvent(event, "yielder:deposit", event.Data[0], data, metadata)
		db.Create(&evt)
	}
}

// Handling yiedler `Claim` event
func YielderClaimSubscriber(storage indexer.Storage, db *gorm.DB, rpc starknet.StarknetRpcClient) nats.MsgHandler {
	return func(m *nats.Msg) {
		event, err := decodeEvent("yielder:claim", storage.Get([]byte("EVENT#"+string(m.Data))))
		if err != nil {
			return
		}

		log.Info("yielder:claim", "event", event)

		metadata := getMetadataFromEvent(rpc, event.FromAddress)
		data := map[string]string{
			"address": event.Data[0],
			"amount":  event.Data[1],
			"time":    event.Data[3],
		}

		evt := leaderboard.DomainEventFromStarknetEvent(event, "yielder:claim", event.Data[0], data, metadata)
		db.Create(&evt)
	}
}

func RegisterYielderSubscribers(args *SubscriberArgs) error {
	if _, err := args.nc.Subscribe("yielder:withdraw", YielderWithdrawSubscriber(args.storage, args.db, args.rpc)); err != nil {
		return err
	}
	if _, err := args.nc.Subscribe("yielder:deposit", YielderDepositSubscriber(args.storage, args.db, args.rpc)); err != nil {
		return err
	}
	if _, err := args.nc.Subscribe("yielder:claim", YielderClaimSubscriber(args.storage, args.db, args.rpc)); err != nil {
		return err
	}

	return nil
}
