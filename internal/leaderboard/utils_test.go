package leaderboard_test

import (
	"time"

	"github.com/carbonable/leaderboard/internal/leaderboard"
	"github.com/oklog/ulid/v2"
)

var evtFeltName = map[string]string{
	"project:transfer": "0x37c14f554a4f46f90d0fff8e69cfd60c04b99b80368f58061f186bce4215053",
}

func newEvent(evtId string, evtName string, data map[string]string, metadata map[string]string, keys []string, ts int64) leaderboard.DomainEvent {
	return leaderboard.DomainEvent{
		RecordedAt:    time.Unix(ts, 0),
		Data:          data,
		Metadata:      metadata,
		EventId:       evtId,
		EventNameFelt: evtFeltName[evtName],
		EventName:     evtName,
		FromAddress:   "0x130b5a3035eef0470cff2f9a450a7a6856a3c5a4ea3f5b7886c2d03a50d2bf",
		WalletAddress: "0x1e2f67d8132831f210e19c5ee0197aa134308e16f7f284bba2c72e28fc464d2",
		Keys:          keys,
		ID:            ulid.Make(),
	}
}

func newProjectTransferEvt(evtId string, data map[string]string, metadata map[string]string, keys []string, ts int64) leaderboard.DomainEvent {
	return newEvent(evtId, "project:transfer", data, metadata, keys, ts)
}

func newProjectTransferValueEvt(evtId string, data map[string]string, metadata map[string]string, ts int64) leaderboard.DomainEvent {
	return newEvent(evtId, "project:transfer-value", data, metadata, []string{}, ts)
}

func newProjectSlotChangedEvt(evtId string, data map[string]string, metadata map[string]string, ts int64) leaderboard.DomainEvent {
	return newEvent(evtId, "project:slot-changed", data, metadata, []string{}, ts)
}

func newMinterBuyEvt(evtId string, data map[string]string, metadata map[string]string, ts int64) leaderboard.DomainEvent {
	return newEvent(evtId, "minter:buy", data, metadata, []string{}, ts)
}

func newMinterAirdropEvt(evtId string, data map[string]string, metadata map[string]string, ts int64) leaderboard.DomainEvent {
	return newEvent(evtId, "minter:airdrop", data, metadata, []string{}, ts)
}

func newYielderClaimEvt(evtId string, data map[string]string, metadata map[string]string, ts int64) leaderboard.DomainEvent {
	return newEvent(evtId, "yielder:claim", data, metadata, []string{}, ts)
}

func newOffseterClaimEvt(evtId string, data map[string]string, metadata map[string]string, ts int64) leaderboard.DomainEvent {
	return newEvent(evtId, "offseter:claim", data, metadata, []string{}, ts)
}

func getTestEvents(events []leaderboard.DomainEvent) []leaderboard.DomainEvent {
	events = append(events, newProjectSlotChangedEvt(
		"0x4aa5ea227fb0457e4cbe20be80a1896796c2d07c9032835dbbd395629c8f42f_1",
		map[string]string{
			"new_slot": "0x1",
			"old_slot": "0x0",
			"token_id": "0x1",
		},
		map[string]string{
			"slot": "0x1", "project_name": "Banegas Farm",
		},
		1703845777,
	))
	events = append(events, newProjectTransferEvt(
		"0x4aa5ea227fb0457e4cbe20be80a1896796c2d07c9032835dbbd395629c8f42f_0",
		map[string]string{
			"to": "0x1e2f67d8132831f210e19c5ee0197aa134308e16f7f284bba2c72e28fc464d2", "from": "0x0", "token_id": "0x1",
		},
		map[string]string{
			"slot": "0x1", "project_name": "Banegas Farm",
		},
		[]string{
			"0x0", "0x1e2f67d8132831f210e19c5ee0197aa134308e16f7f284bba2c72e28fc464d2", "0x1", "0x0",
		},
		1703845777,
	))
	events = append(events, newProjectTransferValueEvt(
		"0x4aa5ea227fb0457e4cbe20be80a1896796c2d07c9032835dbbd395629c8f42f_2",
		map[string]string{
			"value": "0xe4e1c0", "to_token_id": "0x1", "from_token_id": "0x0",
		},
		map[string]string{
			"slot": "0x1", "project_name": "Banegas Farm",
		},
		1703845777,
	))
	events = append(events, newProjectTransferEvt(
		"0x4aa5ea227fb0457e4cbe20be80a1896796c2d07c9032835dbbd395629c8f42f_3",
		map[string]string{
			"to": "0x1e2f67d8132831f210e19c5ee0197aa134308e16f7f284bba2c72e28fc464d2", "from": "0x0", "token_id": "0x1",
		},
		map[string]string{
			"slot": "0x1", "project_name": "Banegas Farm",
		},
		[]string{
			"0x0", "0x1e2f67d8132831f210e19c5ee0197aa134308e16f7f284bba2c72e28fc464d2", "0x1", "0x0",
		},
		1703845960,
	))
	return events
}
