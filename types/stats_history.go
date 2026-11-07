// Copyright (C) 2026 NodeByte LTD

package types

import "time"

type StatsHistoryMonth struct {
	Month      string `json:"month" description:"The month, as YYYY-MM"`
	Votes      *int64 `json:"votes" description:"Votes cast that month, or null before vote history starts"`
	Voters     *int64 `json:"voters" description:"Different people who voted that month, or null before vote history starts"`
	Approved   *int64 `json:"approved" description:"Listings approved that month, or null before review history starts"`
	Denied     *int64 `json:"denied" description:"Listings denied that month, or null before review history starts"`
	NewBots    int64  `json:"new_bots" description:"Bots submitted that month"`
	NewServers int64  `json:"new_servers" description:"Servers added that month"`
	NewUsers   *int64 `json:"new_users" description:"People who signed up that month, or null before sign-up dates were recorded"`
}

type StatsHistory struct {
	VotesAllTime   int64               `json:"votes_all_time" description:"Every vote cast since vote history starts, including ones cleared by monthly resets"`
	VotesThisMonth int64               `json:"votes_this_month" description:"Votes cast since the last monthly reset"`
	VotersAllTime  int64               `json:"voters_all_time" description:"Different people who have ever voted"`
	VotesSince     *time.Time          `json:"votes_since" description:"When vote history starts"`
	Approvals      int64               `json:"approvals" description:"Listings approved since review history starts"`
	Denials        int64               `json:"denials" description:"Listings denied since review history starts"`
	ReviewsSince   *time.Time          `json:"reviews_since" description:"When review history starts"`
	Months         []StatsHistoryMonth `json:"months" description:"One entry per month, oldest first"`
	GeneratedAt    time.Time           `json:"generated_at" description:"When these numbers were worked out. They're cached for a few minutes"`
}
