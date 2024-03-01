// Code generated by github.com/99designs/gqlgen, DO NOT EDIT.

package model

type BoostForValue struct {
	Value      string `json:"value"`
	TotalScore string `json:"total_score"`
	Boost      string `json:"boost"`
}

type Categories struct {
	Fund    string `json:"fund"`
	Farming string `json:"farming"`
	Other   string `json:"other"`
}

type Leaderboard struct {
	Data     []*LeaderboardLineData `json:"data"`
	PageInfo *PageInfo              `json:"page_info"`
}

type LeaderboardLineData struct {
	ID            string          `json:"id"`
	WalletAddress string          `json:"wallet_address"`
	Points        []*PointDetails `json:"points"`
	Categories    *Categories     `json:"categories"`
	TotalScore    string          `json:"total_score"`
	Position      int             `json:"position"`
}

type Metadata struct {
	Slot        *string `json:"slot,omitempty"`
	ProjectName *string `json:"project_name,omitempty"`
}

type NextBoostForValue struct {
	Missing    string `json:"missing"`
	TotalScore string `json:"total_score"`
	Boost      string `json:"boost"`
}

type PageInfo struct {
	MaxPage         int  `json:"max_page"`
	Page            int  `json:"page"`
	Limit           int  `json:"limit"`
	Count           int  `json:"count"`
	HasNextPage     bool `json:"has_next_page"`
	HasPreviousPage bool `json:"has_previous_page"`
}

type Pagination struct {
	Page  int `json:"page"`
	Limit int `json:"limit"`
}

type PointDetails struct {
	Rule     *string   `json:"rule,omitempty"`
	Value    *int      `json:"value,omitempty"`
	Metadata *Metadata `json:"metadata,omitempty"`
}

type Query struct {
}
