package handler

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Kotaro7750/Lab-slackapp/internal/kedalaunch/ui"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/socketmode"
)

// HandleSlashCommand acknowledges /launch and opens the initial request modal.
func (k *KedaLaunchHandler) HandleSlashCommand(evt *socketmode.Event, client *socketmode.Client) {
	cmd, ok := evt.Data.(slack.SlashCommand)
	if !ok {
		if err := k.slackResponder.AckWithUnrecoverableError(evt, client, fmt.Errorf("unexpected slash command payload type: %T", evt.Data)); err != nil {
			slog.Error("failed to ack with error", "error", err)
		}
		return
	}

	if err := k.slackResponder.AckWithSuccess(evt, client); err != nil {
		slog.Error("failed to ack", "error", err)
	}

	candidates, err := k.kedaLauncher.ListScaledObjects()
	if err != nil {
		slog.Error("failed to load launch targets", "error", err)
		k.slackResponder.PostEphemeralError(context.Background(), cmd.ResponseURL, "Failed to load launch targets.", false)
		return
	}
	if len(candidates) == 0 {
		k.slackResponder.PostEphemeralError(context.Background(), cmd.ResponseURL, "No launch targets are currently available.", false)
		return
	}

	// Preserve Slack response context in private metadata for the modal submit.
	metadata := ui.CommandInvocationMetadata{
		UserID:      cmd.UserID,
		ChannelID:   cmd.ChannelID,
		ResponseURL: cmd.ResponseURL,
	}

	if _, err := k.slackResponder.OpenViewContext(context.Background(), cmd.TriggerID, metadata.BuildLaunchModal(candidates)); err != nil {
		slog.Error("failed to open launch modal", "error", err)
		k.slackResponder.PostEphemeralError(context.Background(), cmd.ResponseURL, "Failed to open launch form.", false)
	}
}
