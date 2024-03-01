package subscriber

import (
	"github.com/NethermindEth/juno/core/felt"
	"github.com/carbonable/leaderboard/internal/indexer"
	"github.com/carbonable/leaderboard/internal/leaderboard"
	"github.com/carbonable/leaderboard/internal/starknet"
	"github.com/charmbracelet/log"
	"github.com/nats-io/nats.go"
	"gorm.io/gorm"
)

// Handling migrator `Migration` event
func MigratorMigrationSubscriber(storage indexer.Storage, db *gorm.DB, rpc starknet.StarknetRpcClient) nats.MsgHandler {
	return func(m *nats.Msg) {
		event, err := decodeEvent("migrator:migration", storage.Get([]byte("EVENT#"+string(m.Data))))
		if err != nil {
			return
		}

		log.Info("migrator:migration", "event", event)

		data := map[string]string{
			"address":      event.Data[0],
			"token_id":     event.Data[1],
			"new_token_id": event.Data[3],
			"slot":         event.Data[5],
			"value":        event.Data[7],
		}

		var slot felt.Felt
		err = slot.UnmarshalJSON([]byte(event.Data[5]))
		if err != nil {
			log.Error("failed to unmarshal slot in felt", "error", err)
		}

		metadata := getMetadataFromMigrator(rpc, event.FromAddress, slot.Uint64())

		evt := leaderboard.DomainEventFromStarknetEvent(event, "migrator:migration", event.Data[0], data, metadata)
		db.Create(&evt)
	}
}

func RegisterMigratorSubscribers(args *SubscriberArgs) error {
	if _, err := args.nc.Subscribe("migrator:migration", MigratorMigrationSubscriber(args.storage, args.db, args.rpc)); err != nil {
		return err
	}

	return nil
}

func getMetadataFromMigrator(rpc starknet.StarknetRpcClient, address string, slot uint64) map[string]string {
	projectAddress, err := starknet.MigratorTargetAddress(rpc, address)
	if err != nil {
		log.Error("failed to get project address", "error", err)
		return map[string]string{}
	}

	slotUri, err := starknet.GetSlotUri(rpc, projectAddress, slot)
	if err != nil {
		log.Error("failed to get project slotUri", "error", err)
		return map[string]string{}
	}

	return metadataFromSlotUri(slotUri, slot)
}
