package kedalaunch

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/socketmode"
)

// handleSlashCommand acknowledges /launch and opens the initial request modal.
func (c *kedaLaunchCommand) handleSlashCommand(evt *socketmode.Event, client *socketmode.Client) {
	if !ackIfPresent(evt, client, nil) {
		return
	}

	cmd, ok := evt.Data.(slack.SlashCommand)
	if !ok {
		slog.Warn("unexpected slash command payload", "payload_type", fmt.Sprintf("%T", evt.Data))
		return
	}

	// Preserve Slack response context in private metadata for the modal submit.
	metadata, err := encodeLaunchModalMetadata(launchModalMetadata{
		UserID:      cmd.UserID,
		ChannelID:   cmd.ChannelID,
		ResponseURL: cmd.ResponseURL,
	})
	if err != nil {
		slog.Error("failed to encode launch modal metadata", "error", err)
		c.postEphemeralError(context.Background(), cmd.ResponseURL, "Failed to open launch form.", false)
		return
	}

	if _, err := c.api.OpenViewContext(context.Background(), cmd.TriggerID, buildKedaLaunchModal(metadata)); err != nil {
		slog.Error("failed to open launch modal", "error", err)
		c.postEphemeralError(context.Background(), cmd.ResponseURL, "Failed to open launch form.", false)
	}
}
