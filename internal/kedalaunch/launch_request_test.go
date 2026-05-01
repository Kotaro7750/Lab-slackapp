package kedalaunch

import (
	"context"
	"testing"
	"time"

	domainclient "github.com/Kotaro7750/keda-launcher-scaler/pkg/client"
)

type launcherSpy struct {
	launchReq      domainclient.LaunchRequest
	launchDeadline time.Duration

	deleteReq      domainclient.DeleteRequest
	deleteDeadline time.Duration
}

func (s *launcherSpy) Launch(ctx context.Context, req domainclient.LaunchRequest) (domainclient.AcceptedRequest, error) {
	s.launchReq = req
	s.launchDeadline = deadlineWithin(ctx, time.Now())
	return domainclient.AcceptedRequest{}, nil
}

func (s *launcherSpy) DeleteRequest(ctx context.Context, req domainclient.DeleteRequest) (domainclient.DeletedRequest, error) {
	s.deleteReq = req
	s.deleteDeadline = deadlineWithin(ctx, time.Now())
	return domainclient.DeletedRequest{}, nil
}

func TestLaunchRequestUsesGatewayTimeout(t *testing.T) {
	spy := &launcherSpy{}
	command := &kedaLaunchCommand{launcher: spy}
	req := domainclient.LaunchRequest{
		RequestID: "request-1",
		ScaledObject: domainclient.ScaledObject{
			Namespace: "default",
			Name:      "worker",
		},
		Duration: 10 * time.Minute,
	}

	if _, err := command.launchRequest(req); err != nil {
		t.Fatalf("launchRequest() error = %v", err)
	}
	if spy.launchReq != req {
		t.Fatalf("launchReq = %+v", spy.launchReq)
	}
	assertTimeoutNear(t, spy.launchDeadline)
}

func TestCancelRequestUsesGatewayTimeout(t *testing.T) {
	spy := &launcherSpy{}
	command := &kedaLaunchCommand{launcher: spy}
	req := domainclient.DeleteRequest{
		RequestID: "request-1",
		ScaledObject: domainclient.ScaledObject{
			Namespace: "default",
			Name:      "worker",
		},
	}

	if _, err := command.cancelRequest(req); err != nil {
		t.Fatalf("cancelRequest() error = %v", err)
	}
	if spy.deleteReq != req {
		t.Fatalf("deleteReq = %+v", spy.deleteReq)
	}
	assertTimeoutNear(t, spy.deleteDeadline)
}

func deadlineWithin(ctx context.Context, now time.Time) time.Duration {
	deadline, ok := ctx.Deadline()
	if !ok {
		return 0
	}
	return deadline.Sub(now)
}

func assertTimeoutNear(t *testing.T, got time.Duration) {
	t.Helper()

	min := kedaLaunchTimeout - time.Second
	max := kedaLaunchTimeout + time.Second
	if got < min || got > max {
		t.Fatalf("deadline = %s, want between %s and %s", got, min, max)
	}
}
