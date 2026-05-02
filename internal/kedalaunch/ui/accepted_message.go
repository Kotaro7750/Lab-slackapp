package ui

import (
	"fmt"
	"time"

	domainclient "github.com/Kotaro7750/keda-launcher-scaler/pkg/client"
	"github.com/slack-go/slack"
)

// BuildAcceptedMessage builds the ephemeral Slack response for an accepted launch.
func (metadata *LaunchRequestMetadata) BuildAcceptedMessage(accepted domainclient.AcceptedRequest, req domainclient.LaunchRequest, replaceOriginal bool) *slack.WebhookMessage {
	encodedMetadata := metadata.Encode()
	changeButton := slack.NewButtonBlockElement(KedaChangeActionID, encodedMetadata, slack.NewTextBlockObject(slack.PlainTextType, "Change duration", false, false))
	changeButton.WithStyle(slack.StylePrimary)
	cancelButton := slack.NewButtonBlockElement(KedaCancelActionID, encodedMetadata, slack.NewTextBlockObject(slack.PlainTextType, "Cancel", false, false))
	cancelButton.WithStyle(slack.StyleDanger)

	blocks := []slack.Block{
		slack.NewSectionBlock(
			slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf(
				"*Launch request accepted*\n*Request ID:* `%s`\n*ScaledObject:* `%s/%s`\n*Duration:* `%s`\n*Effective window:* `%s` - `%s`",
				accepted.RequestID,
				accepted.ScaledObject.Namespace,
				accepted.ScaledObject.Name,
				req.Duration.String(),
				accepted.EffectiveStart.Format(time.RFC3339),
				accepted.EffectiveEnd.Format(time.RFC3339),
			), false, false),
			nil,
			nil,
		),
		slack.NewActionBlock("keda_launch_actions", changeButton, cancelButton),
	}

	return &slack.WebhookMessage{
		Text:            "Launch request accepted.",
		ResponseType:    slack.ResponseTypeEphemeral,
		ReplaceOriginal: replaceOriginal,
		Blocks:          &slack.Blocks{BlockSet: blocks},
	}
}
