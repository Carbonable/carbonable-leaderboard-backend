package leaderboard

import (
	"testing"
	"time"

	"github.com/holiman/uint256"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"
)

func TestBuildPointMetadata(t *testing.T) {
	assert := assert.New(t)
	s := Score{
		Points: uint256.NewInt(100),
		Rule:   "event:name",
		Boosts: []Boost{
			{
				Name:        "Karathuru",
				DisplayName: "Funding Karathuru",
				Value:       200,
			},
			{
				Name:        "Project funding",
				DisplayName: "Funding project",
				Value:       150,
			},
		},
		Event: DomainEvent{
			RecordedAt:    time.Unix(1710068400, 0),
			Data:          map[string]string{"fake": "data"},
			Metadata:      map[string]string{"slot": "1", "project_name": "Karathuru"},
			EventId:       "anEventId",
			EventNameFelt: "anEventFeft",
			EventName:     "event:name",
			FromAddress:   "fromaddress",
			WalletAddress: "walletaddress",
			Keys:          []string{},
			ID:            ulid.Make(),
		},
	}

	metadata := buildPointMetadata(s)
	assert.Equal(metadata["date"], "1710068400000", "date should match")
	assert.Equal(metadata["event"], "event:name", "event name should match")
	assert.Equal(metadata["boosts"], "x2.0 - Funding Karathuru // x1.5 - Funding project", "event name should match")
}
