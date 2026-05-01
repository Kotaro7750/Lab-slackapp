package kedalaunch

import (
	"context"
	"time"

	domainclient "github.com/Kotaro7750/keda-launcher-scaler/pkg/client"
)

const kedaLaunchTimeout = 10 * time.Second

// launchRequest sends a KEDA launcher request with a bounded wait time.
func (c *kedaLaunchCommand) launchRequest(req domainclient.LaunchRequest) (domainclient.AcceptedRequest, error) {
	ctx, cancel := context.WithTimeout(context.Background(), kedaLaunchTimeout)
	defer cancel()

	return c.launcher.Launch(ctx, req)
}

// cancelRequest sends a KEDA delete request with the same bounded wait time as launch.
func (c *kedaLaunchCommand) cancelRequest(req domainclient.DeleteRequest) (domainclient.DeletedRequest, error) {
	ctx, cancel := context.WithTimeout(context.Background(), kedaLaunchTimeout)
	defer cancel()

	return c.launcher.DeleteRequest(ctx, req)
}
