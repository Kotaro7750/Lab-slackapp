package kedalaunch

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/socketmode"
)

// handleLaunchSubmission sends a valid launch modal submission to KEDA.
func (c *kedaLaunchCommand) handleLaunchSubmission(evt *socketmode.Event, client *socketmode.Client) {
	interaction, ok := evt.Data.(slack.InteractionCallback)
	if !ok {
		slog.Warn("unexpected launch submission payload", "payload_type", fmt.Sprintf("%T", evt.Data))
		ackIfPresent(evt, client, nil)
		return
	}

	// Invalid modal input must be returned in the ack response for Slack to show field errors.
	req, responseURL, fieldErrors := parseLaunchSubmission(interaction.View, c.now())
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

	if err := c.postWebhook(context.Background(), responseURL, acceptedLaunchMessage(accepted, req, responseURL, false)); err != nil {
		slog.Error("failed to post launch response", "error", err)
	}
}
