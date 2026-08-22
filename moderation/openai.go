package moderation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"time"

	"popplio/state"
)

type Result struct {
	Flagged    bool     `json:"flagged"`
	Categories []string `json:"categories"`
}

type moderationRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

type moderationResponse struct {
	Results []struct {
		Flagged    bool            `json:"flagged"`
		Categories map[string]bool `json:"categories"`
	} `json:"results"`
}

var httpClient = &http.Client{Timeout: 15 * time.Second}

func CheckText(ctx context.Context, inputs ...string) (Result, error) {
	apiKey := state.Config.Meta.OpenAIAPIKey

	if apiKey == "" {
		return Result{}, nil
	}

	nonEmpty := make([]string, 0, len(inputs))

	for _, in := range inputs {
		if in != "" {
			nonEmpty = append(nonEmpty, in)
		}
	}

	if len(nonEmpty) == 0 {
		return Result{}, nil
	}

	body, err := json.Marshal(moderationRequest{
		Model: "omni-moderation-latest",
		Input: nonEmpty,
	})

	if err != nil {
		return Result{}, fmt.Errorf("failed to marshal moderation request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.openai.com/v1/moderations", bytes.NewReader(body))

	if err != nil {
		return Result{}, fmt.Errorf("failed to build moderation request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := httpClient.Do(req)

	if err != nil {
		return Result{}, fmt.Errorf("moderation request failed: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return Result{}, fmt.Errorf("moderation endpoint returned status %d", resp.StatusCode)
	}

	var parsed moderationResponse

	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return Result{}, fmt.Errorf("failed to decode moderation response: %w", err)
	}

	result := Result{Categories: []string{}}

	for _, r := range parsed.Results {
		if r.Flagged {
			result.Flagged = true
		}

		for category, flagged := range r.Categories {
			if flagged {
				result.Categories = append(result.Categories, category)
			}
		}
	}

	slices.Sort(result.Categories)
	result.Categories = slices.Compact(result.Categories)

	return result, nil
}
