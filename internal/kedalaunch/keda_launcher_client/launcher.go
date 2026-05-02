package keda_launcher_client

import (
	"context"
	"time"

	domainclient "github.com/Kotaro7750/keda-launcher-scaler/pkg/client"
)

const kedaLaunchTimeout = 10 * time.Second

// KedaLauncher adds the feature-local timeout policy around the upstream client.
type KedaLauncher struct {
	client KedaLauncherIF
}

// NewKedaLauncher wraps the upstream client with the timeout behavior expected by this feature.
func NewKedaLauncher(client KedaLauncherIF) *KedaLauncher {
	return &KedaLauncher{client: client}
}

// LaunchRequest sends a KEDA launcher request with a bounded wait time.
func (k *KedaLauncher) LaunchRequest(req domainclient.LaunchRequest) (domainclient.AcceptedRequest, error) {
	ctx, cancel := context.WithTimeout(context.Background(), kedaLaunchTimeout)
	defer cancel()

	return k.client.Launch(ctx, req)
}

// CancelRequest sends a KEDA delete request with the same bounded wait time as launch.
func (k *KedaLauncher) CancelRequest(req domainclient.DeleteRequest) (domainclient.DeletedRequest, error) {
	ctx, cancel := context.WithTimeout(context.Background(), kedaLaunchTimeout)
	defer cancel()

	return k.client.DeleteRequest(ctx, req)
}
