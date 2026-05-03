package handler

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Kotaro7750/Lab-slackapp/internal/kedalaunch/ui"
	domainclient "github.com/Kotaro7750/keda-launcher-scaler/pkg/client"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/socketmode"
)

// HandleCancelAction acknowledges the interactive event, then deletes the accepted request once.
func (k *KedaLaunchHandler) HandleCancelAction(evt *socketmode.Event, client *socketmode.Client) {
	interaction, ok := evt.Data.(slack.InteractionCallback)
	if !ok {
		if err := k.slackResponder.AckWithUnrecoverableError(evt, client, fmt.Errorf("unexpected cancel action payload type: %T", evt.Data)); err != nil {
			slog.Error("failed to ack with error", "error", err)
		}
		return
	}

	launchRequestMetadata, err := ui.LaunchRequestMetadataFromAction(interaction)
	if err != nil {
		if err := k.slackResponder.AckWithUnrecoverableError(evt, client, fmt.Errorf("invalid cancel action metadata: %w", err)); err != nil {
			slog.Error("failed to ack with error", "error", err)
		}
		return
	}
	if interaction.ResponseURL != "" {
		// Use the accepted-message action response URL so the follow-up post can replace that artifact.
		launchRequestMetadata.ResponseURL = interaction.ResponseURL
	}

	if err := k.slackResponder.AckWithSuccess(evt, client); err != nil {
		slog.Error("failed to ack", "error", err)
		return
	}

	req := domainclient.DeleteRequest{
		RequestID: launchRequestMetadata.RequestID,
		ScaledObject: domainclient.ScaledObject{
			Namespace: launchRequestMetadata.Namespace,
			Name:      launchRequestMetadata.Name,
		},
	}

	// Post the final canceled artifact separately from the upstream delete result.
	deleted, err := k.kedaLauncher.CancelRequest(req)
	if err != nil {
		slog.Error(
			"KEDA cancel request failed",
			"error", err,
			"requestId", req.RequestID,
			"scaledObject.namespace", req.ScaledObject.Namespace,
			"scaledObject.name", req.ScaledObject.Name,
		)
		k.slackResponder.PostEphemeralError(context.Background(), launchRequestMetadata.ResponseURL, "Launch request was not canceled and might still be active.", false)
		return
	}

	if err := k.slackResponder.PostWebhook(context.Background(), launchRequestMetadata.ResponseURL, launchRequestMetadata.BuildCancelMessage(deleted)); err != nil {
		slog.Error(
			"failed to post cancel success response",
			"error", err,
			"requestId", req.RequestID,
			"scaledObject.namespace", req.ScaledObject.Namespace,
			"scaledObject.name", req.ScaledObject.Name,
		)
	}
}
