package kedalaunch

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/socketmode"
)

// ackIfPresent acknowledges a Socket Mode event when Slack provided an ack handle.
func ackIfPresent(evt *socketmode.Event, client *socketmode.Client, payload any) bool {
	if evt.Request == nil {
		return true
	}
	if payload == nil {
		if err := client.Ack(*evt.Request); err != nil {
			slog.Error("failed to acknowledge Slack event", "error", err)
			return false
		}
		return true
	}
	if err := client.Ack(*evt.Request, payload); err != nil {
		slog.Error("failed to acknowledge Slack event", "error", err)
		return false
	}
	return true
}

// postEphemeralError sends a short ephemeral error response through Slack's response URL.
func (c *kedaLaunchCommand) postEphemeralError(ctx context.Context, responseURL, text string, replaceOriginal bool) {
	if responseURL == "" {
		return
	}
	if err := c.postWebhook(ctx, responseURL, errorMessage(text, replaceOriginal)); err != nil {
		slog.Error("failed to post error response", "error", err)
	}
}

// errorMessage builds the Slack webhook payload for an ephemeral error response.
func errorMessage(text string, replaceOriginal bool) *slack.WebhookMessage {
	return &slack.WebhookMessage{
		Text:            text,
		ResponseType:    slack.ResponseTypeEphemeral,
		ReplaceOriginal: replaceOriginal,
		Blocks: &slack.Blocks{BlockSet: []slack.Block{
			slack.NewSectionBlock(slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("*%s*", text), false, false), nil, nil),
		}},
	}
}
