package kedalaunch

import (
	"fmt"
	"strings"

	"github.com/Kotaro7750/Lab-slackapp/internal/kedalaunch/handler"
	"github.com/Kotaro7750/Lab-slackapp/internal/kedalaunch/slack_responder"
	"github.com/Kotaro7750/Lab-slackapp/internal/kedalaunch/ui"
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
func Register(registeree *socketmode.SocketmodeHandler, api *slack.Client, cfg Config) error {
	if !strings.HasPrefix(cfg.CommandName, "/") {
		return fmt.Errorf("SLACK_LAUNCH_COMMAND must start with %q", "/")
	}

	kedaLauncher, err := httpclient.New(cfg.ReceiverURL)
	if err != nil {
		return fmt.Errorf("create KEDA launcher client: %w", err)
	}

	handler := handler.NewKedaLaunchHandler(kedaLauncher, slack_responder.NewSlackResponder(api))

	// Register handlers in the same order users move through the launch flow.
	registeree.HandleSlashCommand(cfg.CommandName, handler.HandleSlashCommand)
	registeree.HandleViewSubmission(ui.KedaLaunchCallbackID, handler.HandleLaunchSubmission)
	registeree.HandleInteractionBlockAction(ui.KedaChangeActionID, handler.HandleChangeAction)
	registeree.HandleInteractionBlockAction(ui.KedaCancelActionID, handler.HandleCancelAction)
	registeree.HandleViewSubmission(ui.KedaChangeCallbackID, handler.HandleChangeSubmission)

	return nil
}
