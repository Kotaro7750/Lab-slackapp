package slack_responder

import (
	"context"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/socketmode"
)

// SlackResponderIF groups the Slack API operations that handlers depend on.
type SlackResponderIF interface {
	AckWithSuccess(evt *socketmode.Event, client *socketmode.Client) error
	AckWithViewResponse(evt *socketmode.Event, client *socketmode.Client, viewError *slack.ViewSubmissionResponse) error
	AckWithUnrecoverableError(evt *socketmode.Event, client *socketmode.Client, err error) error
	PostEphemeralError(ctx context.Context, responseURL, text string, replaceOriginal bool)
	PostWebhook(context.Context, string, *slack.WebhookMessage) error
	OpenViewContext(ctx context.Context, triggerID string, view slack.ModalViewRequest) (*slack.ViewResponse, error)
}
