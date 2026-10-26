// Copyright (C) 2026 NodeByte LTD

package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/disgoorg/disgo"
	"github.com/disgoorg/disgo/discord"
	"gopkg.in/yaml.v3"
)

var validName = regexp.MustCompile(`^[a-z0-9_]{2,32}$`)

type emojiFile struct {
	name string
	path string
	kind discord.IconType
}

func main() {
	dir := flag.String("dir", "../omniplex/public/brand", "brand kit directory")
	configPath := flag.String("config", "config.yaml", "Popplio config file")
	dryRun := flag.Bool("dry-run", false, "list what would be uploaded without uploading")
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

	files, err := collect(*dir)
	if err != nil {
		fail(err)
	}
	fmt.Printf("found %d emojis in %s\n", len(files), *dir)

	for _, app := range []struct{ label, token string }{
		{"popplio", cfg.DiscordAuth.Token},
		{"arcadia", cfg.Arcadia.Token},
		{"infernoplex", cfg.Infernoplex.Token},
	} {
		if app.token == "" {
			fmt.Printf("[%s] no token configured, skipping\n", app.label)
			continue
		}
		if err := upload(app.label, app.token, files, *dryRun); err != nil {
			fmt.Printf("[%s] failed: %v\n", app.label, err)
		}
	}
}

func collect(dir string) ([]emojiFile, error) {
	var files []emojiFile
	add := func(pattern string, kind discord.IconType, name func(string) string) error {
		matches, err := filepath.Glob(filepath.Join(dir, pattern))
		if err != nil {
			return err
		}
		for _, m := range matches {
			n := name(strings.TrimSuffix(filepath.Base(m), filepath.Ext(m)))
			if !validName.MatchString(n) {
				return fmt.Errorf("%s: %q is not a valid emoji name", m, n)
			}
			files = append(files, emojiFile{name: n, path: m, kind: kind})
		}
		return nil
	}
	same := func(s string) string { return s }
	if err := add("emojis/*.png", discord.IconTypePNG, same); err != nil {
		return nil, err
	}
	if err := add("emojis/animated/*.gif", discord.IconTypeGIF, same); err != nil {
		return nil, err
	}
	if err := add("themes/*/emoji-*.png", discord.IconTypePNG, func(s string) string { return strings.TrimPrefix(s, "emoji-") }); err != nil {
		return nil, err
	}
	if err := add("bots/*/emoji-*.png", discord.IconTypePNG, func(s string) string { return strings.TrimPrefix(s, "emoji-") }); err != nil {
		return nil, err
	}
	return files, nil
}

func upload(label, token string, files []emojiFile, dryRun bool) error {
	client, err := disgo.New(token)
	if err != nil {
		return err
	}
	appID := client.ApplicationID()

	existing, err := client.Rest().GetApplicationEmojis(appID)
	if err != nil {
		return fmt.Errorf("listing application emojis: %w", err)
	}
	have := make(map[string]bool, len(existing))
	for _, e := range existing {
		have[e.Name] = true
	}

	uploaded, skipped := 0, 0
	for _, f := range files {
		if have[f.name] {
			skipped++
			continue
		}
		if dryRun {
			fmt.Printf("[%s] would upload %s\n", label, f.name)
			continue
		}
		data, err := os.ReadFile(f.path)
		if err != nil {
			return err
		}
		e, err := client.Rest().CreateApplicationEmoji(appID, discord.EmojiCreate{
			Name:  f.name,
			Image: *discord.NewIconRaw(f.kind, data),
		})
		if err != nil {
			fmt.Printf("[%s] %s: %v\n", label, f.name, err)
			continue
		}
		uploaded++
		fmt.Printf("[%s] uploaded %s\n", label, e.Mention())
	}
	fmt.Printf("[%s] app %s: %d uploaded, %d already there\n", label, appID, uploaded, skipped)
	return nil
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
