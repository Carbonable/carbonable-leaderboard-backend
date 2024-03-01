package subscriber

import (
	"errors"
	"time"

	"github.com/carbonable/leaderboard/internal/indexer"
	"github.com/carbonable/leaderboard/internal/leaderboard"
	"github.com/carbonable/leaderboard/internal/starknet"
	"github.com/charmbracelet/log"
	u256 "github.com/holiman/uint256"
	"github.com/nats-io/nats.go"
	"github.com/oklog/ulid/v2"
	"gorm.io/gorm"
)

// Handling project `Transfer` event
func ProjectTransferSubscriber(storage indexer.Storage, db *gorm.DB, rpc starknet.StarknetRpcClient) nats.MsgHandler {
	return func(m *nats.Msg) {
		event, err := decodeEvent("project:transfer", storage.Get([]byte("EVENT#"+string(m.Data))))
		if err != nil {
			return
		}

		log.Info("project:transfer", "event", event)
		data := map[string]string{
			"from":     event.Keys[1],
			"to":       event.Keys[2],
			"token_id": event.Keys[3],
		}
		slot, err := starknet.GetSlotOf(rpc, event.FromAddress, starknet.HexStringToUint64(data["token_id"]))
		if err != nil {
			log.Error("project:transfer -> failed to get slot of token_id", "error", err)
		}
		slotUri, err := starknet.GetSlotUri(rpc, event.FromAddress, slot)
		if err != nil {
			log.Error("project:transfer -> failed to get slot_uri", "error", err)
		}
		metadata := metadataFromSlotUri(slotUri, slot)

		evt := leaderboard.DomainEventFromStarknetEvent(event, "project:transfer", data["to"], data, metadata)
		db.Create(&evt)
	}
}

// Handling project `TransferValue` event
func ProjectTransferValueSubscriber(storage indexer.Storage, db *gorm.DB, rpc starknet.StarknetRpcClient) nats.MsgHandler {
	return func(m *nats.Msg) {
		event, err := decodeEvent("project:transfer-value", storage.Get([]byte("EVENT#"+string(m.Data))))
		if err != nil {
			return
		}

		log.Info("project:transfer-value", "event", event)
		data := map[string]string{
			"from_token_id": event.Data[0],
			"to_token_id":   event.Data[2],
			"value":         event.Data[4],
		}
		slot, err := starknet.GetSlotOf(rpc, event.FromAddress, starknet.HexStringToUint64(data["to_token_id"]))
		if err != nil {
			log.Error("project:transfer -> failed to get slot of token_id", "error", err)
		}
		slotUri, err := starknet.GetSlotUri(rpc, event.FromAddress, slot)
		if err != nil {
			log.Error("project:transfer -> failed to get slot_uri", "error", err)
		}
		metadata := metadataFromSlotUri(slotUri, slot)
		// to get wallet : find project:transfer event associated with event
		// domain_event where event_name = project:transfer and data->token_id = data["to_token_id"] and metadata->slot = slot
		wallet := getWalletFromTransferEvent(db, data["to_token_id"], starknet.FeltFromUint64(slot).String())

		evt := leaderboard.DomainEventFromStarknetEvent(event, "project:transfer-value", wallet, data, metadata)
		db.Create(&evt)

		updateMinterBoughtValue(db, evt)
	}
}

// Handling project `SlotChanged` event
func ProjectSlotChangedSubscriber(storage indexer.Storage, db *gorm.DB, rpc starknet.StarknetRpcClient) nats.MsgHandler {
	return func(m *nats.Msg) {
		event, err := decodeEvent("project:slot-changed", storage.Get([]byte("EVENT#"+string(m.Data))))
		if err != nil {
			return
		}

		log.Info("project:slot-changed", "event", event)
		data := map[string]string{
			"token_id": event.Data[0],
			"old_slot": event.Data[2],
			"new_slot": event.Data[4],
		}

		slot, err := starknet.GetSlotOf(rpc, event.FromAddress, starknet.HexStringToUint64(data["token_id"]))
		if err != nil {
			log.Error("project:transfer -> failed to get slot of token_id", "error", err)
		}
		slotUri, err := starknet.GetSlotUri(rpc, event.FromAddress, slot)
		if err != nil {
			log.Error("project:transfer -> failed to get slot_uri", "error", err)
		}
		metadata := metadataFromSlotUri(slotUri, slot)
		// to get wallet : find project:transfer event associated with event
		// domain_event where event_name = project:transfer and data->token_id = data["token_id"] and metadata->slot = new_slot
		wallet := getWalletFromTransferEvent(db, data["token_id"], starknet.FeltFromUint64(slot).String())

		evt := leaderboard.DomainEventFromStarknetEvent(event, "project:slot-changed", wallet, data, metadata)
		db.Create(&evt)
	}
}

func RegisterProjectSubscribers(args *SubscriberArgs) error {
	if _, err := args.nc.Subscribe("project:transfer", ProjectTransferSubscriber(args.storage, args.db, args.rpc)); err != nil {
		return err
	}
	if _, err := args.nc.Subscribe("project:transfer-value", ProjectTransferValueSubscriber(args.storage, args.db, args.rpc)); err != nil {
		return err
	}
	if _, err := args.nc.Subscribe("project:slot-changed", ProjectSlotChangedSubscriber(args.storage, args.db, args.rpc)); err != nil {
		return err
	}

	return nil
}

func metadataFromSlotUri(slotUri *starknet.SlotUri, slot uint64) map[string]string {
	return map[string]string{
		"slot":         starknet.FeltFromUint64(slot).String(),
		"project_name": slotUri.Name,
	}
}

func getTransferEvent(db *gorm.DB, tokenId string, slot string) leaderboard.DomainEvent {
	var evt leaderboard.DomainEvent
	db.Where("event_name = ? AND data->>'token_id' = ? AND metadata->>'slot' = ?", "project:transfer", tokenId, slot).First(&evt)
	// NOTE: handle failure if event without wallet is saved before the others. We are pretty sure the event will come at a point so we do not add max retry
	if evt.WalletAddress == "" {
		log.Error("failed to get wallet from transfer event", "event", evt)
		time.Sleep(10 * time.Second)
		return getTransferEvent(db, tokenId, slot)
	}
	return evt
}

func getWalletFromTransferEvent(db *gorm.DB, tokenId string, slot string) string {
	evt := getTransferEvent(db, tokenId, slot)
	return evt.WalletAddress
}

func updateMinterBoughtValue(db *gorm.DB, evt *leaderboard.DomainEvent) {
	transferEvent := getTransferEvent(db, evt.Data["to_token_id"], evt.Metadata["slot"])

	var minterBuyValue leaderboard.MinterBuyValue
	err := db.Where("name = ? and slot = ?", transferEvent.Metadata["project_name"], transferEvent.Metadata["slot"]).First(&minterBuyValue).Error

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
