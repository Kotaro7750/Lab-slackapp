package keda_launcher_client

import (
	"context"
	"testing"
	"time"

	domainclient "github.com/Kotaro7750/keda-launcher-scaler/pkg/client"
)

type timeoutClientSpy struct {
	launchCtx context.Context
	deleteCtx context.Context
	listCtx   context.Context
}

func (s *timeoutClientSpy) Launch(ctx context.Context, req domainclient.LaunchRequest) (domainclient.AcceptedRequest, error) {
	s.launchCtx = ctx
	return domainclient.AcceptedRequest{}, nil
}

func (s *timeoutClientSpy) DeleteRequest(ctx context.Context, req domainclient.DeleteRequest) (domainclient.DeletedRequest, error) {
	s.deleteCtx = ctx
	return domainclient.DeletedRequest{}, nil
}

func (s *timeoutClientSpy) ListScaledObjects(ctx context.Context) ([]domainclient.ScaledObject, error) {
	s.listCtx = ctx
	return nil, nil
}

func TestKedaLauncherAppliesTimeoutToLaunchCancelAndList(t *testing.T) {
	client := &timeoutClientSpy{}
	launcher := NewKedaLauncher(client)
	start := time.Now()

	_, _ = launcher.LaunchRequest(domainclient.LaunchRequest{})
	_, _ = launcher.CancelRequest(domainclient.DeleteRequest{})
	_, _ = launcher.ListScaledObjects()

	if client.launchCtx == nil {
		t.Fatal("launch context = nil")
	}
	if client.deleteCtx == nil {
		t.Fatal("delete context = nil")
	}
	if client.listCtx == nil {
		t.Fatal("list context = nil")
	}

	launchDeadline, ok := client.launchCtx.Deadline()
	if !ok {
		t.Fatal("launch context has no deadline")
	}
	deleteDeadline, ok := client.deleteCtx.Deadline()
	if !ok {
		t.Fatal("delete context has no deadline")
	}
	listDeadline, ok := client.listCtx.Deadline()
	if !ok {
		t.Fatal("list context has no deadline")
	}

	assertDeadlineAround(t, launchDeadline, start.Add(kedaLaunchTimeout))
	assertDeadlineAround(t, deleteDeadline, start.Add(kedaLaunchTimeout))
	assertDeadlineAround(t, listDeadline, start.Add(kedaLaunchTimeout))
}

func assertDeadlineAround(t *testing.T, got, want time.Time) {
	t.Helper()
	diff := got.Sub(want)
	if diff < 0 {
		diff = -diff
	}
	if diff > 500*time.Millisecond {
		t.Fatalf("deadline diff = %s, got=%s want=%s", diff, got, want)
	}
}
