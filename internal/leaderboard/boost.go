package leaderboard

import (
	"slices"

	"github.com/charmbracelet/log"
	"github.com/holiman/uint256"
)

type (
	BuyValueAggregator interface {
		GetMinterCurrentValue(identifier string) (uint256.Int, error)
	}
	PersistBuyValue interface {
		SaveMinterCurrentValue(identifier string, value uint256.Int) error
	}
)

type BoostCalculator interface {
	Check(e DomainEvent) *Boost
	Apply(e DomainEvent, b *Boost, s *Score) *Score
	GetInterval(value uint64) (boost *uint256.Int, next *uint256.Int, nextBoost *uint256.Int)
}

type Boost struct {
	Name string
	// Fiels used to append to metadata for
	// UI display
	DisplayName string
	Value       int
}

type KarathuruFundingMilestoneBoostCalculator struct {
	Aggregator BuyValueAggregator
	Steps      []boostValueItem
}

func NewKaratFundingMilestoneBoostCalculator(a BuyValueAggregator) KarathuruFundingMilestoneBoostCalculator {
	return KarathuruFundingMilestoneBoostCalculator{
		Aggregator: a,
		Steps: []boostValueItem{
			{50000, 300},
			{150000, 200},
			{300000, 150},
			{500000, 120},
			{700000, 110},
		},
	}
}

func (bc KarathuruFundingMilestoneBoostCalculator) Check(e DomainEvent) *Boost {
	projectName := e.Metadata["project_name"]
	if projectName != "Karathuru" || !slices.ContainsFunc([]string{"minter:buy", "minter:airdrop"}, func(s string) bool { return s == e.EventName }) {
		return nil
	}

	return &Boost{Name: "KarathuruFundingMilestone"}
}

func mulCoeficient(value *uint256.Int, coef *uint256.Int) *uint256.Int {
	var res uint256.Int
	res.Mul(value, coef)
	var total uint256.Int
	total.Div(&res, uint256.NewInt(100))
	return &total
}

func (bc KarathuruFundingMilestoneBoostCalculator) Apply(e DomainEvent, b *Boost, s *Score) *Score {
	mv, err := bc.Aggregator.GetMinterCurrentValue("karathuru")
	if err != nil {
		log.Error("error getting minter value", "error", err)
		return nil
	}
	if s.Rule != AmountFundRuleName {
		return nil
	}

	for _, v := range bc.Steps {
		if mv.Cmp(uint256.NewInt(uint64(v.step))) <= 0 {
			s.Points = mulCoeficient(s.Points, uint256.NewInt(v.coef))
			b.DisplayName = "Funding Karathuru"
			b.Value = int(v.coef)
			s.Boosts = append(s.Boosts, *b)
			return s
		}
	}

	return s
}

func (bc KarathuruFundingMilestoneBoostCalculator) GetInterval(value uint64) (boost *uint256.Int, next *uint256.Int, nextBoost *uint256.Int) {
	return uint256.NewInt(0), uint256.NewInt(0), uint256.NewInt(0)
}

type boostValueItem struct {
	step uint64
	coef uint64
}
type ProjectValueBoostCalculator struct {
	Steps []boostValueItem
}

func DefaultProjectValueBoostCalculator() *ProjectValueBoostCalculator {
	return &ProjectValueBoostCalculator{
		Steps: []boostValueItem{
			{5000, 300},
			{1000, 200},
			{500, 150},
		},
	}
}

func (bc *ProjectValueBoostCalculator) Check(e DomainEvent) *Boost {
	if e.EventName != "minter:buy" && e.EventName != "minter:airdrop" {
		return nil
	}
	return &Boost{Name: "ProjectValue"}
}

// FIX: v.Cmp seems to cause consistence issues...
func (bc ProjectValueBoostCalculator) Apply(e DomainEvent, b *Boost, s *Score) *Score {
	if s.Rule != AmountFundRuleName {
		return nil
	}

	for _, v := range bc.Steps {
		if s.Points.Cmp(uint256.NewInt(uint64(v.step*1000000))) >= 0 {
			s.Points = mulCoeficient(s.Points, uint256.NewInt(v.coef))
			b.DisplayName = "Funding Value"
			b.Value = int(v.coef)
			s.Boosts = append(s.Boosts, *b)
			return s
		}
	}

	return s
}

// FIX: v.Cmp seems to cause consistence issues...
func (bc *ProjectValueBoostCalculator) GetInterval(value uint64) (boost *uint256.Int, next *uint256.Int, nextBoost *uint256.Int) {
	boost = uint256.NewInt(0)
	next = uint256.NewInt(0)
	nextBoost = uint256.NewInt(0)
	v := uint256.NewInt(value)

	for i, value := range bc.Steps {

		if v.Cmp(uint256.NewInt(value.step)) >= 0 {
			boost = uint256.NewInt(value.coef)
			prevIdx := i - 1
			if prevIdx < 0 {
				return boost, uint256.NewInt(0), uint256.NewInt(0)
			}
			next = uint256.NewInt(bc.Steps[i-1].step)
			nextBoost = uint256.NewInt(bc.Steps[i-1].coef)
			break
		}
		next = uint256.NewInt(bc.Steps[i].step)
		nextBoost = uint256.NewInt(bc.Steps[i].coef)
	}
	return boost, next, nextBoost
}
