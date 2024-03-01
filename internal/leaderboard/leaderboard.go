package leaderboard

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/carbonable/leaderboard/internal/starknet"
	u256 "github.com/holiman/uint256"
	"github.com/oklog/ulid/v2"
)

// NOTE: Types to ease use of json in gorm and postgres
type (
	LeaderboardCategory int
	EventData           map[string]string
	EventMetadata       map[string]string
	EventKeys           []string
	Points              []Point
)

const (
	Global LeaderboardCategory = iota
	Customer
)

type DomainEvent struct {
	RecordedAt    time.Time
	Data          EventData     `gorm:"serializer:json;type:jsonb"`
	Metadata      EventMetadata `gorm:"serializer:json;type:jsonb"`
	EventId       string        `gorm:"unique"`
	EventNameFelt string
	EventName     string
	FromAddress   string
	WalletAddress string
	Keys          EventKeys `gorm:"serializer:json;type:jsonb"`
	ID            ulid.ULID `gorm:"primaryKey"`
}

func DomainEventFromStarknetEvent(event *starknet.Event, eventName string, wallet string, data map[string]string, metadata map[string]string) *DomainEvent {
	return &DomainEvent{
		RecordedAt:    event.RecordedAt,
		EventId:       event.EventId,
		EventNameFelt: event.Keys[0],
		EventName:     eventName,
		FromAddress:   event.FromAddress,
		WalletAddress: wallet,
		Keys:          event.Keys[1:],
		Data:          data,
		Metadata:      metadata,
		ID:            ulid.Make(),
	}
}

type LeaderboardLine struct {
	Categories    CategorisedScore `gorm:"serializer:json;type:jsonb"`
	WalletAddress string           `gorm:"unique"`
	TotalScore    string
	Points        Points    `gorm:"serializer:json;type:jsonb"`
	ID            ulid.ULID `gorm:"primaryKey"`
}

type Point struct {
	Metadata EventMetadata `json:"metadata" gorm:"serializer:json;type:jsonb"`
	Rule     string        `json:"rule"`
	Value    uint          `json:"value"`
}

type CategorisedScore struct {
	Fund    string `json:"fund" gorm:"serializer:json;type:jsonb"`
	Farming string `json:"farming" gorm:"serializer:json;type:jsonb"`
	Other   string `json:"other" gorm:"serializer:json;type:jsonb"`
}

type MinterBuyValue struct {
	Name  string
	Slot  string
	ID    ulid.ULID `gorm:"primaryKey"`
	Value u256.Int  `gorm:"type:numeric"`
}

func LeaderboardLineFromScore(wallet string, score []Score, totalScore u256.Int, categories *CategorisedScore) *LeaderboardLine {
	var points Points
	for _, s := range score {
		points = append(points, Point{Metadata: s.Event.Metadata, Rule: string(s.Rule), Value: uint(s.Points.Uint64())})
	}
	return &LeaderboardLine{
		WalletAddress: wallet,
		Points:        points,
		ID:            ulid.Make(),
		TotalScore:    totalScore.String(),
		Categories:    *categories,
	}
}

// EventData
func (a EventData) Value() (driver.Value, error) {
	return json.Marshal(a)
}

func (a *EventData) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	return json.Unmarshal(b, &a)
}

// EventMetadata
func (a EventMetadata) Value() (driver.Value, error) {
	return json.Marshal(a)
}

func (a *EventMetadata) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	return json.Unmarshal(b, &a)
}

// EventKeys
func (a EventKeys) Value() (driver.Value, error) {
	return json.Marshal(a)
}

func (a *EventKeys) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	return json.Unmarshal(b, &a)
}

// Points
func (a Points) Value() (driver.Value, error) {
	return json.Marshal(a)
}

func (a *Points) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	return json.Unmarshal(b, &a)
}

// CategorisedScore
func (a CategorisedScore) Value() (driver.Value, error) {
	return json.Marshal(a)
}

func (a *CategorisedScore) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	return json.Unmarshal(b, &a)
}

// LeaderboardCategory
func (lbc LeaderboardCategory) String() string {
	return []string{"Global", "Customer"}[lbc]
}

func (lbc *LeaderboardCategory) Scan(value interface{}) error {
	*lbc = LeaderboardCategory(value.(int64))
	return nil
}

func (lbc LeaderboardCategory) Value() (driver.Value, error) {
	return int(lbc), nil
}
