package kedalaunch

import (
	"fmt"
	"strings"
	"time"

	httpclient "github.com/Kotaro7750/keda-launcher-scaler/pkg/client/http"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/socketmode"
)

// Config contains the runtime settings for the /launch command.
type Config struct {
	CommandName string
	ReceiverURL string
}

// Register wires the /launch command and its related Slack callbacks.
func Register(handler *socketmode.SocketmodeHandler, api *slack.Client, cfg Config) error {
	if !strings.HasPrefix(cfg.CommandName, "/") {
		return fmt.Errorf("SLACK_LAUNCH_COMMAND must start with %q", "/")
	}

	launcher, err := httpclient.New(cfg.ReceiverURL)
	if err != nil {
		return fmt.Errorf("create KEDA launcher client: %w", err)
	}

	command := &kedaLaunchCommand{
		api:         api,
		launcher:    launcher,
		postWebhook: slack.PostWebhookContext,
		now:         time.Now,
	}

	// Register handlers in the same order users move through the launch flow.
	handler.HandleSlashCommand(cfg.CommandName, command.handleSlashCommand)
	handler.HandleViewSubmission(kedaLaunchCallbackID, command.handleLaunchSubmission)
	handler.HandleInteractionBlockAction(kedaChangeActionID, command.handleChangeAction)
	handler.HandleViewSubmission(kedaChangeCallbackID, command.handleChangeSubmission)

	return nil
}
