package subscriber

import (
	"github.com/carbonable/leaderboard/internal/indexer"
	"github.com/charmbracelet/log"
	"github.com/nats-io/nats.go"
)

// Handling yielder `Withdraw` event
func YielderWithdrawSubscriber(storage indexer.Storage) nats.MsgHandler {
	return func(m *nats.Msg) {
		event, err := decodeEvent("yielder:withdraw", storage.Get([]byte("EVENT#"+string(m.Data))))
		if err != nil {
			return
		}

		log.Info("yielder:withdraw", "event", event)
	}
}

// Handling yielder `Deposit` event
func YielderDepositSubscriber(storage indexer.Storage) nats.MsgHandler {
	return func(m *nats.Msg) {
		event, err := decodeEvent("yielder:deposit", storage.Get([]byte("EVENT#"+string(m.Data))))
		if err != nil {
			return
		}
		log.Info("yielder:deposit", "event", event)
	}
}

// Handling yiedler `Claim` event
func YielderClaimSubscriber(storage indexer.Storage) nats.MsgHandler {
	return func(m *nats.Msg) {
		event, err := decodeEvent("yielder:claim", storage.Get([]byte("EVENT#"+string(m.Data))))
		if err != nil {
			return
		}
		log.Info("yielder:claim", "event", event)
	}
}

func RegisterYielderSubscribers(args *SubscriberArgs) error {
	if _, err := args.nc.Subscribe("yielder:withdraw", YielderWithdrawSubscriber(args.storage)); err != nil {
		return err
	}
	if _, err := args.nc.Subscribe("yielder:deposit", YielderDepositSubscriber(args.storage)); err != nil {
		return err
	}
	if _, err := args.nc.Subscribe("yielder:claim", YielderClaimSubscriber(args.storage)); err != nil {
		return err
	}

	return nil
}
