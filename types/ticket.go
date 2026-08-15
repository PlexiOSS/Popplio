package types

import (
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/infinitybotlist/eureka/dovewing/dovetypes"
	"github.com/jackc/pgx/v5/pgtype"
)

type Ticket struct {
	ID            string                  `db:"id" json:"id"`
	ChannelID     string                  `db:"channel_id" json:"channel_id"`
	TopicID       string                  `db:"topic_id" json:"topic_id"`
	Issue         string                  `db:"issue" json:"issue"`
	TicketContext map[string]string       `db:"ticket_context" json:"ticket_context"`
	Messages      []Message               `db:"messages" json:"messages"`
	UserID        string                  `db:"user_id" json:"-"`
	Author        *dovetypes.PlatformUser `db:"-" json:"author"`
	CloseUserID   pgtype.Text             `db:"close_user_id" json:"-"`
	CloseUser     *dovetypes.PlatformUser `db:"-" json:"close_user"`
	Open          bool                    `db:"open" json:"open"`
	CreatedAt     time.Time               `db:"created_at" json:"created_at"`
	EncKey        pgtype.Text             `db:"enc_key" json:"enc_key"`
}

type Message struct {
	ID          string                  `json:"id"`
	Timestamp   time.Time               `json:"timestamp"` // Not in DB, but generated from snowflake ID
	Content     string                  `json:"content"`
	Embeds      []discord.Embed         `json:"embeds"`
	AuthorID    string                  `json:"author_id"`
	Author      *dovetypes.PlatformUser `json:"author"`
	Attachments []Attachment            `json:"attachments"`
}

type Attachment struct {
	ID          string   `json:"id"`           // ID of the attachment within the ticket
	Name        string   `json:"name"`         // Name of the attachment
	ContentType string   `json:"content_type"` // Content type of the attachment
	Size        int      `json:"size"`         // Size of the attachment in bytes
	Errors      []string `json:"errors"`       // Non-fatal errors that occurred while uploading the attachment
}

// TicketTopic is a predefined category a ticket can be filed under.
type TicketTopic struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type TicketTopicList struct {
	Topics []TicketTopic `json:"topics"`
}

type CreateTicket struct {
	Topic   string `json:"topic" validate:"required" msg:"Topic is required."`
	Issue   string `json:"issue" validate:"required,min=10,max=200" msg:"Issue must be between 10 and 200 characters."`
	Message string `json:"message" validate:"required,min=20,max=4000" msg:"Message must be between 20 and 4000 characters."`
}

type CreateTicketMessage struct {
	Content string `json:"content" validate:"required,min=1,max=4000" msg:"Message content is required."`
}

type PatchTicket struct {
	Open bool `json:"open"`
}

type TicketList struct {
	Tickets []Ticket `json:"tickets"`
}
