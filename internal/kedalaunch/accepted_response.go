package kedalaunch

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	domainclient "github.com/Kotaro7750/keda-launcher-scaler/pkg/client"
	"github.com/slack-go/slack"
)

// acceptedRequestMetadata is the shared follow-up contract owned by the accepted response artifact.
type acceptedRequestMetadata struct {
	RequestID   string `json:"request_id"`
	Namespace   string `json:"namespace"`
	Name        string `json:"name"`
	Duration    string `json:"duration"`
	ResponseURL string `json:"response_url"`
}

type acceptedRequestAction struct {
	Metadata    acceptedRequestMetadata
	ResponseURL string
}

const kedaCancelActionID = "cancel_keda_launch_request"

// acceptedLaunchMessage builds the ephemeral Slack response for an accepted launch.
func acceptedLaunchMessage(accepted domainclient.AcceptedRequest, req domainclient.LaunchRequest, responseURL string, replaceOriginal bool) *slack.WebhookMessage {
	metadata := acceptedRequestMetadata{
		RequestID:   req.RequestID,
		Namespace:   req.ScaledObject.Namespace,
		Name:        req.ScaledObject.Name,
		Duration:    req.Duration.String(),
		ResponseURL: responseURL,
	}
	encodedMetadata, err := encodeAcceptedRequestMetadata(metadata)
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
		changeButton := slack.NewButtonBlockElement(kedaChangeActionID, encodedMetadata, slack.NewTextBlockObject(slack.PlainTextType, "Change duration", false, false))
		changeButton.WithStyle(slack.StylePrimary)
		cancelButton := slack.NewButtonBlockElement(kedaCancelActionID, encodedMetadata, slack.NewTextBlockObject(slack.PlainTextType, "Cancel", false, false))
		cancelButton.WithStyle(slack.StyleDanger)
		blocks = append(blocks, slack.NewActionBlock("keda_launch_actions", changeButton, cancelButton))
	}

	return &slack.WebhookMessage{
		Text:            "Launch request accepted.",
		ResponseType:    slack.ResponseTypeEphemeral,
		ReplaceOriginal: replaceOriginal,
		Blocks:          &slack.Blocks{BlockSet: blocks},
	}
}

// canceledLaunchMessage replaces the accepted response after a successful cancel.
func canceledLaunchMessage(deleted domainclient.DeletedRequest, responseURL string) *slack.WebhookMessage {
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

func encodeAcceptedRequestMetadata(metadata acceptedRequestMetadata) (string, error) {
	raw, err := json.Marshal(metadata)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func decodeAcceptedRequestMetadata(value string) (acceptedRequestMetadata, error) {
	var metadata acceptedRequestMetadata
	if strings.TrimSpace(value) == "" {
		return acceptedRequestMetadata{}, fmt.Errorf("metadata is empty")
	}
	if err := json.Unmarshal([]byte(value), &metadata); err != nil {
		return acceptedRequestMetadata{}, err
	}
	if metadata.RequestID == "" || metadata.Namespace == "" || metadata.Name == "" || metadata.ResponseURL == "" {
		return acceptedRequestMetadata{}, fmt.Errorf("missing required metadata")
	}
	return metadata, nil
}
