// Copyright (C) 2026 NodeByte LTD

package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/disgoorg/disgo"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/json"
	"gopkg.in/yaml.v3"
)

func main() {
	dir := flag.String("dir", "../omniplex/public/brand/bots", "bot brand directory")
	configPath := flag.String("config", "config.yaml", "Popplio config file")
	static := flag.Bool("static", false, "use the static PNG avatar instead of the animated GIF")
	only := flag.String("only", "", "comma-separated bots to update (popplio, arcadia, infernoplex); empty means all")
	dryRun := flag.Bool("dry-run", false, "show what would change without changing it")
	flag.Parse()

	raw, err := os.ReadFile(*configPath)
	if err != nil {
		fail(err)
	}

	var cfg struct {
		DiscordAuth struct {
			Token string `yaml:"token"`
		} `yaml:"discord_auth"`
		Arcadia struct {
			Token string `yaml:"token"`
		} `yaml:"arcadia"`
		Infernoplex struct {
			Token string `yaml:"token"`
		} `yaml:"infernoplex"`
	}
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		fail(err)
	}

	want := map[string]bool{}
	for _, b := range strings.Split(*only, ",") {
		if b = strings.TrimSpace(b); b != "" {
			want[b] = true
		}
	}

	for _, bot := range []struct{ key, token string }{
		{"popplio", cfg.DiscordAuth.Token},
		{"arcadia", cfg.Arcadia.Token},
		{"infernoplex", cfg.Infernoplex.Token},
	} {
		if len(want) > 0 && !want[bot.key] {
			continue
		}
		if bot.token == "" {
			fmt.Printf("[%s] no token configured, skipping\n", bot.key)
			continue
		}
		if err := apply(bot.key, bot.token, filepath.Join(*dir, bot.key), *static, *dryRun); err != nil {
			fmt.Printf("[%s] failed: %v\n", bot.key, err)
		}
	}
}

func apply(key, token, dir string, static, dryRun bool) error {
	avatarPath, avatarType := filepath.Join(dir, "avatar-animated.gif"), discord.IconTypeGIF
	if static {
		avatarPath, avatarType = filepath.Join(dir, "avatar-1024.png"), discord.IconTypePNG
	}
	bannerPath := filepath.Join(dir, "discord-profile-banner-1360x480.png")

	avatar, err := os.ReadFile(avatarPath)
	if err != nil {
		return err
	}
	banner, err := os.ReadFile(bannerPath)
	if err != nil {
		return err
	}

	client, err := disgo.New(token)
	if err != nil {
		return err
	}
	self, err := client.Rest().GetCurrentUser("")
	if err != nil {
		return fmt.Errorf("fetching bot user: %w", err)
	}

	if dryRun {
		fmt.Printf("[%s] would set %s (%s) avatar to %s and banner to %s\n", key, self.Username, self.ID, filepath.Base(avatarPath), filepath.Base(bannerPath))
		return nil
	}

	updated, err := client.Rest().UpdateCurrentUser(discord.UserUpdate{
		Avatar: json.NewNullablePtr(*discord.NewIconRaw(avatarType, avatar)),
		Banner: json.NewNullablePtr(*discord.NewIconRaw(discord.IconTypePNG, banner)),
	})
	if err != nil {
		return err
	}

	fmt.Printf("[%s] updated %s: avatar %s, banner %s\n", key, updated.Username, deref(updated.Avatar), deref(updated.Banner))
	return nil
}

func deref(s *string) string {
	if s == nil {
		return "(none)"
	}
	return *s
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
