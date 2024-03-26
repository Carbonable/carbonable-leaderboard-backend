package leaderboard

import (
	"github.com/charmbracelet/log"
	u256 "github.com/holiman/uint256"
)

type ScoreBuilder interface {
	Supports(e DomainEvent, score []Score) bool
	Compute(e DomainEvent, score []Score) *Score
}

type baseScoreCalculator struct{}

func (s *baseScoreCalculator) GetPoints() *u256.Int {
	return u256.NewInt(0)
}

func (s *baseScoreCalculator) Aggregate(n ScoreCalculator) ScoreCalculator {
	return s
}

type ScoreCalculator interface {
	GetPoints() *u256.Int
	Aggregate(n ScoreCalculator) ScoreCalculator
}

type CompositeScoreCalculator struct {
	nodes []ScoreCalculator
}

func (s *CompositeScoreCalculator) Add(node ScoreCalculator) {
	s.nodes = append(s.nodes, node)
}

func (s *CompositeScoreCalculator) Get(i int) ScoreCalculator {
	return s.nodes[i]
}

func (s *CompositeScoreCalculator) AddLeaf(i int, node ScoreCalculator) {
	s.nodes[i] = node.Aggregate(s.nodes[i])
}

func (s *CompositeScoreCalculator) GetPoints() *u256.Int {
	var c ScoreCalculator
	c = &baseScoreCalculator{}
	for _, n := range s.nodes {
		c = NewAdditionScore(n.GetPoints(), c)
	}
	return c.GetPoints()
}

func (s *CompositeScoreCalculator) Aggregate(n ScoreCalculator) ScoreCalculator {
	return s
}

type AdditionScore struct {
	inner ScoreCalculator
	value *u256.Int
}

func NewAdditionScore(value *u256.Int, inner ScoreCalculator) ScoreCalculator {
	return &AdditionScore{
		inner,
		value,
	}
}

func (s *AdditionScore) GetPoints() *u256.Int {
	if s.inner == nil {
		return s.value
	}

	var res u256.Int
	return res.Add(s.value, s.inner.GetPoints())
}

func (s *AdditionScore) Aggregate(n ScoreCalculator) ScoreCalculator {
	s.inner = n
	return s
}

type BoostScore struct {
	inner  ScoreCalculator
	factor *u256.Int
}

func NewBoostScore(factor *u256.Int, inner ScoreCalculator) *BoostScore {
	return &BoostScore{
		inner,
		factor,
	}
}

func (s *BoostScore) GetPoints() *u256.Int {
	if s.inner == nil {
		return u256.NewInt(0)
	}

	var res u256.Int
	res.Mul(s.factor, s.inner.GetPoints())
	// NOTE: this may cause issues with precision
	var final u256.Int
	final.Div(&res, u256.NewInt(100))
	return &final
}

func (s *BoostScore) Aggregate(n ScoreCalculator) ScoreCalculator {
	s.inner = n
	return s
}

type (
	ScoreCalculatorManagerFunc func(*ScoreCalculatorManagerOpts)
	ScoreCalculatorManagerOpts struct {
		builder    []ScoreBuilder
		calculator []ScoreCalculator
		booster    []BoostCalculator
	}

	ScoreCalculatorManager struct {
		builder    []ScoreBuilder
		calculator []ScoreCalculator
		booster    []BoostCalculator
	}
)

func defaultScoreCalculatorManagerOpts() *ScoreCalculatorManagerOpts {
	return &ScoreCalculatorManagerOpts{
		builder:    []ScoreBuilder{},
		calculator: []ScoreCalculator{},
		booster:    []BoostCalculator{},
	}
}

func NewScoreCalculatorManager(opts ...ScoreCalculatorManagerFunc) *ScoreCalculatorManager {
	opt := defaultScoreCalculatorManagerOpts()

	for _, optFn := range opts {
		optFn(opt)
	}

	return &ScoreCalculatorManager{
		builder:    opt.builder,
		calculator: opt.calculator,
		booster:    opt.booster,
	}
}

func WithBuilders(builders ...ScoreBuilder) ScoreCalculatorManagerFunc {
	return func(opt *ScoreCalculatorManagerOpts) {
		opt.builder = append(opt.builder, builders...)
	}
}

func WithBoosters(boosters ...BoostCalculator) ScoreCalculatorManagerFunc {
	return func(opt *ScoreCalculatorManagerOpts) {
		opt.booster = append(opt.booster, boosters...)
	}
}

func FullScoreCalculatorManager(aggregator BuyValueAggregator) *ScoreCalculatorManager {
	return NewScoreCalculatorManager(
		WithBuilders(
			&AmountFundedScoreCalculator{},
			&NumberOfProjectsScoreCalculator{},
			&ResalerScoreCalculator{},
			&OffseterScoreCalculator{},
			&EarlyAdopterScoreCalculator{},
		),
		WithBoosters(
			NewKaratFundingMilestoneBoostCalculator(aggregator),
			DefaultProjectValueBoostCalculator(),
		),
	)
}

func MintPageCalculatorManager() *ScoreCalculatorManager {
	return NewScoreCalculatorManager(WithBuilders(&AmountFundedScoreCalculator{}), WithBoosters(DefaultProjectValueBoostCalculator()))
}

func (scm ScoreCalculatorManager) ComputeScore(evt DomainEvent, score []Score) []Score {
	for _, c := range scm.builder {
		if c.Supports(evt, score) {
			s := c.Compute(evt, score)
			if nil != s {
				for _, c := range scm.booster {
					b := c.Check(evt)
					if nil != b {
						c.Apply(evt, b, s)
					}
				}
				score = append(score, *s)
			}
		}
	}
	return score
}

// Compute total score based on all scores item
// Divides by 10^6 to avoid loosing precision
func TotalScore(score []Score) *u256.Int {
	var sc ScoreCalculator
	sc = &baseScoreCalculator{}
	for _, s := range score {
		sc = NewAdditionScore(s.Points, sc)
	}

	var total u256.Int
	total.Div(sc.GetPoints(), u256.NewInt(1000000))
	return &total
}

func AggregateCategories(scores []Score) *CategorisedScore {
	eventToCategoriesMapping := map[RuleName]string{
		AmountFundRuleName:       FundCategory,
		NumberOfProjectsRuleName: FundCategory,
		EarlyAdopterRuleName:     FundCategory,
		OffseterRuleName:         FarmingCategory,
		ResalerRuleName:          FarmingCategory,
	}
	categorisedEvents := map[string][]Score{
		FundCategory:    {},
		FarmingCategory: {},
		OtherCategory:   {},
	}

	for _, s := range scores {
		c := eventToCategoriesMapping[s.Rule]
		categorisedEvents[c] = append(categorisedEvents[c], s)
	}

	return &CategorisedScore{
		Fund:    TotalScore(categorisedEvents[FundCategory]).String(),
		Farming: TotalScore(categorisedEvents[FarmingCategory]).String(),
		Other:   TotalScore(categorisedEvents[OtherCategory]).String(),
	}
}

// Amount Funded - Calculate points based on funded amount
type AmountFundedScoreCalculator struct{}

func (sc *AmountFundedScoreCalculator) Supports(e DomainEvent, score []Score) bool {
	return e.EventName == "minter:buy" || e.EventName == "minter:airdrop" || e.EventName == "migrator:migration"
}

func (sc *AmountFundedScoreCalculator) Compute(e DomainEvent, score []Score) *Score {
	// Data has 3 keys address(felt), value(u256), time(u64)
	// 1 value = 1 $
	// 1 $ = 1 point
	value, err := u256.FromHex(e.Data["value"])
	if err != nil {
		log.Fatal("failed to parse value to u256", "error", err)
	}

	// NOTE: value is coming from blockchain multiplied by 10^6
	return &Score{Points: value, Event: e, Rule: AmountFundRuleName}
}

// Number of projects - Calculate points based on number of projects
type NumberOfProjectsScoreCalculator struct{}

func (sc *NumberOfProjectsScoreCalculator) Supports(e DomainEvent, score []Score) bool {
	pName, exists := e.Metadata["project_name"]
	if !exists {
		return false
	}
	return !ruleWasApplied(NumberOfProjectsRuleName, pName, score)
}

func ruleWasApplied(rule RuleName, projectName string, score []Score) bool {
	for _, s := range score {
		pName, exists := s.Event.Metadata["project_name"]
		if exists && s.Rule == rule && pName == projectName {
			return true
		}
	}

	return false
}

func getScoreFromEventValue(e DomainEvent) *Score {
	p, err := u256.FromHex(e.Data["value"])
	if err != nil {
		// NOTE: if event doesnt have value, just return nil as it will be ignored
		return nil
	}

	var points u256.Int
	// NOTE: Multiply by 10^6 to avoid loosing precision
	points.Mul(p, u256.NewInt(1000000))

	return &Score{Points: p, Event: e, Rule: NumberOfProjectsRuleName}
}

func (sc *NumberOfProjectsScoreCalculator) Compute(e DomainEvent, score []Score) *Score {
	found := false
	projectName, exists := e.Metadata["project_name"]
	if exists {
		for _, s := range score {
			pName, ex := s.Event.Metadata["project_name"]
			if !ex || pName == projectName && s.Rule == NumberOfProjectsRuleName {
				found = true
				continue
			}
			// NOTE: Multiply by 10^6 to avoid loosing precision
			// Each project = 200 points * 10^6
			return &Score{Points: u256.NewInt(200000000), Event: e, Rule: NumberOfProjectsRuleName}
		}
	}

	if !found && e.EventName == "minter:buy" {
		return getScoreFromEventValue(e)
	}

	return nil
}

type ResalerScoreCalculator struct{}

func (sc *ResalerScoreCalculator) Supports(e DomainEvent, score []Score) bool {
	return e.EventName == "yielder:claim"
}

func (sc *ResalerScoreCalculator) Compute(e DomainEvent, score []Score) *Score {
	// computes points based on 1$ = 1 point
	// e.Data["amount"] equals dollars * 10^6 (usdc payment token decimals)
	// points = dollars * 10^-6
	p, err := u256.FromHex(e.Data["amount"])
	if err != nil {
		log.Error("yielder:claim - failed to parse points from hex", "error", err)
		return nil
	}

	// NOTE: We divide by 10^6 at the total end of computation
	return &Score{Points: p, Event: e, Rule: ResalerRuleName}
}

type OffseterScoreCalculator struct{}

func (sc *OffseterScoreCalculator) Supports(e DomainEvent, score []Score) bool {
	return e.EventName == "offseter:claim"
}

func (sc *OffseterScoreCalculator) Compute(e DomainEvent, score []Score) *Score {
	// computes points based on 1tCO2 = 100 point
	// e.Data["amount"] equals grams
	// points = grams * 100

	p, err := u256.FromHex(e.Data["amount"])
	if err != nil {
		log.Error("offseter:claim - failed to parse points from hex", "error", err)
		return nil
	}

	var points u256.Int
	// NOTE: 100 for base multiplier 1000000 to avoid loosing precision
	points.Mul(p, u256.NewInt(100))

	return &Score{Points: &points, Event: e, Rule: OffseterRuleName}
}

var pointsPerProject = map[string]uint64{
	"Banegas Farm": 200,
	"Las Delicias": 150,
	"Manjarisoa":   100,
}

type EarlyAdopterScoreCalculator struct{}

func (sc *EarlyAdopterScoreCalculator) Supports(e DomainEvent, score []Score) bool {
	projectName, exists := e.Metadata["project_name"]
	if !exists {
		return false
	}
	_, exists = pointsPerProject[projectName]
	return exists
}

func (sc *EarlyAdopterScoreCalculator) Compute(e DomainEvent, score []Score) *Score {
	projectName, exists := e.Metadata["project_name"]
	if exists {
		for _, s := range score {
			pName, ex := s.Event.Metadata["project_name"]
			if !ex || pName == projectName && s.Rule == EarlyAdopterRuleName {
				return nil
			}
		}
	}

	defaultScore := uint64(50)
	p, exists := pointsPerProject[projectName]
	if exists {
		defaultScore = p
	}

	var points u256.Int
	// NOTE: mul by 1000000 to avoid loosing precision
	points.Mul(u256.NewInt(defaultScore), u256.NewInt(1000000))

	return &Score{Points: &points, Event: e, Rule: EarlyAdopterRuleName}
}
