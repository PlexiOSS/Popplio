// Copyright (C) 2026 NodeByte LTD

package linkedroles

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"maps"
	"time"

	"popplio/db"
	"popplio/state"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/rest"
	"github.com/disgoorg/snowflake/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

const (
	MaxBadges    = 5
	PlatformName = "Omniplex"
)

var (
	ErrDisabled = errors.New("linked roles are not configured")
	oauth       = rest.NewOAuth2(rest.NewClient(""))
)

func Enabled() bool {
	return state.Config.Meta.LinkedRolesKey != ""
}

func gcm() (cipher.AEAD, error) {
	key, err := base64.StdEncoding.DecodeString(state.Config.Meta.LinkedRolesKey)

	if err != nil {
		return nil, fmt.Errorf("linked_roles_key is not valid base64: %w", err)
	}

	if len(key) != 32 {
		return nil, fmt.Errorf("linked_roles_key must decode to 32 bytes, got %d", len(key))
	}

	block, err := aes.NewCipher(key)

	if err != nil {
		return nil, err
	}

	return cipher.NewGCM(block)
}

func encrypt(plain string) ([]byte, error) {
	g, err := gcm()

	if err != nil {
		return nil, err
	}

	nonce := make([]byte, g.NonceSize())

	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	return g.Seal(nonce, nonce, []byte(plain), nil), nil
}

func decrypt(sealed []byte) (string, error) {
	g, err := gcm()

	if err != nil {
		return "", err
	}

	if len(sealed) < g.NonceSize() {
		return "", errors.New("stored token is too short")
	}

	plain, err := g.Open(nil, sealed[:g.NonceSize()], sealed[g.NonceSize():], nil)

	if err != nil {
		return "", err
	}

	return string(plain), nil
}

func ClientID() string {
	return state.Config.Infernoplex.ClientID
}

func clientID() (snowflake.ID, error) {
	return snowflake.Parse(ClientID())
}

func RegisterMetadata(ctx context.Context) error {
	if !Enabled() {
		return nil
	}

	badges, err := db.New(state.Pool).ListLinkedRoleBadges(ctx)

	if err != nil {
		return err
	}

	if len(badges) > MaxBadges {
		badges = badges[:MaxBadges]
	}

	records := make([]discord.ApplicationRoleConnectionMetadata, 0, len(badges))

	for _, b := range badges {
		desc := b.Description

		if desc == "" {
			desc = "Has the " + b.Name + " badge on Omniplex"
		}

		if len(desc) > 200 {
			desc = desc[:200]
		}

		name := b.Name

		if len(name) > 100 {
			name = name[:100]
		}

		records = append(records, discord.ApplicationRoleConnectionMetadata{
			Type:        discord.ApplicationRoleConnectionMetadataTypeBooleanEqual,
			Key:         b.ID,
			Name:        name,
			Description: desc,
		})
	}

	appID, err := clientID()

	if err != nil {
		return err
	}

	apps := rest.NewApplications(rest.NewClient(state.Config.Infernoplex.Token))
	_, err = apps.UpdateApplicationRoleConnectionMetadata(appID, records)

	return err
}

func RegisterMetadataAsync() {
	go func() {
		if err := RegisterMetadata(context.Background()); err != nil {
			state.Logger.Error("Failed to register linked roles metadata with Discord", zap.Error(err))
		}
	}()
}

func metadataFor(ctx context.Context, userID string) (map[string]string, error) {
	q := db.New(state.Pool)

	linked, err := q.ListLinkedRoleBadges(ctx)

	if err != nil {
		return nil, err
	}

	has, err := q.GetUserLinkedRoleBadgeIDs(ctx, userID)

	if err != nil {
		return nil, err
	}

	owned := make(map[string]bool, len(has))

	for _, id := range has {
		owned[id] = true
	}

	m := make(map[string]string, len(linked))

	for i, b := range linked {
		if i >= MaxBadges {
			break
		}

		if owned[b.ID] {
			m[b.ID] = "1"
		} else {
			m[b.ID] = "0"
		}
	}

	return m, nil
}

func push(accessToken, username string, metadata map[string]string) error {
	id, err := clientID()

	if err != nil {
		return err
	}

	name := PlatformName
	md := metadata

	_, err = oauth.UpdateCurrentUserApplicationRoleConnection(accessToken, id, discord.ApplicationRoleConnectionUpdate{
		PlatformName:     &name,
		PlatformUsername: &username,
		Metadata:         &md,
	})

	return err
}

func toAny(m map[string]string) map[string]any {
	out := make(map[string]any, len(m))

	for k, v := range m {
		out[k] = v
	}

	return out
}

func fromAny(m map[string]any) map[string]string {
	out := make(map[string]string, len(m))

	for k, v := range m {
		if s, ok := v.(string); ok {
			out[k] = s
		}
	}

	return out
}

type Result struct {
	UserID   string   `json:"user_id"`
	Username string   `json:"username"`
	Badges   []string `json:"badges"`
}

func Connect(ctx context.Context, code, redirectURI string) (*Result, error) {
	if !Enabled() {
		return nil, ErrDisabled
	}

	id, err := clientID()

	if err != nil {
		return nil, err
	}

	tok, err := oauth.GetAccessToken(id, state.Config.Infernoplex.ClientSecret, code, redirectURI)

	if err != nil {
		return nil, fmt.Errorf("exchanging the Discord code: %w", err)
	}

	user, err := oauth.GetCurrentUser(tok.AccessToken)

	if err != nil {
		return nil, fmt.Errorf("fetching the Discord user: %w", err)
	}

	userID := user.ID.String()
	metadata, err := metadataFor(ctx, userID)

	if err != nil {
		return nil, err
	}

	if err := push(tok.AccessToken, user.Username, metadata); err != nil {
		return nil, fmt.Errorf("updating the Discord role connection: %w", err)
	}

	access, err := encrypt(tok.AccessToken)

	if err != nil {
		return nil, err
	}

	refresh, err := encrypt(tok.RefreshToken)

	if err != nil {
		return nil, err
	}

	err = db.New(state.Pool).UpsertUserRoleConnection(ctx, db.UpsertUserRoleConnectionParams{
		UserID:       userID,
		AccessToken:  access,
		RefreshToken: refresh,
		ExpiresAt:    pgtype.Timestamptz{Time: time.Now().Add(tok.ExpiresIn * time.Second), Valid: true},
		LastMetadata: toAny(metadata),
	})

	if err != nil {
		return nil, err
	}

	names := []string{}
	linked, _ := db.New(state.Pool).ListLinkedRoleBadges(ctx)

	for _, b := range linked {
		if metadata[b.ID] == "1" {
			names = append(names, b.Name)
		}
	}

	return &Result{UserID: userID, Username: user.Username, Badges: names}, nil
}

func syncOne(ctx context.Context, c db.ListUserRoleConnectionsRow) error {
	q := db.New(state.Pool)

	metadata, err := metadataFor(ctx, c.UserID)

	if err != nil {
		return err
	}

	if maps.Equal(metadata, fromAny(c.LastMetadata)) {
		return nil
	}

	accessToken, err := decrypt(c.AccessToken)

	if err != nil {
		return err
	}

	if time.Until(c.ExpiresAt.Time) < 10*time.Minute {
		refreshToken, err := decrypt(c.RefreshToken)

		if err != nil {
			return err
		}

		id, err := clientID()

		if err != nil {
			return err
		}

		tok, err := oauth.RefreshAccessToken(id, state.Config.Infernoplex.ClientSecret, refreshToken)

		if err != nil {
			state.Logger.Info("Linked roles refresh token rejected, forgetting the connection", zap.String("userID", c.UserID), zap.Error(err))
			return q.DeleteUserRoleConnection(ctx, c.UserID)
		}

		access, err := encrypt(tok.AccessToken)

		if err != nil {
			return err
		}

		refresh, err := encrypt(tok.RefreshToken)

		if err != nil {
			return err
		}

		if err := q.UpdateUserRoleConnectionTokens(ctx, db.UpdateUserRoleConnectionTokensParams{
			UserID:       c.UserID,
			AccessToken:  access,
			RefreshToken: refresh,
			ExpiresAt:    pgtype.Timestamptz{Time: time.Now().Add(tok.ExpiresIn * time.Second), Valid: true},
		}); err != nil {
			return err
		}

		accessToken = tok.AccessToken
	}

	username := c.UserID

	if u, err := oauth.GetCurrentUser(accessToken); err == nil {
		username = u.Username
	}

	if err := push(accessToken, username, metadata); err != nil {
		return err
	}

	return q.UpdateUserRoleConnectionMetadata(ctx, db.UpdateUserRoleConnectionMetadataParams{UserID: c.UserID, LastMetadata: toAny(metadata)})
}

func SyncAll(ctx context.Context) error {
	if !Enabled() {
		return nil
	}

	conns, err := db.New(state.Pool).ListUserRoleConnections(ctx)

	if err != nil {
		return err
	}

	for _, c := range conns {
		if err := syncOne(ctx, db.ListUserRoleConnectionsRow(c)); err != nil {
			state.Logger.Warn("Failed to sync a linked roles connection", zap.String("userID", c.UserID), zap.Error(err))
		}
	}

	return nil
}

func SyncUserAsync(userID string) {
	if !Enabled() {
		return
	}

	go func() {
		ctx := context.Background()
		c, err := db.New(state.Pool).GetUserRoleConnection(ctx, userID)

		if errors.Is(err, pgx.ErrNoRows) {
			return
		}

		if err != nil {
			state.Logger.Warn("Failed to load a linked roles connection", zap.String("userID", userID), zap.Error(err))
			return
		}

		if err := syncOne(ctx, db.ListUserRoleConnectionsRow(c)); err != nil {
			state.Logger.Warn("Failed to sync a linked roles connection", zap.String("userID", userID), zap.Error(err))
		}
	}()
}
