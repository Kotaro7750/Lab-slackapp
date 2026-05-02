package handler

import (
	"time"

	"github.com/Kotaro7750/Lab-slackapp/internal/kedalaunch/keda_launcher_client"
	"github.com/Kotaro7750/Lab-slackapp/internal/kedalaunch/slack_responder"
)

// KedaLaunchHandler owns the dependencies used by the /launch Slack flow.
type KedaLaunchHandler struct {
	kedaLauncher   keda_launcher_client.KedaLauncher
	slackResponder slack_responder.SlackResponderIF
	now            func() time.Time
}

// NewKedaLaunchHandler wires the Slack-facing handlers to the launcher and response adapters.
func NewKedaLaunchHandler(kedaLauncher keda_launcher_client.KedaLauncherIF, responder slack_responder.SlackResponderIF) *KedaLaunchHandler {
	return &KedaLaunchHandler{
		kedaLauncher:   *keda_launcher_client.NewKedaLauncher(kedaLauncher),
		slackResponder: responder,
		now:            time.Now,
	}
}
