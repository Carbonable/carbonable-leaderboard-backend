package leaderboard_test

import (
	"fmt"

	"github.com/carbonable/leaderboard/internal/leaderboard"
	"github.com/holiman/uint256"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ScoreCalculator", func() {
	Describe("ScoreCalculatorManager", func() {
		Context("empty manager", func() {
			It("should do nothing with no args", func() {
				scm := leaderboard.NewScoreCalculatorManager()

				score := scm.ComputeScore(leaderboard.DomainEvent{}, []leaderboard.Score{})

				Expect(len(score)).To(BeZero())
			})
		})

		Context("manager without correct event handler", func() {
			It("should process without any errors but score should be null", func() {
				scm := leaderboard.NewScoreCalculatorManager(leaderboard.WithBuilders(&leaderboard.AmountFundedScoreCalculator{}))

				score := scm.ComputeScore(leaderboard.DomainEvent{}, []leaderboard.Score{})

				Expect(len(score)).To(Equal(0))
			})
		})
	})

	Describe("FundedAmountScoreCalculator", func() {
		It("supports only minter:buy event", func() {
			c := &leaderboard.AmountFundedScoreCalculator{}
			Expect(c.Supports(leaderboard.DomainEvent{EventName: "minter:buy"}, []leaderboard.Score{})).To(BeTrue())
			Expect(c.Supports(leaderboard.DomainEvent{EventName: "minter:airdrop"}, []leaderboard.Score{})).To(BeTrue())

			Expect(c.Supports(leaderboard.DomainEvent{EventName: "project:transfer"}, []leaderboard.Score{})).To(BeFalse())
			Expect(c.Supports(leaderboard.DomainEvent{EventName: "project:slot-changed"}, []leaderboard.Score{})).To(BeFalse())
		})

		Context("when event is computed", func() {
			It("should return a score with points", func() {
				c := &leaderboard.AmountFundedScoreCalculator{}
				evt := newMinterBuyEvt("0x4aa5ea227fb0457e4cbe20be80a1896796c2d07c9032835dbbd395629c8f42f_0", map[string]string{
					"address": "0x01e2f67d8132831f210e19c5ee0197aa134308e16f7f284bba2c72e28fc464d2",
					// means 100*1000000$
					"value":     "0x5F5E100",
					"timestamp": "1703845777",
				},
					map[string]string{
						"slot": "0x1", "project_name": "Banegas Farm",
					},
					1703845777,
				)

				Expect(c.Supports(evt, []leaderboard.Score{})).To(BeTrue())
				score := c.Compute(evt, []leaderboard.Score{})
				Expect(score.Points.String()).To(Equal("100000000"))
			})
		})
	})

	Describe("NumberOfProjectsScoreCalculator", func() {
		It("supports all events with project_name", func() {
			c := &leaderboard.NumberOfProjectsScoreCalculator{}
			Expect(c.Supports(leaderboard.DomainEvent{EventName: "minter:buy"}, []leaderboard.Score{})).To(BeFalse())

			Expect(c.Supports(leaderboard.DomainEvent{EventName: "project:transfer"}, []leaderboard.Score{})).To(BeFalse())
			Expect(c.Supports(leaderboard.DomainEvent{EventName: "project:slot-changed"}, []leaderboard.Score{})).To(BeFalse())
			Expect(c.Supports(leaderboard.DomainEvent{EventName: "not-a-real-event-wedontcare"}, []leaderboard.Score{})).To(BeFalse())

			Expect(c.Supports(buyProjectEvt("Banegas Farm", 100), []leaderboard.Score{})).To(BeTrue())
		})

		Context("when event is computed", func() {
			It("should return nil if scores are empty and event is not minter:buy", func() {
				c := &leaderboard.NumberOfProjectsScoreCalculator{}
				score := c.Compute(leaderboard.DomainEvent{EventName: "not-minter:buy"}, []leaderboard.Score{})
				Expect(score).To(BeNil())
			})

			It("should compute score of minter:buy", func() {
				c := &leaderboard.NumberOfProjectsScoreCalculator{}
				evt := buyHundredEvt()
				score := c.Compute(evt, []leaderboard.Score{})
				Expect(score.Points.String()).To(Equal("100"))
			})

			It("should return not duplicate points for same project", func() {
				var totalScore []leaderboard.Score
				c := &leaderboard.NumberOfProjectsScoreCalculator{}
				evt := buyHundredEvt()
				score := c.Compute(evt, []leaderboard.Score{})
				if score != nil {
					totalScore = append(totalScore, *score)
				}
				evt2 := buyHundredEvt()
				score2 := c.Compute(evt2, totalScore)
				if score2 != nil {
					totalScore = append(totalScore, *score2)
				}

				Expect(len(totalScore)).To(Equal(1))
			})
		})
	})

	Describe("ResalerScoreCalculator", func() {
		It("supports only yielder:claim event", func() {
			c := &leaderboard.ResalerScoreCalculator{}

			Expect(c.Supports(leaderboard.DomainEvent{EventName: "yielder:claim"}, []leaderboard.Score{})).To(BeTrue())
			Expect(c.Supports(leaderboard.DomainEvent{EventName: "offseter:claim"}, []leaderboard.Score{})).To(BeFalse())
			Expect(c.Supports(leaderboard.DomainEvent{EventName: "project:transfer"}, []leaderboard.Score{})).To(BeFalse())
			Expect(c.Supports(leaderboard.DomainEvent{EventName: "minter:buy"}, []leaderboard.Score{})).To(BeFalse())
		})

		It("should get points with a factor of one", func() {
			c := &leaderboard.ResalerScoreCalculator{}
			score := c.Compute(resaleHundredEvt(), []leaderboard.Score{})

			Expect(score.Points.String()).To(Equal("100"))
		})
	})

	Describe("OffseterScoreCalculator", func() {
		It("supports only offseter:claim event", func() {
			c := &leaderboard.OffseterScoreCalculator{}

			Expect(c.Supports(leaderboard.DomainEvent{EventName: "offseter:claim"}, []leaderboard.Score{})).To(BeTrue())
			Expect(c.Supports(leaderboard.DomainEvent{EventName: "yielder:claim"}, []leaderboard.Score{})).To(BeFalse())
			Expect(c.Supports(leaderboard.DomainEvent{EventName: "project:transfer"}, []leaderboard.Score{})).To(BeFalse())
			Expect(c.Supports(leaderboard.DomainEvent{EventName: "minter:buy"}, []leaderboard.Score{})).To(BeFalse())
		})

		It("should get points with a factor of a hundred", func() {
			c := &leaderboard.OffseterScoreCalculator{}
			score := c.Compute(offsetHundredEvt(), []leaderboard.Score{})

			Expect(score.Points.String()).To(Equal("10000"))
		})
	})

	Describe("EarlyAdopterScoreCalculator", func() {
		It("supports all events", func() {
			c := &leaderboard.EarlyAdopterScoreCalculator{}

			Expect(c.Supports(leaderboard.DomainEvent{EventName: "offseter:claim"}, []leaderboard.Score{})).To(BeFalse())
			Expect(c.Supports(leaderboard.DomainEvent{EventName: "yielder:claim"}, []leaderboard.Score{})).To(BeFalse())
			Expect(c.Supports(leaderboard.DomainEvent{EventName: "project:transfer"}, []leaderboard.Score{})).To(BeFalse())
			Expect(c.Supports(leaderboard.DomainEvent{EventName: "minter:buy"}, []leaderboard.Score{})).To(BeFalse())

			Expect(c.Supports(buyProjectEvt("Banegas Farm", 100), []leaderboard.Score{})).To(BeTrue())
		})

		It("should get points for different projects, but not duplicate points", func() {
			c := &leaderboard.EarlyAdopterScoreCalculator{}
			score := c.Compute(offsetHundredEvt(), []leaderboard.Score{})

			Expect(score.Points.String()).To(Equal("200000000"))
			score2 := c.Compute(resaleHundredEvt(), []leaderboard.Score{*score})
			Expect(score2).To(BeNil())
		})
		It("should add points for different projects", func() {
			c := &leaderboard.EarlyAdopterScoreCalculator{}
			score := c.Compute(buyHundredEvt(), []leaderboard.Score{})

			Expect(score.Points.String()).To(Equal("200000000"))
			score2 := c.Compute(buyProjectEvt("Las Delicias", 100), []leaderboard.Score{*score})
			Expect(score2.Points.String()).To(Equal("150000000"))
			score3 := c.Compute(buyProjectEvt("Las Delicias", 100), []leaderboard.Score{*score, *score2})
			Expect(score3).To(BeNil())
			score4 := c.Compute(buyProjectEvt("Manjarisoa", 100), []leaderboard.Score{*score, *score2})
			Expect(score4.Points.String()).To(Equal("100000000"))
		})
	})

	Describe("CompositeScoreCalculator", func() {
		It("should add points", func() {
			cc := &leaderboard.CompositeScoreCalculator{}
			cc.Add(leaderboard.NewAdditionScore(uint256.NewInt(10), nil))
			cc.Add(leaderboard.NewAdditionScore(uint256.NewInt(10), nil))

			Expect(cc.GetPoints().String()).To(Equal("20"))
		})
		It("should aggregate poitns", func() {
			cc := &leaderboard.CompositeScoreCalculator{}
			cc.Add(leaderboard.NewAdditionScore(uint256.NewInt(10), nil))
			cc.AddLeaf(0, leaderboard.NewBoostScore(uint256.NewInt(200), nil))
			cc.Add(leaderboard.NewAdditionScore(uint256.NewInt(10), nil))

			Expect(cc.GetPoints().String()).To(Equal("30"))
		})
	})
})

func buyHundredEvt() leaderboard.DomainEvent {
	return buyProjectEvt("Banegas Farm", 100)
}

func buyValueEvt(value uint64) leaderboard.DomainEvent {
	return buyProjectEvt("Banegas Farm", value)
}

func buyProjectEvt(project string, value uint64) leaderboard.DomainEvent {
	return newMinterBuyEvt("0x4aa5ea227fb0457e4cbe20be80a1896796c2d07c9032835dbbd395629c8f42f_0", map[string]string{
		"address": "0x01e2f67d8132831f210e19c5ee0197aa134308e16f7f284bba2c72e28fc464d2",
		// means 100$
		"value": fmt.Sprintf("0x%x", value),
		"time":  "1703845777",
	},
		map[string]string{
			"slot": "0x1", "project_name": project,
		},
		1703845777,
	)
}

func resaleHundredEvt() leaderboard.DomainEvent {
	return newYielderClaimEvt("0x4aa5ea227fb0457e4cbe20be80a1896796c2d07c9032835dbbd395629c8f42f_0", map[string]string{
		"address": "0x01e2f67d8132831f210e19c5ee0197aa134308e16f7f284bba2c72e28fc464d2",
		// amount is dollars * 10^6
		"amount": "0x64",
		"time":   "1703845777",
	}, map[string]string{
		"slot": "0x1", "project_name": "Banegas Farm",
	}, 1703845777)
}

func offsetHundredEvt() leaderboard.DomainEvent {
	return newOffseterClaimEvt("0x4aa5ea227fb0457e4cbe20be80a1896796c2d07c9032835dbbd395629c8f42f_0", map[string]string{
		"address": "0x01e2f67d8132831f210e19c5ee0197aa134308e16f7f284bba2c72e28fc464d2",
		// amount is quantity of co2 in grams
		"amount": "0x64",
		"time":   "1703845777",
	}, map[string]string{
		"slot": "0x1", "project_name": "Banegas Farm",
	}, 1703845777)
}
