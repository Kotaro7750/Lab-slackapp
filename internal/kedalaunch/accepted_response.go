package kedalaunch

import (
	"fmt"
	"log/slog"
	"time"

	domainclient "github.com/Kotaro7750/keda-launcher-scaler/pkg/client"
	"github.com/slack-go/slack"
)

// acceptedLaunchMessage builds the ephemeral Slack response for an accepted launch.
func acceptedLaunchMessage(accepted domainclient.AcceptedRequest, req domainclient.LaunchRequest, responseURL string, replaceOriginal bool) *slack.WebhookMessage {
	metadata := kedaRequestMetadata{
		RequestID:   req.RequestID,
		Namespace:   req.ScaledObject.Namespace,
		Name:        req.ScaledObject.Name,
		Duration:    req.Duration.String(),
		ResponseURL: responseURL,
	}
	encodedMetadata, err := encodeKedaRequestMetadata(metadata)
	if err != nil {
		slog.Error("failed to encode change button metadata", "error", err)
		// Keep the accepted response visible even if the follow-up change action is unavailable.
		encodedMetadata = ""
	}

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
	}
	if encodedMetadata != "" {
		button := slack.NewButtonBlockElement(kedaChangeActionID, encodedMetadata, slack.NewTextBlockObject(slack.PlainTextType, "Change duration", false, false))
		button.WithStyle(slack.StylePrimary)
		blocks = append(blocks, slack.NewActionBlock("keda_launch_actions", button))
	}

	return &slack.WebhookMessage{
		Text:            "Launch request accepted.",
		ResponseType:    slack.ResponseTypeEphemeral,
		ReplaceOriginal: replaceOriginal,
		Blocks:          &slack.Blocks{BlockSet: blocks},
	}
}
