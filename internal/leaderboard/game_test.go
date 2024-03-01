package leaderboard_test

import (
	"testing"

	"github.com/carbonable/leaderboard/internal/leaderboard"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestGame(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Game Suite")
}

var _ = Describe("Leaderboard", func() {
	var events []leaderboard.DomainEvent
	BeforeEach(func() {
		events = []leaderboard.DomainEvent{}
		events = getTestEvents(events)
	})

	Describe("PersonnalRanking", func() {
		Context("when personnal ranking is created", func() {
			// events may arrive in an unordered way
			// check they are correcly ordered on creation
			It("should order events by RecordedAt", func() {
				pr := leaderboard.NewPersonnalRanking("0x1e2f67d8132831f210e19c5ee0197aa134308e16f7f284bba2c72e28fc464d2", events)
				Expect(len(pr.Events)).To(Equal(4))

				// Check events are ordered properly
				Expect(pr.Events[0].RecordedAt.Unix()).Should(BeNumerically("<=", pr.Events[1].RecordedAt.Unix()))
				Expect(pr.Events[1].RecordedAt.Unix()).Should(BeNumerically("<=", pr.Events[2].RecordedAt.Unix()))
				Expect(pr.Events[2].RecordedAt.Unix()).Should(BeNumerically("<=", pr.Events[3].RecordedAt.Unix()))
			})
			It("should order events by their EventName", func() {
				pr := leaderboard.NewPersonnalRanking("0x1e2f67d8132831f210e19c5ee0197aa134308e16f7f284bba2c72e28fc464d2", events)
				Expect(len(pr.Events)).To(Equal(4))

				// Check events are ordered properly
				Expect(pr.Events[0].EventName).To(Equal("project:transfer"))
				Expect(pr.Events[1].EventName).To(Equal("project:transfer-value"))
				Expect(pr.Events[2].EventName).To(Equal("project:slot-changed"))
				Expect(pr.Events[3].EventName).To(Equal("project:transfer"))
			})
		})

		Context("when personnal ranking is computed", func() {
			// when events are handled they must get out of the queue
			It("should add events to handled events", func() {
				pr := leaderboard.NewPersonnalRanking("0x1e2f67d8132831f210e19c5ee0197aa134308e16f7f284bba2c72e28fc464d2", events)
				startingLen := len(pr.Events)
				scm := leaderboard.NewScoreCalculatorManager()
				pr.ComputeScore(scm)
				Expect(len(pr.HandledEvents)).To(Equal(startingLen))
			})

			// we should keep track of points in the returned struct
			It("should keep track of points in the returned struct", func() {
				pr := leaderboard.NewPersonnalRanking("0x1e2f67d8132831f210e19c5ee0197aa134308e16f7f284bba2c72e28fc464d2", events)
				scm := leaderboard.FullScoreCalculatorManager(newGivenValueMinterValueAggregator(50000))
				pr.ComputeScore(scm)

				Expect(pr.HandledEvents).ShouldNot(BeEmpty())
			})

			It("should cumulate points", func() {
				events = append(events, newMinterBuyEvt("minter:buy_1", map[string]string{"address": "0x1e2f67d8132831f210e19c5ee0197aa134308e16f7f284bba2c72e28fc464d2", "value": "0x1", "time": "1703845777"}, map[string]string{
					"slot": "0x1", "project_name": "Banegas Farm",
				}, 1703845777))
				events = append(events, newMinterAirdropEvt("minter:airdrop_2", map[string]string{"to": "0x1e2f67d8132831f210e19c5ee0197aa134308e16f7f284bba2c72e28fc464d2", "value": "0x1", "time": "1703845777"}, map[string]string{
					"slot": "0x2", "project_name": "Las Delicias",
				}, 1703845777))
				pr := leaderboard.NewPersonnalRanking("0x1e2f67d8132831f210e19c5ee0197aa134308e16f7f284bba2c72e28fc464d2", events)
				scm := leaderboard.FullScoreCalculatorManager(newGivenValueMinterValueAggregator(50000))
				leaderboardLine := pr.ComputeScore(scm)

				Expect(len(leaderboardLine.Points)).To(Equal(6))
			})
		})
	})
})
