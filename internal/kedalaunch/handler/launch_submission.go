package handler

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Kotaro7750/Lab-slackapp/internal/kedalaunch/ui"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/socketmode"
)

// HandleLaunchSubmission sends a valid launch modal submission to KEDA.
func (k *KedaLaunchHandler) HandleLaunchSubmission(evt *socketmode.Event, client *socketmode.Client) {
	interaction, ok := evt.Data.(slack.InteractionCallback)
	if !ok {
		if err := k.slackResponder.AckWithUnrecoverableError(evt, client, fmt.Errorf("unexpected launch submission payload type: %T", evt.Data)); err != nil {
			slog.Error("failed to ack with error", "error", err)
		}
		return
	}

	metadata, err := ui.DecodeCommandInvocationMetadata(interaction.View.PrivateMetadata)
	if err != nil {
		if err := k.slackResponder.AckWithUnrecoverableError(evt, client, fmt.Errorf("invalid launch modal metadata: %w", err)); err != nil {
			slog.Error("failed to ack with error", "error", err)
		}
		return
	}

	// Invalid modal input must be returned in the ack response for Slack to show field errors.
	req, fieldErrors := metadata.ParseLaunchModal(interaction.View, k.now())
	if len(fieldErrors) > 0 {
		if err := k.slackResponder.AckWithViewResponse(evt, client, slack.NewErrorsViewSubmissionResponse(fieldErrors)); err != nil {
			slog.Error("failed to ack with view error", "error", err)
		}
		return
	}

	if err := k.slackResponder.AckWithSuccess(evt, client); err != nil {
		slog.Error("failed to ack", "error", err)
		return
	}

	accepted, err := k.kedaLauncher.LaunchRequest(req)
	if err != nil {
		slog.Error(
			"KEDA launch request failed",
			"error", err,
			"requestId", req.RequestID,
			"scaledObject.namespace", req.ScaledObject.Namespace,
			"scaledObject.name", req.ScaledObject.Name,
		)
		k.slackResponder.PostEphemeralError(context.Background(), metadata.ResponseURL, "Launch request failed.", false)
		return
	}

	launchRequestMetadata := ui.LaunchRequestMetadata{
		RequestID:   req.RequestID,
		Namespace:   req.ScaledObject.Namespace,
		Name:        req.ScaledObject.Name,
		Duration:    req.Duration.String(),
		ResponseURL: metadata.ResponseURL,
	}

	if err := k.slackResponder.PostWebhook(context.Background(), metadata.ResponseURL, launchRequestMetadata.BuildAcceptedMessage(accepted, req, false)); err != nil {
		slog.Error("failed to post launch response", "error", err)
	}
}
