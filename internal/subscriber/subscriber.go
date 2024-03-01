package subscriber

import (
	"github.com/carbonable/leaderboard/internal/config"
	"github.com/carbonable/leaderboard/internal/indexer"
	"github.com/carbonable/leaderboard/internal/starknet"
	"github.com/charmbracelet/log"
	"github.com/nats-io/nats.go"
	"gorm.io/gorm"
)

type (
	SubscriberCallback func(indexer.Storage) nats.MsgHandler
	SubscriberArgs     struct {
		nc      *nats.Conn
		db      *gorm.DB
		storage indexer.Storage
		cfg     *config.Config
		rpc     starknet.StarknetRpcClient
	}
)

func NewSubscriberArgs(nc *nats.Conn, db *gorm.DB, storage indexer.Storage, cfg *config.Config, rpc starknet.StarknetRpcClient) *SubscriberArgs {
	return &SubscriberArgs{
		nc:      nc,
		db:      db,
		storage: storage,
		cfg:     cfg,
		rpc:     rpc,
	}
}

// Main event publisher :
// every events thats is saved into system get through this subscriber wich dispatch domain specific events
func EventPublishedSubscriber(args *SubscriberArgs) nats.MsgHandler {
	return func(m *nats.Msg) {
		encodedEvent := args.storage.Get([]byte("EVENT#" + string(m.Data)))
		event, err := starknet.DecodeGob[starknet.Event](encodedEvent)
		if err != nil {
			log.Error("event:published", "error", err)
			return
		}
		contract := args.cfg.GetContract(starknet.EnsureStarkFelt(event.FromAddress))
		if nil == contract {
			log.Error("contract not found")
			return
		}

		feltEventName := event.Keys[0]
		for i, e := range contract.Events {
			felt, _ := starknet.StarknetKeccak([]byte(i))
			if felt.String() != feltEventName {
				continue
			}

			_ = args.nc.Publish(e, []byte(event.EventId))
			log.Info("event:published", "eventId", event.EventId, "eventName", i)
		}
	}
}

// Register application specific subscribers
func RegisterSubscribers(args *SubscriberArgs) error {
	if _, err := args.nc.Subscribe("event:published", EventPublishedSubscriber(args)); err != nil {
		log.Error("failed to register event published subscriber", "error", err)
		return err
	}

	if err := RegisterProjectSubscribers(args); err != nil {
		log.Error("failed to register project subscribers", "error", err)
		return err
	}
	if err := RegisterMinterSubscribers(args); err != nil {
		log.Error("failed to register minter subscribers", "error", err)
		return err
	}
	if err := RegisterOffseterSubscribers(args); err != nil {
		log.Error("failed to register offseter subscribers", "error", err)
		return err
	}
	if err := RegisterYielderSubscribers(args); err != nil {
		log.Error("failed to register yielder subscribers", "error", err)
		return err
	}
	if err := RegisterMigratorSubscribers(args); err != nil {
		log.Error("failed to register migrator subscribers", "error", err)
		return err
	}

	return nil
}

func decodeEvent(name string, encodedEvent []byte) (*starknet.Event, error) {
	event, err := starknet.DecodeGob[starknet.Event](encodedEvent)
	if err != nil {
		log.Error(name, "error", err)
		return nil, err
	}
	return event, nil
}
