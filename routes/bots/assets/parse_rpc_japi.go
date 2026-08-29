// Copyright (C) 2026 NodeByte LTD

package assets

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"popplio/db"
	"popplio/state"
	"popplio/types"

	"go.uber.org/zap"

	"github.com/PlexiOSS/Keel/dovewing"
	"github.com/PlexiOSS/Keel/jsonimpl"
)

type japidata struct {
	Cached bool `json:"cached"`
	Data   struct {
		Message     string `json:"message,omitempty"`
		Application *struct {
			ID          string   `json:"id"`
			BotPublic   bool     `json:"bot_public"`
			Description string   `json:"description"`
			Tags        []string `json:"tags"`
		} `json:"application"`
		Bot *struct {
			ID                    string   `json:"id"`
			ApproximateGuildCount int      `json:"approximate_guild_count"`
			Username              string   `json:"username"`
			AvatarURL             string   `json:"avatarURL"`
			AvatarHash            string   `json:"avatarHash"`
			PublicFlagsArray      []string `json:"public_flags_array"`
		} `json:"bot"`
	} `json:"data"`
}

const japiUserAgent = "Popplio (Omniplex API, +https://omniplex.gg)"

func CheckBot(ctx context.Context, fallbackBotId, clientId string) (*types.DiscordBotMeta, error) {
	var fetchErrors = map[string]string{}

	cidInt, err := strconv.ParseInt(clientId, 10, 64)

	if err != nil {
		return nil, fmt.Errorf("error parsing client id as int: %s", clientId)
	}

	cli := http.Client{
		Timeout: 5 * time.Second,
	}

	var metadata *types.DiscordBotMeta

	rpcReq, err := http.NewRequestWithContext(ctx, "GET", state.Config.Meta.PopplioProxy+"/api/v10/applications/"+clientId+"/rpc", nil)

	if err != nil {
		return nil, err
	}

	rpcResp, err := cli.Do(rpcReq)

	switch {
	case err != nil:
		fetchErrors["rpc.doError"] = err.Error()
	case rpcResp.StatusCode == http.StatusTooManyRequests:
		fetchErrors["rpc.ratelimit"] = fmt.Sprintf("we're being ratelimited by discord! Please try again in %s seconds", rpcResp.Header.Get("Retry-After"))
	case rpcResp.StatusCode != http.StatusOK:
		fetchErrors["rpc.status"] = fmt.Sprintf("we couldn't find a bot with that client ID! Status code: %d", rpcResp.StatusCode)
	default:
		defer rpcResp.Body.Close()

		var rpcData struct {
			BotPublic bool `json:"bot_public"`
		}

		if err := jsonimpl.UnmarshalReader(rpcResp.Body, &rpcData); err != nil {
			fetchErrors["rpc.parse"] = err.Error()
			break
		}

		resolvedBotId := fallbackBotId

		if resolvedBotId == "" {
			if cidInt < 132550911590400000 {
				return nil, errors.New("fallbackNeeded")
			}

			resolvedBotId = clientId
		}

		user, userErr := dovewing.GetUser(ctx, resolvedBotId, state.DovewingPlatformDiscord)

		if userErr != nil {
			return nil, errors.New("the client id provided is not an actual bot id")
		}

		metadata = &types.DiscordBotMeta{
			BotID:       resolvedBotId,
			ClientID:    clientId,
			Name:        user.Username,
			Avatar:      user.Avatar,
			BotPublic:   rpcData.BotPublic,
			FetchErrors: fetchErrors,
			Fallback:    false,
		}
	}

	if metadata == nil {
		req, err := http.NewRequestWithContext(ctx, "GET", "https://japi.rest/discord/v1/application/"+clientId, nil)

		if err != nil {
			return nil, fmt.Errorf("error creating request: %s", err.Error())
		}

		req.Header.Set("User-Agent", japiUserAgent)

		japiKey := state.Config.JAPI.Key
		if japiKey != "" {
			req.Header.Set("Authorization", japiKey)
		}

		resp, err := cli.Do(req)

		if err != nil {
			fetchErrors["japi.doError"] = err.Error()
			resp = &http.Response{
				Status:     "418 I'm a teapot",
				StatusCode: http.StatusTeapot,
			}
		}

		switch {
		case resp.StatusCode == http.StatusTooManyRequests:
			fetchErrors["japi.ratelimit"] = fmt.Sprintf("we're being ratelimited by our anti-abuse provider! Please try again in %s seconds", resp.Header.Get("Retry-After"))
		case resp.StatusCode > 500:
			fetchErrors["japi.server"] = fmt.Sprintf("the JAPI server is having issues! Status code: %d", resp.StatusCode)
		case resp.StatusCode == http.StatusRequestTimeout:
			fetchErrors["japi.timeout"] = "the JAPI server is taking too long to respond!"
		case resp.StatusCode == http.StatusTeapot:
			fetchErrors["japi.status"] = "the JAPI server did not respond to the request correctly!"
		case resp.StatusCode > 400:
			return nil, fmt.Errorf("we couldn't find a bot with that client ID! Status code: %d", resp.StatusCode)
		case resp.StatusCode > 200:
			fetchErrors["japi.status"] = fmt.Sprintf("the JAPI server returned an invalid status code! Status code: %d", resp.StatusCode)
		case resp.StatusCode == 200:
			defer resp.Body.Close()

			var data japidata

			err = jsonimpl.UnmarshalReader(resp.Body, &data)

			if err != nil {
				return nil, err
			}

			if data.Data.Message != "" {
				fetchErrors["japi.message"] = data.Data.Message
			}

			if data.Data.Bot == nil || data.Data.Application == nil {
				return nil, errors.New("woah there, we found an application with no associated bot?")
			}

			if data.Data.Bot.ID == "" {
				return nil, errors.New("woah there, we found an application with no associated bot?")
			}

			if !data.Cached {
				state.Logger.With(
					zap.String("bot_id", data.Data.Bot.ID),
					zap.String("client_id", clientId),
				).Info("JAPI cache MISS")
			} else {
				state.Logger.With(
					zap.String("bot_id", data.Data.Bot.ID),
					zap.String("client_id", clientId),
				).Info("JAPI cache HIT")
			}

			user, err := dovewing.GetUser(ctx, data.Data.Bot.ID, state.DovewingPlatformDiscord)

			if err != nil {
				return nil, errors.New("please contact support, an error has occured while trying to fetch basic info")
			}

			metadata = &types.DiscordBotMeta{
				BotID:       data.Data.Bot.ID,
				ClientID:    clientId,
				Name:        user.Username,
				GuildCount:  data.Data.Bot.ApproximateGuildCount,
				BotPublic:   data.Data.Application.BotPublic,
				Avatar:      user.Avatar,
				Flags:       data.Data.Bot.PublicFlagsArray,
				Description: data.Data.Application.Description,
				Tags:        data.Data.Application.Tags,
				FetchErrors: fetchErrors,
				Fallback:    true,
			}
		}
	}

	if metadata == nil {
		var reasons []string

		for _, msg := range fetchErrors {
			reasons = append(reasons, msg)
		}

		return nil, fmt.Errorf("failed to fetch bot metadata: %s", strings.Join(reasons, "; "))
	}

	q := db.New(state.Pool)

	count, err := q.CountBotByID(ctx, metadata.BotID)

	if err != nil {
		return nil, errors.New("failed to check if bot is already in the database")
	}

	if count > 0 {
		listType, err := q.GetBotType(ctx, metadata.BotID)

		if err != nil {
			return nil, errors.New("failed to get bot type")
		}

		if listType == "" {
			return nil, errors.New("list type is invalid, contact support")
		}

		metadata.ListType = listType
	}

	return metadata, nil
}
