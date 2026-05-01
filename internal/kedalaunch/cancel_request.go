package kedalaunch

import (
	"context"
	"fmt"
	"log/slog"

	domainclient "github.com/Kotaro7750/keda-launcher-scaler/pkg/client"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/socketmode"
)

var cancelActionAck = ackIfPresent

// handleCancelAction acknowledges the interactive event, then deletes the accepted request once.
func (c *kedaLaunchCommand) handleCancelAction(evt *socketmode.Event, client *socketmode.Client) {
	if !cancelActionAck(evt, client, nil) {
		return
	}

	interaction, ok := evt.Data.(slack.InteractionCallback)
	if !ok {
		slog.Warn("unexpected cancel action payload", "payload_type", fmt.Sprintf("%T", evt.Data))
		return
	}

	action, err := acceptedRequestActionFromCancelAction(interaction)
	if err != nil {
		slog.Error("failed to decode cancel action metadata", "error", err)
		if action.ResponseURL != "" {
			c.postEphemeralError(context.Background(), action.ResponseURL, "Failed to cancel launch request.", false)
		}
		return
	}

	req := domainclient.DeleteRequest{
		RequestID: action.Metadata.RequestID,
		ScaledObject: domainclient.ScaledObject{
			Namespace: action.Metadata.Namespace,
			Name:      action.Metadata.Name,
		},
	}

	// Post the final canceled artifact separately from the upstream delete result.
	deleted, err := c.cancelRequest(req)
	if err != nil {
		slog.Error(
			"KEDA cancel request failed",
			"error", err,
			"requestId", req.RequestID,
			"scaledObject.namespace", req.ScaledObject.Namespace,
			"scaledObject.name", req.ScaledObject.Name,
		)
		c.postEphemeralError(context.Background(), action.Metadata.ResponseURL, "Launch request was not canceled and might still be active.", false)
		return
	}

	if err := c.postWebhook(context.Background(), action.Metadata.ResponseURL, canceledLaunchMessage(deleted, action.Metadata.ResponseURL)); err != nil {
		slog.Error(
			"failed to post cancel success response",
			"error", err,
			"requestId", req.RequestID,
			"scaledObject.namespace", req.ScaledObject.Namespace,
			"scaledObject.name", req.ScaledObject.Name,
		)
	}
}

func acceptedRequestActionFromCancelAction(interaction slack.InteractionCallback) (acceptedRequestAction, error) {
	if len(interaction.ActionCallback.BlockActions) == 0 {
		return acceptedRequestAction{ResponseURL: interaction.ResponseURL}, fmt.Errorf("missing block action")
	}
	metadata, err := decodeAcceptedRequestMetadata(interaction.ActionCallback.BlockActions[0].Value)
	if err != nil {
		return acceptedRequestAction{ResponseURL: interaction.ResponseURL}, err
	}
	return acceptedRequestAction{
		Metadata:    metadata,
		ResponseURL: interaction.ResponseURL,
	}, nil
}
