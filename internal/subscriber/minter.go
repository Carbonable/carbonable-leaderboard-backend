package subscriber

import (
	"github.com/carbonable/leaderboard/internal/indexer"
	"github.com/carbonable/leaderboard/internal/leaderboard"
	"github.com/carbonable/leaderboard/internal/starknet"
	"github.com/charmbracelet/log"
	"github.com/nats-io/nats.go"
	"gorm.io/gorm"
)

// Handling minter `Buy` event
func MinterBuySubscriber(storage indexer.Storage, rpc starknet.StarknetRpcClient, db *gorm.DB) nats.MsgHandler {
	return func(m *nats.Msg) {
		event, err := decodeEvent("minter:buy", storage.Get([]byte("EVENT#"+string(m.Data))))
		if err != nil {
			return
		}

		log.Info("minter:buy", "event", event)

		data := map[string]string{
			"address": event.Data[0],
			"value":   event.Data[1],
			"time":    event.Data[3],
		}

		metadata := getMetadataFromEvent(rpc, event.FromAddress)

		evt := leaderboard.DomainEventFromStarknetEvent(event, "minter:buy", event.Data[0], data, metadata)
		db.Create(&evt)
	}
}

// Handling minter `Airdrop` event
func MinterAirdropSubscriber(storage indexer.Storage, rpc starknet.StarknetRpcClient, db *gorm.DB) nats.MsgHandler {
	return func(m *nats.Msg) {
		event, err := decodeEvent("minter:airdrop", storage.Get([]byte("EVENT#"+string(m.Data))))
		if err != nil {
			return
		}
		log.Info("minter:airdrop", "event", event)

		data := map[string]string{
			"to":    event.Data[0],
			"value": event.Data[1],
			"time":  event.Data[3],
		}
		metadata := getMetadataFromEvent(rpc, event.FromAddress)

		evt := leaderboard.DomainEventFromStarknetEvent(event, "minter:airdrop", event.Data[0], data, metadata)
		db.Create(&evt)
	}
}

func RegisterMinterSubscribers(args *SubscriberArgs) error {
	if _, err := args.nc.Subscribe("minter:buy", MinterBuySubscriber(args.storage, args.rpc, args.db)); err != nil {
		return err
	}
	if _, err := args.nc.Subscribe("minter:airdrop", MinterAirdropSubscriber(args.storage, args.rpc, args.db)); err != nil {
		return err
	}

	return nil
}

func getMetadataFromEvent(rpc starknet.StarknetRpcClient, address string) map[string]string {
	projectAddress, err := starknet.MinterGetProjectAddress(rpc, address)
	if err != nil {
		log.Error("failed to get project address", "error", err)
		return map[string]string{}
	}
	projectSlot, err := starknet.MinterGetProjectSlot(rpc, address)
	if err != nil {
		log.Error("failed to get project slot", "error", err)
		return map[string]string{}
	}

	slotUri, err := starknet.GetSlotUri(rpc, projectAddress, projectSlot)
	if err != nil {
		log.Error("failed to get project slotUri", "error", err)
		return map[string]string{}
	}

	return metadataFromSlotUri(slotUri, projectSlot)
}
