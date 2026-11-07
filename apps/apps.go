// Copyright (C) 2026 NodeByte LTD

package apps

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"popplio/db"
	"popplio/state"
	"popplio/types"
	"popplio/validators/timex"

	"github.com/disgoorg/snowflake/v2"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"

	"github.com/PlexiOSS/Keel/uapi"
)

var Stable = true

type Behaviour struct {
	ID                  string
	Name                string
	Description         string
	LockedQuestions     []string
	AllowedForBanned    bool
	BannedOnly          bool
	ReviewLogic         func(d uapi.RouteData, resp types.AppResponse, reason string, approved bool) (string, error)
	ExtraLogic          func(d uapi.RouteData, p types.Position, answers map[string]string) error
	Channel             func() snowflake.ID
	PositionDescription func(d uapi.RouteData, p types.Position) string
}

var Behaviours = map[string]Behaviour{
	"staff": {
		ID:          "staff",
		Name:        "Staff Team",
		Description: "Approving gives the applicant the Awaiting Staff role in the Discord server, if they're in it.",
		ReviewLogic: reviewLogicStaff,
	},
	"banappeal": {
		ID:               "banappeal",
		Name:             "Ban Appeal",
		Description:      "Only banned members can apply. Approving unbans them from the Discord server. Posts go to the ban appeals channel.",
		AllowedForBanned: true,
		BannedOnly:       true,
		ReviewLogic:      reviewLogicBanAppeal,
		Channel: func() snowflake.ID {
			return state.Config.Channels.BanAppeals
		},
		PositionDescription: func(d uapi.RouteData, p types.Position) string {
			return "User <@" + d.Auth.ID + "> wants to be unbanned now? 🤔"
		},
	},
	"resubmit": {
		ID:              "resubmit",
		Name:            "Bot Resubmission",
		Description:     "Checks the applicant can manage the bot in the id answer, then puts the denied bot back in the review queue.",
		LockedQuestions: []string{"id"},
		ExtraLogic:      extraLogicResubmit,
	},
	"certification": {
		ID:              "certification",
		Name:            "Bot Certification",
		Description:     "Checks the bot in the id answer is approved, theirs and old enough to apply. Approving certifies it.",
		LockedQuestions: []string{"id"},
		ExtraLogic:      extraLogicCert,
		ReviewLogic:     reviewLogicCert,
	},
	"server_certification": {
		ID:              "server_certification",
		Name:            "Server Certification",
		Description:     "Checks the server in the id answer is approved, theirs and old enough to apply. Approving certifies it.",
		LockedQuestions: []string{"id"},
		ExtraLogic:      extraLogicCertServer,
		ReviewLogic:     reviewLogicCertServer,
	},
}

const cacheTTL = 30 * time.Second

var (
	cacheMu   sync.Mutex
	cached    []types.Position
	cachedAt  time.Time
	cacheFull bool
)

func Invalidate() {
	cacheMu.Lock()
	cacheFull = false
	cacheMu.Unlock()
}

func FromRow(row db.AppPosition) (types.Position, error) {
	var questions []types.Question

	if err := json.Unmarshal(row.Questions, &questions); err != nil {
		return types.Position{}, fmt.Errorf("position %s has invalid questions: %w", row.ID, err)
	}

	if row.Tags == nil {
		row.Tags = []string{}
	}

	p := types.Position{
		ID:                 row.ID,
		Name:               row.Name,
		Info:               row.Info,
		Tags:               row.Tags,
		Questions:          questions,
		Hidden:             row.Hidden,
		Closed:             row.Closed,
		RequiresMainServer: row.RequiresMainServer,
		Cooldown:           timex.Duration(time.Duration(row.CooldownSecs) * time.Second),
		ApprovalNote:       row.ApprovalNote,
	}

	if row.HelpUrl.Valid {
		p.HelpURL = row.HelpUrl.String
	}

	if row.HelpLabel.Valid {
		p.HelpLabel = row.HelpLabel.String
	}

	if row.Behaviour.Valid {
		if b, ok := Behaviours[row.Behaviour.String]; ok {
			p.Behaviour = b.ID
			p.ReviewLogic = b.ReviewLogic
			p.ExtraLogic = b.ExtraLogic
			p.Channel = b.Channel
			p.PositionDescription = b.PositionDescription
			p.AllowedForBanned = b.AllowedForBanned
			p.BannedOnly = b.BannedOnly
		}
	}

	return p, nil
}

func List(ctx context.Context) ([]types.Position, error) {
	cacheMu.Lock()
	defer cacheMu.Unlock()

	if cacheFull && time.Since(cachedAt) < cacheTTL {
		return cached, nil
	}

	rows, err := db.New(state.Pool).ListAppPositions(ctx)

	if err != nil {
		return nil, err
	}

	out := make([]types.Position, 0, len(rows))

	for _, row := range rows {
		p, err := FromRow(row)

		if err != nil {
			state.Logger.Error("Skipping an application position that couldn't be loaded", zap.Error(err), zap.String("id", row.ID))
			continue
		}

		out = append(out, p)
	}

	cached, cachedAt, cacheFull = out, time.Now(), true

	return out, nil
}

func FindPosition(ctx context.Context, id string) (*types.Position, error) {
	row, err := db.New(state.Pool).GetAppPosition(ctx, id)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	p, err := FromRow(row)

	if err != nil {
		return nil, err
	}

	return &p, nil
}
