package subscriber

import (
	"errors"

	"github.com/carbonable/leaderboard/internal/indexer"
	"github.com/carbonable/leaderboard/internal/leaderboard"
	"github.com/carbonable/leaderboard/internal/starknet"
	"github.com/charmbracelet/log"
	u256 "github.com/holiman/uint256"
	"github.com/nats-io/nats.go"
	"github.com/oklog/ulid/v2"
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

		// update the minterbuyValue
		updateMinterBoughtValue(db, evt)
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

		// update the minterbuyValue
		updateMinterBoughtValue(db, evt)
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

func updateMinterBoughtValue(db *gorm.DB, evt *leaderboard.DomainEvent) {
	var minterBuyValue leaderboard.MinterBuyValue
	err := db.Where("name = ? and slot = ?", evt.Metadata["project_name"], evt.Metadata["slot"]).First(&minterBuyValue).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		minterBuyValue = leaderboard.MinterBuyValue{
			Name:  evt.Metadata["project_name"],
			Slot:  evt.Metadata["slot"],
			Value: *u256.NewInt(0),
			ID:    ulid.Make(),
		}
	}
	value, conversionErr := u256.FromHex(evt.Data["value"])
	if conversionErr != nil {
		log.Error("failed to convert value to u256", "error", err)
	}
	var newVal u256.Int
	newVal.Add(&minterBuyValue.Value, value)
	minterBuyValue.Value = newVal

	if errors.Is(err, gorm.ErrRecordNotFound) {
		db.Model(&minterBuyValue).Create(map[string]interface{}{
			"ID":    minterBuyValue.ID,
			"Name":  minterBuyValue.Name,
			"Slot":  minterBuyValue.Slot,
			"Value": minterBuyValue.Value.Uint64(),
		})
		return
	}
	db.Model(&minterBuyValue).Where("name = ? and slot = ?", minterBuyValue.Name, minterBuyValue.Slot).Update("value", minterBuyValue.Value.Uint64())
}
