package get_ticket

import (
	"testing"

	"popplio/types"

	"github.com/disgoorg/disgo/discord"
)

func TestCollectMentions(t *testing.T) {
	msg := types.Message{
		Content: "<@510065483693817867> <@&805761849601294336> see <#871440804638519337> and <@!123456789012345678>",
		Embeds: []discord.Embed{{
			Description: "ping <@&111111111111111111>",
			Fields:      []discord.EmbedField{{Name: "who", Value: "<@222222222222222222>"}},
			Footer:      &discord.EmbedFooter{Text: "in <#333333333333333333>"},
		}},
	}
	texts := messageTexts(msg)

	users, roles, channels := map[string]struct{}{}, map[string]struct{}{}, map[string]struct{}{}
	collect(userMention, texts, users)
	collect(roleMention, texts, roles)
	collect(channelMention, texts, channels)

	for _, id := range []string{"510065483693817867", "123456789012345678", "222222222222222222"} {
		if _, ok := users[id]; !ok {
			t.Errorf("missing user %s", id)
		}
	}
	for _, id := range []string{"805761849601294336", "111111111111111111"} {
		if _, ok := roles[id]; !ok {
			t.Errorf("missing role %s", id)
		}
	}
	for _, id := range []string{"871440804638519337", "333333333333333333"} {
		if _, ok := channels[id]; !ok {
			t.Errorf("missing channel %s", id)
		}
	}
	if _, ok := users["805761849601294336"]; ok {
		t.Error("role mention was collected as a user")
	}
	if len(users) != 3 || len(roles) != 2 || len(channels) != 2 {
		t.Errorf("unexpected counts users=%d roles=%d channels=%d", len(users), len(roles), len(channels))
	}
}
