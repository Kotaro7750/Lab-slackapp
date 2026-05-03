package keda_launcher_client

import (
	"context"

	domainclient "github.com/Kotaro7750/keda-launcher-scaler/pkg/client"
)

// KedaLauncherIF is the subset of the KEDA launcher client used by this feature.
type KedaLauncherIF interface {
	Launch(context.Context, domainclient.LaunchRequest) (domainclient.AcceptedRequest, error)
	DeleteRequest(context.Context, domainclient.DeleteRequest) (domainclient.DeletedRequest, error)
	ListScaledObjects(context.Context) ([]domainclient.ScaledObject, error)
}
