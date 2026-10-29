// Copyright (C) 2026 NodeByte LTD

package captcha

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"popplio/db"
	"popplio/state"
)

const (
	VerifyURL      = "https://challenges.cloudflare.com/turnstile/v0/siteverify"
	Action         = "vote"
	MaxTokenLength = 2048
)

var (
	ErrUnavailable = errors.New("captcha verification is unavailable")
	ErrInvalid     = errors.New("captcha verification failed")
	ErrExpired     = errors.New("captcha expired or already used")
)

var client = &http.Client{Timeout: 10 * time.Second}

type Solution struct {
	Token string `json:"token"`
}

type siteverifyResponse struct {
	Success    bool     `json:"success"`
	ErrorCodes []string `json:"error-codes"`
	Action     string   `json:"action"`
	Hostname   string   `json:"hostname"`
}

func Verify(ctx context.Context, s Solution) error {
	token := strings.TrimSpace(s.Token)

	if token == "" || len(token) > MaxTokenLength {
		return ErrInvalid
	}

	form := url.Values{
		"secret":   {state.Config.Captcha.TurnstileSecret},
		"response": {token},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, VerifyURL, strings.NewReader(form.Encode()))

	if err != nil {
		return fmt.Errorf("%w: %w", ErrUnavailable, err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	res, err := client.Do(req)

	if err != nil {
		return fmt.Errorf("%w: %w", ErrUnavailable, err)
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: siteverify returned %d", ErrUnavailable, res.StatusCode)
	}

	var out siteverifyResponse

	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return fmt.Errorf("%w: %w", ErrUnavailable, err)
	}

	if !out.Success {
		switch {
		case slices.Contains(out.ErrorCodes, "timeout-or-duplicate"):
			return ErrExpired
		case slices.Contains(out.ErrorCodes, "missing-input-secret"), slices.Contains(out.ErrorCodes, "invalid-input-secret"), slices.Contains(out.ErrorCodes, "internal-error"):
			return fmt.Errorf("%w: %s", ErrUnavailable, strings.Join(out.ErrorCodes, ","))
		}

		return ErrInvalid
	}

	if out.Action != "" && out.Action != Action {
		return ErrInvalid
	}

	return nil
}

func RequiresCaptcha(ctx context.Context, targetType, targetId string) (bool, error) {
	q := db.New(state.Pool)

	var optOut bool
	var err error

	switch targetType {
	case "bot":
		optOut, err = q.GetBotCaptchaOptOut(ctx, targetId)
	case "server":
		optOut, err = q.GetServerCaptchaOptOut(ctx, targetId)
	default:
		return false, nil
	}

	if err != nil {
		return false, fmt.Errorf("failed to check captcha_opt_out: %w", err)
	}

	return !optOut, nil
}
