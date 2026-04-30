package kedalaunch

import (
	"context"
	"time"

	domainclient "github.com/Kotaro7750/keda-launcher-scaler/pkg/client"
	"github.com/slack-go/slack"
)

// kedaLauncher is the subset of the KEDA launcher client used by this feature.
type kedaLauncher interface {
	Launch(context.Context, domainclient.LaunchRequest) (domainclient.AcceptedRequest, error)
}

// kedaLaunchCommand owns the dependencies used by the /launch Slack flow.
type kedaLaunchCommand struct {
	api         *slack.Client
	launcher    kedaLauncher
	postWebhook func(context.Context, string, *slack.WebhookMessage) error
	now         func() time.Time
}
