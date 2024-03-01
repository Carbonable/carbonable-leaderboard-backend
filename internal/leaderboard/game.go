package leaderboard

import (
	"sort"

	u256 "github.com/holiman/uint256"
)

var eventPriority = []string{"project:transfer", "project:transfer-value", "project:slot-changed"}

// Ordering util type
type ByRecordedAtAndEventName []DomainEvent

func (b ByRecordedAtAndEventName) Len() int      { return len(b) }
func (b ByRecordedAtAndEventName) Swap(i, j int) { b[i], b[j] = b[j], b[i] }
func (b ByRecordedAtAndEventName) Less(i, j int) bool {
	e1, e2 := b[i], b[j]
	e1RecAt, e2RecAt, e1EvtIdx, e2EvtIdx := e1.RecordedAt.Unix(), e2.RecordedAt.Unix(), indexOf(eventPriority, e1.EventName), indexOf(eventPriority, e2.EventName)
	if e1RecAt == e2RecAt {
		return e1EvtIdx < e2EvtIdx
	}
	return e1RecAt < e2RecAt
}

type RuleName string

const (
	AmountFundRuleName       RuleName = "amount_funded"
	NumberOfProjectsRuleName RuleName = "number_of_projects"
	EarlyAdopterRuleName     RuleName = "early_adopter"
	OffseterRuleName         RuleName = "offseter"
	ResalerRuleName          RuleName = "resaler"
	BoostRuleName            RuleName = "boost"

	FundCategory    string = "fund"
	FarmingCategory string = "farming"
	OtherCategory   string = "other"
)

type PersonnalRanking struct {
	CustomerWallet string
	Events         []DomainEvent
	HandledEvents  []DomainEvent
}

type ScoreCalculatorBuilderFn func() ScoreCalculator

type Score struct {
	Points *u256.Int
	Rule   RuleName
	Event  DomainEvent
}

func (pr *PersonnalRanking) ComputeScore(scm *ScoreCalculatorManager) *LeaderboardLine {
	var scores []Score
	for _, e := range pr.Events {
		scores = scm.ComputeScore(e, scores)
		pr.HandledEvents = append(pr.HandledEvents, e)
	}

	totalScore := TotalScore(scores)
	categories := AggregateCategories(scores)

	return LeaderboardLineFromScore(pr.CustomerWallet, scores, *totalScore, categories)
}

func NewPersonnalRanking(wallet string, events []DomainEvent) *PersonnalRanking {
	sort.Sort(ByRecordedAtAndEventName(events))

	return &PersonnalRanking{
		CustomerWallet: wallet,
		Events:         events,
		HandledEvents:  []DomainEvent{},
	}
}

func indexOf(s []string, e string) int {
	for i, a := range s {
		if a == e {
			return i
		}
	}
	return 99999
}
