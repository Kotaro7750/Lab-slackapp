package ui

import (
	"fmt"
	"time"

	domainclient "github.com/Kotaro7750/keda-launcher-scaler/pkg/client"
	"github.com/slack-go/slack"
)

// BuildCancelMessage replaces the accepted response after a successful cancel.
func (metadata *LaunchRequestMetadata) BuildCancelMessage(deleted domainclient.DeletedRequest) *slack.WebhookMessage {
	return &slack.WebhookMessage{
		Text:            "Launch request canceled.",
		ResponseType:    slack.ResponseTypeEphemeral,
		ReplaceOriginal: true,
		Blocks: &slack.Blocks{BlockSet: []slack.Block{
			slack.NewSectionBlock(
				slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf(
					"*Launch request canceled*\n*Request ID:* `%s`\n*ScaledObject:* `%s/%s`\n*Effective window:* `%s` - `%s`",
					deleted.RequestID,
					deleted.ScaledObject.Namespace,
					deleted.ScaledObject.Name,
					deleted.EffectiveStart.Format(time.RFC3339),
					deleted.EffectiveEnd.Format(time.RFC3339),
				), false, false),
				nil,
				nil,
			),
		}},
	}
}
