package subscriber

import (
	"github.com/carbonable/leaderboard/internal/indexer"
	"github.com/charmbracelet/log"
	"github.com/nats-io/nats.go"
)

// Handling offseter `Withdraw` event
func OffseterWithdrawSubscriber(storage indexer.Storage) nats.MsgHandler {
	return func(m *nats.Msg) {
		event, err := decodeEvent("offseter:withdraw", storage.Get([]byte("EVENT#"+string(m.Data))))
		if err != nil {
			return
		}

		log.Info("offseter:withdraw", "event", event)
	}
}

// Handling offseter `Deposit` event
func OffseterDepositSubscriber(storage indexer.Storage) nats.MsgHandler {
	return func(m *nats.Msg) {
		event, err := decodeEvent("offseter:deposit", storage.Get([]byte("EVENT#"+string(m.Data))))
		if err != nil {
			return
		}
		log.Info("offseter:deposit", "event", event)
	}
}

// Handling offseter `Claim` event
func OffseterClaimSubscriber(storage indexer.Storage) nats.MsgHandler {
	return func(m *nats.Msg) {
		event, err := decodeEvent("offseter:claim", storage.Get([]byte("EVENT#"+string(m.Data))))
		if err != nil {
			return
		}
		log.Info("offseter:claim", "event", event)
	}
}

func RegisterOffseterSubscribers(args *SubscriberArgs) error {
	if _, err := args.nc.Subscribe("offseter:withdraw", OffseterWithdrawSubscriber(args.storage)); err != nil {
		return err
	}
	if _, err := args.nc.Subscribe("offseter:deposit", OffseterDepositSubscriber(args.storage)); err != nil {
		return err
	}
	if _, err := args.nc.Subscribe("offseter:claim", OffseterClaimSubscriber(args.storage)); err != nil {
		return err
	}

	return nil
}
