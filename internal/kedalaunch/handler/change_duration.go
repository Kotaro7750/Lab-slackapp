package handler

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Kotaro7750/Lab-slackapp/internal/kedalaunch/ui"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/socketmode"
)

// HandleChangeAction opens the duration-change modal from an accepted response.
func (k *KedaLaunchHandler) HandleChangeAction(evt *socketmode.Event, client *socketmode.Client) {
	interaction, ok := evt.Data.(slack.InteractionCallback)
	if !ok {
		if err := k.slackResponder.AckWithUnrecoverableError(evt, client, fmt.Errorf("unexpected change action payload type: %T", evt.Data)); err != nil {
			slog.Error("failed to ack with error", "error", err)
		}
		return
	}

	metadata, err := ui.LaunchRequestMetadataFromAction(interaction)
	if err != nil {
		if err := k.slackResponder.AckWithUnrecoverableError(evt, client, fmt.Errorf("invalid change action metadata: %w", err)); err != nil {
			slog.Error("failed to ack with error", "error", err)
		}
		return
	}
	if interaction.ResponseURL != "" {
		// Use the accepted-message action response URL so the follow-up submit can replace that artifact.
		metadata.ResponseURL = interaction.ResponseURL
	}

	if err := k.slackResponder.AckWithSuccess(evt, client); err != nil {
		slog.Error("failed to ack", "error", err)
		return
	}

	if _, err := k.slackResponder.OpenViewContext(context.Background(), interaction.TriggerID, metadata.BuildChangeDurationModal()); err != nil {
		slog.Error("failed to open change modal", "error", err)
		k.slackResponder.PostEphemeralError(context.Background(), metadata.ResponseURL, "Failed to open change form.", false)
	}
}

// HandleChangeSubmission resends the original request with only the duration updated.
func (k *KedaLaunchHandler) HandleChangeSubmission(evt *socketmode.Event, client *socketmode.Client) {
	interaction, ok := evt.Data.(slack.InteractionCallback)
	if !ok {
		if err := k.slackResponder.AckWithUnrecoverableError(evt, client, fmt.Errorf("unexpected change submission payload type: %T", evt.Data)); err != nil {
			slog.Error("failed to ack with error", "error", err)
		}
		return
	}

	metadata, err := ui.DecodeLaunchRequestMetadata(interaction.View.PrivateMetadata)
	if err != nil {
		if err := k.slackResponder.AckWithUnrecoverableError(evt, client, fmt.Errorf("invalid change modal metadata: %w", err)); err != nil {
			slog.Error("failed to ack with error", "error", err)
		}
		return
	}

	// Invalid modal input must be returned in the ack response for Slack to show field errors.
	req, fieldErrors := metadata.ParseChangeDurationModal(interaction.View)
	if len(fieldErrors) > 0 {
		if err := k.slackResponder.AckWithViewResponse(evt, client, slack.NewErrorsViewSubmissionResponse(fieldErrors)); err != nil {
			slog.Error("failed to ack with view error", "error", err)
		}
		return
	}

	if err := k.slackResponder.AckWithSuccess(evt, client); err != nil {
		slog.Error("failed to ack with error", "error", err)
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

	if err := k.slackResponder.PostWebhook(context.Background(), metadata.ResponseURL, metadata.BuildAcceptedMessage(accepted, req, true)); err != nil {
		slog.Error("failed to post launch response", "error", err)
	}
}
