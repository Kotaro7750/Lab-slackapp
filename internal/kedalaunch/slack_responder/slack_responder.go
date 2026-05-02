package slack_responder

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/socketmode"
)

// SlackResponder keeps direct Slack SDK calls behind a small feature-local seam.
type SlackResponder struct {
	apiClient *slack.Client
}

// NewSlackResponder builds the feature-local adapter over the shared Slack client.
func NewSlackResponder(apiClient *slack.Client) *SlackResponder {
	return &SlackResponder{apiClient: apiClient}
}

// ack sends the immediate Socket Mode acknowledgement that Slack expects for interactive events.
func (s *SlackResponder) ack(evt *socketmode.Event, client *socketmode.Client, payload any) error {
	if evt.Request == nil {
		return fmt.Errorf("no ack handle in event")
	}

	if err := client.Ack(*evt.Request, payload); err != nil {
		return fmt.Errorf("failed to ack event: %w", err)
	}
	return nil
}

// AckWithViewResponse returns field-level validation errors to a Slack modal submission.
func (s *SlackResponder) AckWithViewResponse(evt *socketmode.Event, client *socketmode.Client, viewResponse *slack.ViewSubmissionResponse) error {
	if err := s.ack(evt, client, viewResponse); err != nil {
		return fmt.Errorf("failed to ack event with view response: %w", err)
	}
	return nil
}

// AckWithSuccess sends the normal success acknowledgement for an interactive event.
func (s *SlackResponder) AckWithSuccess(evt *socketmode.Event, client *socketmode.Client) error {
	if err := s.ack(evt, client, nil); err != nil {
		return fmt.Errorf("failed to ack event with success: %w", err)
	}
	return nil
}

// AckWithUnrecoverableError ends the Slack interaction with a generic user-facing failure response.
func (s *SlackResponder) AckWithUnrecoverableError(evt *socketmode.Event, client *socketmode.Client, err error) error {
	slog.Error("ack event with unrecoverable error", "error", err)

	errPayload := map[string]interface{}{
		"response_type": "ephemeral",
		"text":          "An unrecoverable error occurred. Please contact the administrator.",
	}

	if ackErr := s.ack(evt, client, errPayload); ackErr != nil {
		return fmt.Errorf("failed to ack event with unrecoverable error: %w", ackErr)
	}
	return nil
}

// PostEphemeralError sends a short ephemeral error response through Slack's response URL.
func (s *SlackResponder) PostEphemeralError(ctx context.Context, responseURL, text string, replaceOriginal bool) {
	if responseURL == "" {
		return
	}
	if err := slack.PostWebhookContext(ctx, responseURL, errorMessage(text, replaceOriginal)); err != nil {
		slog.Error("failed to post error response", "error", err)
	}
}

// PostWebhook posts a follow-up message through Slack's response URL contract.
func (s *SlackResponder) PostWebhook(ctx context.Context, responseURL string, message *slack.WebhookMessage) error {
	return slack.PostWebhookContext(ctx, responseURL, message)
}

// OpenViewContext opens a modal with the shared Slack Web API client.
func (s *SlackResponder) OpenViewContext(ctx context.Context, triggerID string, view slack.ModalViewRequest) (*slack.ViewResponse, error) {
	return s.apiClient.OpenViewContext(ctx, triggerID, view)
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
