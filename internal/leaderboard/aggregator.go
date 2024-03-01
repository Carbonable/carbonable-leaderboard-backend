package leaderboard

import (
	"context"
	"fmt"

	"github.com/charmbracelet/log"
	"github.com/holiman/uint256"
	"gorm.io/gorm"
)

type LeaderboardAggregator interface {
	Run()
	GetParticipants() ([]string, error)
	GetParticipantEvents(wallet string) ([]DomainEvent, error)
}

type PgLeaderboardAggregator struct {
	db *gorm.DB
}

type PgMinterBuyValueAggregator struct {
	db *gorm.DB
}

func (a *PgMinterBuyValueAggregator) GetMinterCurrentValue(identifier string) (uint256.Int, error) {
	return *uint256.NewInt(0), nil
}

func NewPgMinterBuyValueAggregator(db *gorm.DB) *PgMinterBuyValueAggregator {
	return &PgMinterBuyValueAggregator{
		db: db,
	}
}

func (a *PgLeaderboardAggregator) Run(ctx context.Context) {
	errch := make(chan error)
	// create tmp table
	createTempTable(a.db)
	scm := FullScoreCalculatorManager(&PgMinterBuyValueAggregator{
		db: a.db,
	})

	p, err := a.GetParticipants()
	if err != nil {
		log.Fatal("failed to get participants", "error", err)
	}
	for _, w := range p {
		// add participant score to tmp table
		go a.computeParticipantEvents(w, scm, errch)
		err := <-errch
		if err != nil {
			log.Error("failed to compute participant events", "error", err)
		}
	}

	backupLeaderboardLines(a.db)
	hotSwapTables(a.db)
	cleanupTmpTables(a.db)
	fmt.Printf("\n")
}

func (a *PgLeaderboardAggregator) computeParticipantEvents(wallet string, scm *ScoreCalculatorManager, errch chan<- error) {
	log.Info("computing participant events", "wallet", wallet)
	events, err := a.GetParticipantEvents(wallet)
	if err != nil {
		log.Fatal("failed to get participant events", "error", err)
	}

	pr := NewPersonnalRanking(wallet, events)
	leaderboardLine := pr.ComputeScore(scm)

	a.db.Exec("INSERT INTO tmp_leaderboard_lines (wallet_address, points, categories, id, total_score) VALUES (?, ?, ?, ?, ?)", leaderboardLine.WalletAddress, leaderboardLine.Points, leaderboardLine.Categories, leaderboardLine.ID, leaderboardLine.TotalScore)

	errch <- nil
}

func (a *PgLeaderboardAggregator) GetParticipants() ([]string, error) {
	var wallets []string
	a.db.Raw("SELECT DISTINCT wallet_address FROM domain_events").Scan(&wallets)
	return wallets, nil
}

func (a *PgLeaderboardAggregator) GetParticipantEvents(wallet string) ([]DomainEvent, error) {
	var events []DomainEvent
	a.db.Where("wallet_address = ?", wallet).Find(&events)
	return events, nil
}

func NewPgAggregrator(db *gorm.DB) *PgLeaderboardAggregator {
	return &PgLeaderboardAggregator{
		db: db,
	}
}

func createTempTable(db *gorm.DB) {
	db.Exec("CREATE TABLE tmp_leaderboard_lines AS SELECT * FROM leaderboard_lines WHERE false")
}

func backupLeaderboardLines(db *gorm.DB) {
	db.Exec("CREATE TEMP TABLE bck_leaderboard_lines AS SELECT * FROM leaderboard_lines")
}

func hotSwapTables(db *gorm.DB) {
	db.Exec("DROP TABLE leaderboard_lines")
	db.Exec("ALTER TABLE tmp_leaderboard_lines RENAME TO leaderboard_lines")
}

func cleanupTmpTables(db *gorm.DB) {
	db.Exec("DROP TABLE bck_leaderboard_lines")
}
