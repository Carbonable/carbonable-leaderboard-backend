package leaderboard_test

import (
	"github.com/carbonable/leaderboard/internal/leaderboard"
	"github.com/holiman/uint256"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type givenValueMinterValueAggregator struct {
	value *uint256.Int
}

func (a *givenValueMinterValueAggregator) GetMinterCurrentValue(identifier string) (uint256.Int, error) {
	return *a.value, nil
}

func newGivenValueMinterValueAggregator(value uint64) *givenValueMinterValueAggregator {
	return &givenValueMinterValueAggregator{
		value: uint256.NewInt(value),
	}
}

var milestoneData = []struct {
	expected string
	value    uint64
}{
	{value: 50000, expected: "300"},
	{value: 150000, expected: "200"},
	{value: 300000, expected: "150"},
	{value: 500000, expected: "120"},
	{value: 700000, expected: "110"},
	{value: 1000000, expected: "100"},
	{value: 1200000, expected: "100"},
}

var projectBoostMultiplyData = []struct {
	expected string
	value    uint64
}{
	{value: 500, expected: "750"},
	{value: 1000, expected: "2000"},
	{value: 5000, expected: "15000"},
}

var projectBoostInterval = []struct {
	value     uint64
	boost     uint64
	next      uint64
	nextBoost uint64
}{
	{value: 450, boost: 0, next: 500, nextBoost: 150},
	{value: 500, boost: 150, next: 1000, nextBoost: 200},
	{value: 900, boost: 150, next: 1000, nextBoost: 200},
	{value: 1000, boost: 200, next: 5000, nextBoost: 300},
	{value: 5000, boost: 300, next: 0, nextBoost: 0},
	{value: 5500, boost: 300, next: 0, nextBoost: 0},
}

var _ = Describe("Boost", func() {
	Context("KarathuruFundingMilestoneBoostCalculator", func() {
		When("I check if a boost exists", func() {
			It("boost only Karaturu project", func() {
				boost := leaderboard.NewKaratFundingMilestoneBoostCalculator(newGivenValueMinterValueAggregator(1000000))
				buy := buyProjectEvt("Banegas Farm", 100)
				Expect(boost.Check(buy)).To(BeNil())

				buy = buyProjectEvt("Karathuru", 100)
				Expect(boost.Check(buy)).NotTo(BeNil())

				buy = buyProjectEvt("Las Delicias", 100)
				Expect(boost.Check(buy)).To(BeNil())
			})

			// Test milestone data
			for _, v := range milestoneData {
				It("boost based on minter value milestone", testKarathuruMilestone(v.value, v.expected))
			}
		})

		When("I apply a boost", func() {
			It("apply directly", func() {
				aggregator := newGivenValueMinterValueAggregator(50000)
				boost := leaderboard.NewKaratFundingMilestoneBoostCalculator(aggregator)

				buy := buyProjectEvt("Karathuru", 100*1000000)
				Expect(boost.Check(buy)).NotTo(BeNil())

				scm := leaderboard.FullScoreCalculatorManager(aggregator)
				pr := leaderboard.NewPersonnalRanking("aBeautifulWallet", []leaderboard.DomainEvent{buy})
				leaderboardLine := pr.ComputeScore(scm)

				Expect(leaderboardLine.TotalScore).To(Equal("500"))
			})

			It("should apply only to the previous event", func() {
				// First buy of 100 = 300 points with the * 3 multiplier
				buy := buyProjectEvt("Karathuru", 100*1000000)
				// 100 = 300 points with the * 2 multiplier
				buy2 := buyProjectEvt("Karathuru", 100*1000000)

				scm := leaderboard.FullScoreCalculatorManager(newGivenValueMinterValueAggregator(50000))
				pr := leaderboard.NewPersonnalRanking("aBeautifulWallet", []leaderboard.DomainEvent{buy, buy2})
				leaderboardLine := pr.ComputeScore(scm)

				// 2 boost - 2 amount funded - 1 number of project
				Expect(len(leaderboardLine.Points)).To(Equal(3))
				Expect(leaderboardLine.TotalScore).To(Equal("800"))
			})
		})
	})

	Context("ProjectValueBoostCalculator", func() {
		It("should apply to minter events", func() {
			boost := &leaderboard.ProjectValueBoostCalculator{}
			buy := buyProjectEvt("Banegas Farm", 100)
			airdrop := newMinterAirdropEvt("event_id_1", map[string]string{}, map[string]string{}, 1709030340)
			resale := resaleHundredEvt()
			offset := offsetHundredEvt()

			Expect(boost.Check(buy)).NotTo(BeNil())
			Expect(boost.Check(airdrop)).NotTo(BeNil())
			Expect(boost.Check(resale)).To(BeNil())
			Expect(boost.Check(offset)).To(BeNil())
		})

		It("should apply proper score", func() {
			buy := buyProjectEvt("Banegas Farm", 100*1000000)
			scm := leaderboard.MintPageCalculatorManager()
			pr := leaderboard.NewPersonnalRanking("aBeautifulWallet", []leaderboard.DomainEvent{buy})
			leaderboardLine := pr.ComputeScore(scm)

			Expect(leaderboardLine).NotTo(BeNil())
			Expect(leaderboardLine.TotalScore).To(Equal("100"))
		})

		for _, v := range projectBoostMultiplyData {
			It("should multiply proper score", testProperMultiply(v.value, v.expected))
		}

		It("should multiply proper score", func() {
		})

		for _, v := range projectBoostInterval {
			It("should get proper interval", testProperInterval(v.value, v.boost, v.next, v.nextBoost))
		}
	})

	Context("Boost Aggregation", func() {
		It("should apply both boosts", func() {
			buy := buyProjectEvt("Karathuru", 11000*1000000)

			scm := leaderboard.FullScoreCalculatorManager(newGivenValueMinterValueAggregator(74109))
			pr := leaderboard.NewPersonnalRanking("aBeautifulWallet", []leaderboard.DomainEvent{buy})
			ll := pr.ComputeScore(scm)

			Expect(len(ll.Points)).To(Equal(2))
			for _, p := range ll.Points {
				if p.Rule == string(leaderboard.AmountFundRuleName) {
					Expect(p.Metadata["boosts"]).To(Equal("x2.0 - Funding Karathuru // x3.0 - Funding Value"))
				} else {
					Expect(p.Metadata["boosts"]).To(Equal(""))
				}
			}
		})
	})
})

func testKarathuruMilestone(value uint64, expected string) func() {
	return func() {
		boost := leaderboard.NewKaratFundingMilestoneBoostCalculator(newGivenValueMinterValueAggregator(value))

		buy := buyProjectEvt("Karathuru", 100)
		b := boost.Check(buy)
		Expect(b).NotTo(BeNil())
		s := boost.Apply(buy, b, &leaderboard.Score{Rule: leaderboard.AmountFundRuleName, Points: uint256.NewInt(100)})
		Expect(s.Points.String()).To(Equal(expected))
	}
}

func testProperInterval(value uint64, boost uint64, next uint64, nextBoost uint64) func() {
	return func() {
		bc := leaderboard.DefaultProjectValueBoostCalculator()

		b, n, nb := bc.GetInterval(value)
		Expect(b).To(Equal(uint256.NewInt(boost)))
		Expect(n).To(Equal(uint256.NewInt(next)))
		Expect(nb).To(Equal(uint256.NewInt(nextBoost)))
	}
}

func testProperMultiply(value uint64, expected string) func() {
	return func() {
		buy := buyProjectEvt("Banegas Farm", value*1000000)
		scm := leaderboard.MintPageCalculatorManager()
		pr := leaderboard.NewPersonnalRanking("aBeautifulWallet", []leaderboard.DomainEvent{buy})
		leaderboardLine := pr.ComputeScore(scm)

		Expect(leaderboardLine.TotalScore).To(Equal(expected))
	}
}
