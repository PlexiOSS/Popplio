// Copyright (C) 2026 NodeByte LTD

package japi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sync/atomic"
	"time"
)

const (
	baseURL   = "https://japi.rest/discord/v1"
	userAgent = "Popplio (Omniplex API, +https://omniplex.gg)"
)

var (
	ErrNotFound    = errors.New("japi: not found")
	ErrRateLimited = errors.New("japi: rate limited")
)

var client = &http.Client{Timeout: 8 * time.Second}

var apiKey atomic.Pointer[string]

func SetKey(key string) {
	apiKey.Store(&key)
}

var snowflakeRe = regexp.MustCompile(`^[0-9]{16,20}$`)

type User struct {
	ID               string   `json:"id"`
	Username         string   `json:"username"`
	GlobalName       *string  `json:"global_name"`
	Avatar           *string  `json:"avatar"`
	Banner           *string  `json:"banner"`
	Bot              bool     `json:"bot"`
	PublicFlagsArray []string `json:"public_flags_array"`
	AvatarURL        *string  `json:"avatarURL"`
	DefaultAvatarURL string   `json:"defaultAvatarURL"`
	BannerURL        *string  `json:"bannerURL"`
}

func (u *User) EffectiveAvatarURL() string {
	if u.AvatarURL != nil && *u.AvatarURL != "" {
		return *u.AvatarURL
	}
	return u.DefaultAvatarURL
}

func (u *User) EffectiveName() string {
	if u.GlobalName != nil && *u.GlobalName != "" {
		return *u.GlobalName
	}
	return u.Username
}

type Application struct {
	Application *struct {
		ID          string   `json:"id"`
		BotPublic   bool     `json:"bot_public"`
		Description string   `json:"description"`
		Tags        []string `json:"tags"`
	} `json:"application"`
	Bot *struct {
		ID                    string   `json:"id"`
		ApproximateGuildCount *int32   `json:"approximate_guild_count"`
		Username              string   `json:"username"`
		GlobalName            *string  `json:"global_name"`
		Avatar                *string  `json:"avatar"`
		AvatarURL             *string  `json:"avatarURL"`
		PublicFlagsArray      []string `json:"public_flags_array"`
	} `json:"bot"`
	Message *string `json:"message"`
}

type envelope[T any] struct {
	Cached bool `json:"cached"`
	Data   T    `json:"data"`
}

func GetUser(ctx context.Context, id string) (*User, error) {
	var out envelope[User]

	if err := get(ctx, "/user/"+id, id, &out); err != nil {
		return nil, err
	}

	if out.Data.ID == "" {
		return nil, ErrNotFound
	}

	return &out.Data, nil
}

func GetApplication(ctx context.Context, id string) (*Application, error) {
	var out envelope[Application]

	if err := get(ctx, "/application/"+id, id, &out); err != nil {
		return nil, err
	}

	if out.Data.Application == nil && out.Data.Bot == nil {
		return nil, ErrNotFound
	}

	return &out.Data, nil
}

func get(ctx context.Context, path, id string, out any) error {
	if !snowflakeRe.MatchString(id) {
		return fmt.Errorf("japi: invalid snowflake %q", id)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+path, nil)

	if err != nil {
		return err
	}

	req.Header.Set("User-Agent", userAgent)

	if key := apiKey.Load(); key != nil && *key != "" {
		req.Header.Set("Authorization", *key)
	}

	resp, err := client.Do(req)

	if err != nil {
		return fmt.Errorf("japi: %w", err)
	}

	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusTooManyRequests:
		return fmt.Errorf("%w (retry after %ss)", ErrRateLimited, resp.Header.Get("Retry-After"))
	case resp.StatusCode == http.StatusNotFound, resp.StatusCode == http.StatusBadRequest:
		_, _ = io.Copy(io.Discard, resp.Body)
		return ErrNotFound
	case resp.StatusCode < 200 || resp.StatusCode > 299:
		_, _ = io.Copy(io.Discard, resp.Body)
		return fmt.Errorf("japi: unexpected status %d", resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("japi: decoding response: %w", err)
	}

	return nil
}
