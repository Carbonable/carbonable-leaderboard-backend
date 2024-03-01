package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.
// Code generated by github.com/99designs/gqlgen version v0.17.43

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/NethermindEth/juno/core/felt"
	"github.com/carbonable/leaderboard/graph/model"
	appdb "github.com/carbonable/leaderboard/internal/db"
	"github.com/carbonable/leaderboard/internal/leaderboard"
	"github.com/holiman/uint256"
)

// Leaderboard is the resolver for the leaderboard field.
func (r *queryResolver) Leaderboard(ctx context.Context, pagination model.Pagination) (*model.Leaderboard, error) {
	var lines []leaderboardQueryResult
	var count int64
	r.db.Model(&leaderboard.LeaderboardLine{}).Count(&count)
	r.db.Raw(appdb.PaginateRaw(leaderboardQuery, pagination.Page, pagination.Limit)).Scan(&lines)
	data := dbModelToGqlModel(lines)

	totalPages := math.Ceil(float64(count) / float64(pagination.Limit))

	return &model.Leaderboard{
		Data: data,

		PageInfo: &model.PageInfo{
			MaxPage:         int(totalPages),
			Page:            pagination.Page,
			Limit:           pagination.Limit,
			Count:           int(count),
			HasNextPage:     pagination.Page < int(totalPages),
			HasPreviousPage: pagination.Page > 1,
		},
	}, nil
}

// LeaderboardForWallet is the resolver for the leaderboardForWallet field.
func (r *queryResolver) LeaderboardForWallet(ctx context.Context, walletAddress string) (*model.LeaderboardLineData, error) {
	var walletFelt felt.Felt
	err := walletFelt.UnmarshalJSON([]byte(walletAddress))
	if err != nil {
		return nil, err
	}
	var line leaderboardQueryResult
	res := r.db.Raw(leaderboardQueryWhere, walletFelt.String()).Scan(&line)
	data := itemToGqlModel(line)
	return data, res.Error
}

// BoostForWallet is the resolver for the boostForWallet field.
func (r *queryResolver) BoostForWallet(ctx context.Context, walletAddress string, valueToBuy int, address string, slot int) (*model.BoostForValue, error) {
	scm := leaderboard.MintPageCalculatorManager()
	// generate fake event to enable computation
	buy := leaderboard.DomainEvent{
		RecordedAt:    time.Now(),
		EventName:     "minter:buy",
		WalletAddress: walletAddress,
		Data: map[string]string{
			"value": fmt.Sprintf("0x%x", valueToBuy*1000000),
		},
	}
	pr := leaderboard.NewPersonnalRanking(walletAddress, []leaderboard.DomainEvent{buy})
	leaderboardLine := pr.ComputeScore(scm)

	bc := leaderboard.DefaultProjectValueBoostCalculator()
	boost, _, _ := bc.GetInterval(uint64(valueToBuy))
	return &model.BoostForValue{
		Value:      fmt.Sprintf("%d", valueToBuy),
		TotalScore: leaderboardLine.TotalScore,
		Boost:      boost.String(),
	}, nil
}

// NextBoostForWallet is the resolver for the nextBoostForWallet field.
func (r *queryResolver) NextBoostForWallet(ctx context.Context, walletAddress string, valueToBuy int, address string, slot int) (*model.NextBoostForValue, error) {
	scm := leaderboard.MintPageCalculatorManager()
	// generate fake event to enable computation
	buy := leaderboard.DomainEvent{
		RecordedAt:    time.Now(),
		EventName:     "minter:buy",
		WalletAddress: walletAddress,
		Data: map[string]string{
			"value": fmt.Sprintf("0x%x", valueToBuy*1000000),
		},
	}
	pr := leaderboard.NewPersonnalRanking(walletAddress, []leaderboard.DomainEvent{buy})
	leaderboardLine := pr.ComputeScore(scm)
	bc := leaderboard.DefaultProjectValueBoostCalculator()

	_, next, boost := bc.GetInterval(uint64(valueToBuy))
	var missing uint256.Int
	missing.Sub(next, uint256.NewInt(uint64(valueToBuy)))

	return &model.NextBoostForValue{
		Missing:    missing.String(),
		TotalScore: leaderboardLine.TotalScore,
		Boost:      boost.String(),
	}, nil
}

// Query returns QueryResolver implementation.
func (r *Resolver) Query() QueryResolver { return &queryResolver{r} }

type queryResolver struct{ *Resolver }

// !!! WARNING !!!
// The code below was going to be deleted when updating resolvers. It has been copied here so you have
// one last chance to move it out of harms way if you want. There are two reasons this happens:
//   - When renaming or deleting a resolver the old code will be put in here. You can safely delete
//     it when you're done.
//   - You have helper methods in this file. Move them out to keep these resolver files clean.
const leaderboardQuery = `WITH leaderboard AS (
   SELECT l.*,
	ROW_NUMBER() OVER(ORDER BY l.total_score::INT DESC) AS position
     FROM leaderboard_lines l)
SELECT l.*
	FROM leaderboard l ORDER BY l.total_score::INT DESC;`

const leaderboardQueryWhere = `WITH leaderboard AS (
   SELECT l.*,
	ROW_NUMBER() OVER(ORDER BY l.total_score::INT DESC) AS position
     FROM leaderboard_lines l)
SELECT l.*
	FROM leaderboard l WHERE l.wallet_address = ? ORDER BY l.total_score::INT DESC;`

type leaderboardQueryResult struct {
	leaderboard.LeaderboardLine
	Position int
}

func itemToGqlModel(item leaderboardQueryResult) *model.LeaderboardLineData {
	var points []*model.PointDetails

	for _, point := range item.Points {
		slot := point.Metadata["slot"]
		pName := point.Metadata["project_name"]
		value := int(point.Value)
		points = append(points, &model.PointDetails{Rule: &point.Rule, Value: &value, Metadata: &model.Metadata{Slot: &slot, ProjectName: &pName}})
	}
	return &model.LeaderboardLineData{
		ID:            item.ID.String(),
		WalletAddress: item.WalletAddress,
		Points:        points,
		TotalScore:    item.TotalScore,
		Categories: &model.Categories{
			Fund:    item.Categories.Fund,
			Farming: item.Categories.Farming,
			Other:   item.Categories.Other,
		},
		Position: item.Position,
	}
}

func dbModelToGqlModel(dbModel []leaderboardQueryResult) []*model.LeaderboardLineData {
	var gqlModel []*model.LeaderboardLineData
	for _, line := range dbModel {
		gqlModel = append(gqlModel, itemToGqlModel(line))
	}
	return gqlModel
}