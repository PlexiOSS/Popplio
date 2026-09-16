// Copyright (C) 2026 NodeByte LTD

package events

import (
	"fmt"

	"popplio/webhooks/core/events"

	"github.com/PlexiOSS/Keel/ptr"

	"github.com/disgoorg/disgo/discord"

	"github.com/PlexiOSS/Keel/dovewing/dovetypes"
)

type WebhookNewReviewData struct {
	ReviewID    string `json:"review_id" description:"The ID of the review"`
	Content     string `json:"content" description:"The content of the review"`
	Stars       int32  `json:"stars" description:"The number of stars the auther gave to the review"`
	OwnerReview bool   `json:"owner_review" description:"Whether or not the review was left by the owner of the entity"`
}

func (v WebhookNewReviewData) TargetTypes() []string {
	return []string{
		"bot",
		"server",
		"team",
	}
}

func (n WebhookNewReviewData) Event() string {
	return "NEW_REVIEW"
}

func (n WebhookNewReviewData) Summary() string {
	return "New Review"
}

func (n WebhookNewReviewData) Description() string {
	return "This webhook is sent when a user creates a new review on an entity."
}

func (n WebhookNewReviewData) CreateDiscordEmbed(creator *dovetypes.PlatformUser, targets events.Target) *discord.Embed {

	var baseURL string

	switch {
	case targets.Bot != nil:
		baseURL = "https://omniplex.gg/bots/" + targets.GetID()
	case targets.Server != nil:
		baseURL = "https://omniplex.gg/servers/" + targets.GetID()
	case targets.Team != nil:
		baseURL = "https://omniplex.gg/teams/" + targets.GetID()
	default:
		baseURL = "https://omniplex.gg/" + targets.GetID()
	}

	return &discord.Embed{
		URL: baseURL,
		Thumbnail: &discord.EmbedResource{
			URL: targets.GetAvatarURL(),
		},
		Title:       "📝 New Review!",
		Description: ":heart: " + creator.DisplayName + " has left a review for " + targets.GetTargetName(),
		Color:       0x8A6BFD,
		Fields: []discord.EmbedField{
			{
				Name:   "Review ID",
				Value:  n.ReviewID,
				Inline: ptr.TruePtr,
			},
			{
				Name:   "User ID",
				Value:  creator.ID,
				Inline: ptr.TruePtr,
			},
			{
				Name:   "Stars",
				Value:  fmt.Sprintf("%d/5", n.Stars),
				Inline: ptr.TruePtr,
			},
			{
				Name: "Review Content",
				Value: func() string {
					if len(n.Content) > 1000 {
						return n.Content[:1000] + "..."
					}

					return n.Content
				}(),
				Inline: ptr.TruePtr,
			},
			{
				Name: "Owner Review",
				Value: func() string {
					if n.OwnerReview {
						return "Yes"
					}

					return "No"
				}(),
			},
			{
				Name:   "Review Page",
				Value:  "[View " + targets.GetDisplayName() + "](" + baseURL + ")",
				Inline: ptr.TruePtr,
			},
		},
	}
}

func init() {
	events.AddEvent(WebhookNewReviewData{})
}
