package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"gopkg.in/yaml.v3"
)

type cfg struct {
	Meta struct {
		PostgresURL string `yaml:"postgres_url"`
	} `yaml:"meta"`
}

func main() {
	ctx := context.Background()
	raw, err := os.ReadFile("config.yaml")
	if err != nil {
		panic(err)
	}
	var c cfg
	if err := yaml.Unmarshal(raw, &c); err != nil {
		panic(err)
	}
	pool, err := pgxpool.New(ctx, c.Meta.PostgresURL)
	if err != nil {
		panic(err)
	}
	defer pool.Close()

	fmt.Println("=== automated_vote_resets (last 20) ===")
	rows, err := pool.Query(ctx, "SELECT id, created_at FROM automated_vote_resets ORDER BY created_at DESC LIMIT 20")
	if err != nil {
		fmt.Println("ERROR:", err)
	} else {
		for rows.Next() {
			var id string
			var createdAt any
			rows.Scan(&id, &createdAt)
			fmt.Println(id, createdAt)
		}
		rows.Close()
	}

	fmt.Println()
	fmt.Println("=== Octoflow bot lookup ===")
	var botID, name string
	var approxVotes int
	err = pool.QueryRow(ctx, `
		SELECT b.bot_id, u.username, b.approximate_votes
		FROM bots b
		JOIN internal_user_cache__discord u ON u.id = b.bot_id
		WHERE u.username ILIKE '%Octoflow%'
		LIMIT 1
	`).Scan(&botID, &name, &approxVotes)
	if err != nil {
		fmt.Println("lookup via internal_user_cache failed:", err)
	} else {
		fmt.Println("bot_id:", botID, "name:", name, "approximate_votes:", approxVotes)
	}

	if botID == "" {
		// fallback: maybe dovewing cache table is named differently; try bots table alone with a raw id guess skipped
		fmt.Println("No bot_id resolved, skipping entity_votes dump")
		return
	}

	fmt.Println()
	fmt.Println("=== entity_votes for Octoflow (last 30) ===")
	evRows, err := pool.Query(ctx, `
		SELECT itag, author, upvote, void, void_reason, immutable, created_at, voided_at
		FROM entity_votes
		WHERE target_id = $1 AND target_type = 'bot'
		ORDER BY created_at DESC
		LIMIT 30
	`, botID)
	if err != nil {
		fmt.Println("ERROR:", err)
		return
	}
	defer evRows.Close()
	for evRows.Next() {
		var itag, author string
		var upvote, void, immutable bool
		var voidReason *string
		var createdAt any
		var voidedAt *string
		if err := evRows.Scan(&itag, &author, &upvote, &void, &voidReason, &immutable, &createdAt, &voidedAt); err != nil {
			fmt.Println("scan err:", err)
			continue
		}
		reason := ""
		if voidReason != nil {
			reason = *voidReason
		}
		voided := ""
		if voidedAt != nil {
			voided = *voidedAt
		}
		fmt.Printf("author=%s upvote=%v void=%v immutable=%v created=%v voided_at=%s reason=%q\n", author, upvote, void, immutable, createdAt, voided, reason)
	}
}
