package kedalaunch

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/socketmode"
)

// handleChangeAction opens the duration-change modal from an accepted response.
func (c *kedaLaunchCommand) handleChangeAction(evt *socketmode.Event, client *socketmode.Client) {
	if !ackIfPresent(evt, client, nil) {
		return
	}

	interaction, ok := evt.Data.(slack.InteractionCallback)
	if !ok {
		slog.Warn("unexpected change action payload", "payload_type", fmt.Sprintf("%T", evt.Data))
		return
	}

	metadata, err := metadataFromChangeAction(interaction)
	if err != nil {
		slog.Error("failed to decode change action metadata", "error", err)
		return
	}
	if interaction.ResponseURL != "" {
		// Block action payloads can provide a fresher response URL than the button metadata.
		metadata.ResponseURL = interaction.ResponseURL
	}

	encoded, err := encodeKedaRequestMetadata(metadata)
	if err != nil {
		slog.Error("failed to encode change modal metadata", "error", err)
		c.postEphemeralError(context.Background(), metadata.ResponseURL, "Failed to open change form.", false)
		return
	}

	if _, err := c.api.OpenViewContext(context.Background(), interaction.TriggerID, buildKedaChangeModal(metadata, encoded)); err != nil {
		slog.Error("failed to open change modal", "error", err)
		c.postEphemeralError(context.Background(), metadata.ResponseURL, "Failed to open change form.", false)
	}
}

// handleChangeSubmission resends the original request with only the duration updated.
func (c *kedaLaunchCommand) handleChangeSubmission(evt *socketmode.Event, client *socketmode.Client) {
	interaction, ok := evt.Data.(slack.InteractionCallback)
	if !ok {
		slog.Warn("unexpected change submission payload", "payload_type", fmt.Sprintf("%T", evt.Data))
		ackIfPresent(evt, client, nil)
		return
	}

	// Invalid modal input must be returned in the ack response for Slack to show field errors.
	req, responseURL, fieldErrors := parseChangeSubmission(interaction.View)
	if len(fieldErrors) > 0 {
		ackIfPresent(evt, client, slack.NewErrorsViewSubmissionResponse(fieldErrors))
		return
	}

	// Ack before the external KEDA request to avoid Slack's interactive timeout.
	ackIfPresent(evt, client, nil)

	accepted, err := c.launchRequest(req)
	if err != nil {
		slog.Error(
			"KEDA launch request failed",
			"error", err,
			"requestId", req.RequestID,
			"scaledObject.namespace", req.ScaledObject.Namespace,
			"scaledObject.name", req.ScaledObject.Name,
		)
		c.postEphemeralError(context.Background(), responseURL, "Launch request failed.", false)
		return
	}

	if err := c.postWebhook(context.Background(), responseURL, acceptedLaunchMessage(accepted, req, responseURL, true)); err != nil {
		slog.Error("failed to post launch response", "error", err)
	}
}
