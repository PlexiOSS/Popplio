package types

import "time"

const (
	MaxBotCommands        = 100
	MaxBotCommandNameLen  = 100
	MaxBotCommandDescLen  = 500
	MaxBotCommandUsageLen = 200
)

// @ci table=bot_commands, unfilled=1
type BotCommand struct {
	ID          string    `db:"id" json:"id"`
	Name        string    `db:"name" json:"name"`
	Description string    `db:"description" json:"description"`
	Usage       string    `db:"usage" json:"usage"`
	Category    string    `db:"category" json:"category"`
	Position    int       `db:"position" json:"position"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time `db:"updated_at" json:"updated_at"`
}

type BotCommandList struct {
	Commands []BotCommand `json:"commands"`
}

// BotCommandInput is one command in a PUT /bots/{id}/commands body — the
// whole list is replaced each time, same convention bots.extra_links
// already uses, since this is a small freely-reordered list rather than a
// history.
type BotCommandInput struct {
	Name        string `json:"name" validate:"required,min=1,max=100"`
	Description string `json:"description" validate:"max=500"`
	Usage       string `json:"usage" validate:"max=200"`
	Category    string `json:"category" validate:"max=50"`
}

type UpdateBotCommands struct {
	Commands []BotCommandInput `json:"commands" validate:"max=100,dive"`
}
