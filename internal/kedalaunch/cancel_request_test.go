package kedalaunch

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	domainclient "github.com/Kotaro7750/keda-launcher-scaler/pkg/client"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/socketmode"
)

type cancelLauncherSpy struct {
	deleteCalls       int
	deleteReq         domainclient.DeleteRequest
	deleteResp        domainclient.DeletedRequest
	deleteErr         error
	ackedBeforeDelete bool
	ackCalled         *bool
}

func (s *cancelLauncherSpy) Launch(context.Context, domainclient.LaunchRequest) (domainclient.AcceptedRequest, error) {
	return domainclient.AcceptedRequest{}, nil
}

func (s *cancelLauncherSpy) DeleteRequest(ctx context.Context, req domainclient.DeleteRequest) (domainclient.DeletedRequest, error) {
	s.deleteCalls++
	s.deleteReq = req
	s.ackedBeforeDelete = s.ackCalled != nil && *s.ackCalled
	return s.deleteResp, s.deleteErr
}

type webhookSpy struct {
	urls []string
	msgs []*slack.WebhookMessage
	err  error
}

func (s *webhookSpy) post(_ context.Context, url string, msg *slack.WebhookMessage) error {
	s.urls = append(s.urls, url)
	s.msgs = append(s.msgs, msg)
	return s.err
}

func TestAcceptedRequestActionFromCancelActionUsesSharedMetadataContract(t *testing.T) {
	metadata, err := encodeAcceptedRequestMetadata(acceptedRequestMetadata{
		RequestID:   "request-1",
		Namespace:   "default",
		Name:        "worker",
		Duration:    "10m0s",
		ResponseURL: "https://hooks.slack.test/original",
	})
	if err != nil {
		t.Fatalf("encodeAcceptedRequestMetadata() error = %v", err)
	}

	action, err := acceptedRequestActionFromCancelAction(slack.InteractionCallback{
		ResponseURL: "https://hooks.slack.test/fallback",
		ActionCallback: slack.ActionCallbacks{
			BlockActions: []*slack.BlockAction{
				{ActionID: kedaCancelActionID, Value: metadata},
			},
		},
	})
	if err != nil {
		t.Fatalf("acceptedRequestActionFromCancelAction() error = %v", err)
	}

	if action.ResponseURL != "https://hooks.slack.test/fallback" {
		t.Fatalf("responseURL = %q", action.ResponseURL)
	}
	if action.Metadata.RequestID != "request-1" {
		t.Fatalf("requestID = %q", action.Metadata.RequestID)
	}
	if action.Metadata.Namespace != "default" || action.Metadata.Name != "worker" {
		t.Fatalf("scaledObject = %s/%s", action.Metadata.Namespace, action.Metadata.Name)
	}
	if action.Metadata.ResponseURL != "https://hooks.slack.test/original" {
		t.Fatalf("metadata responseURL = %q", action.Metadata.ResponseURL)
	}
}

func TestAcceptedRequestActionFromCancelActionKeepsFallbackResponseURLOnDecodeFailure(t *testing.T) {
	action, err := acceptedRequestActionFromCancelAction(slack.InteractionCallback{
		ResponseURL: "https://hooks.slack.test/fallback",
		ActionCallback: slack.ActionCallbacks{
			BlockActions: []*slack.BlockAction{
				{ActionID: kedaCancelActionID, Value: "not-json"},
			},
		},
	})
	if err == nil {
		t.Fatal("acceptedRequestActionFromCancelAction() error = nil")
	}

	if action.ResponseURL != "https://hooks.slack.test/fallback" {
		t.Fatalf("responseURL = %q", action.ResponseURL)
	}
	if action.Metadata.RequestID != "" || action.Metadata.Namespace != "" || action.Metadata.Name != "" {
		t.Fatalf("metadata = %+v", action.Metadata)
	}
}

func TestHandleCancelActionStopsWhenAckFails(t *testing.T) {
	client := socketmode.New(slack.New("test-token"))
	launcher := &cancelLauncherSpy{}
	webhooks := &webhookSpy{}
	command := &kedaLaunchCommand{
		launcher:    launcher,
		postWebhook: webhooks.post,
	}
	previousAck := cancelActionAck
	cancelActionAck = func(evt *socketmode.Event, client *socketmode.Client, payload any) bool {
		return false
	}
	defer func() {
		cancelActionAck = previousAck
	}()

	command.handleCancelAction(&socketmode.Event{
		Data: slack.InteractionCallback{
			ResponseURL: "https://hooks.slack.test/fallback",
			ActionCallback: slack.ActionCallbacks{
				BlockActions: []*slack.BlockAction{
					{ActionID: kedaCancelActionID, Value: "not-json"},
				},
			},
		},
		Request: &socketmode.Request{EnvelopeID: "envelope-1"},
	}, client)

	if launcher.deleteCalls != 0 {
		t.Fatalf("deleteCalls = %d, want 0", launcher.deleteCalls)
	}
	if got := len(webhooks.msgs); got != 0 {
		t.Fatalf("webhook posts = %d, want 0", got)
	}
}

func TestHandleCancelActionPostsCanceledReplacementOnSuccess(t *testing.T) {
	client := socketmode.New(slack.New("test-token"))
	ackCalled := false
	launcher := &cancelLauncherSpy{
		ackCalled: &ackCalled,
		deleteResp: domainclient.DeletedRequest{
			RequestID: "request-1",
			ScaledObject: domainclient.ScaledObject{
				Namespace: "default",
				Name:      "worker",
			},
			EffectiveStart: time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC),
			EffectiveEnd:   time.Date(2026, 5, 1, 12, 10, 0, 0, time.UTC),
		},
	}
	webhooks := &webhookSpy{}
	command := &kedaLaunchCommand{
		launcher:    launcher,
		postWebhook: webhooks.post,
	}
	previousAck := cancelActionAck
	cancelActionAck = func(evt *socketmode.Event, client *socketmode.Client, payload any) bool {
		ackCalled = true
		return true
	}
	defer func() {
		cancelActionAck = previousAck
	}()

	metadata, err := encodeAcceptedRequestMetadata(acceptedRequestMetadata{
		RequestID:   "request-1",
		Namespace:   "default",
		Name:        "worker",
		Duration:    "10m0s",
		ResponseURL: "https://hooks.slack.test/original",
	})
	if err != nil {
		t.Fatalf("encodeAcceptedRequestMetadata() error = %v", err)
	}

	command.handleCancelAction(&socketmode.Event{
		Data: slack.InteractionCallback{
			ResponseURL: "https://hooks.slack.test/fallback",
			ActionCallback: slack.ActionCallbacks{
				BlockActions: []*slack.BlockAction{
					{ActionID: kedaCancelActionID, Value: metadata},
				},
			},
		},
		Request: &socketmode.Request{EnvelopeID: "envelope-1"},
	}, client)

	if got, want := launcher.deleteCalls, 1; got != want {
		t.Fatalf("deleteCalls = %d, want %d", got, want)
	}
	if !launcher.ackedBeforeDelete {
		t.Fatal("delete ran before Slack ack was queued")
	}
	if got := launcher.deleteReq.RequestID; got != "request-1" {
		t.Fatalf("requestID = %q", got)
	}
	if got := launcher.deleteReq.ScaledObject; got.Namespace != "default" || got.Name != "worker" {
		t.Fatalf("scaledObject = %+v", got)
	}
	if !ackCalled {
		t.Fatal("ack was not invoked")
	}
	if got, want := len(webhooks.msgs), 1; got != want {
		t.Fatalf("webhook posts = %d, want %d", got, want)
	}
	if got := webhooks.urls[0]; got != "https://hooks.slack.test/original" {
		t.Fatalf("responseURL = %q", got)
	}
	if !webhooks.msgs[0].ReplaceOriginal {
		t.Fatal("ReplaceOriginal = false")
	}
	if got := webhooks.msgs[0].Text; got != "Launch request canceled." {
		t.Fatalf("message text = %q", got)
	}
	section := findSectionText(t, webhooks.msgs[0].Blocks)
	if !strings.Contains(section.Text, "*Launch request canceled*") {
		t.Fatalf("section text = %q", section.Text)
	}
}

func TestHandleCancelActionRejectsInvalidMetadataWithoutDelete(t *testing.T) {
	client := socketmode.New(slack.New("test-token"))
	ackCalled := false
	launcher := &cancelLauncherSpy{ackCalled: &ackCalled}
	webhooks := &webhookSpy{}
	command := &kedaLaunchCommand{
		launcher:    launcher,
		postWebhook: webhooks.post,
	}
	previousAck := cancelActionAck
	cancelActionAck = func(evt *socketmode.Event, client *socketmode.Client, payload any) bool {
		ackCalled = true
		return true
	}
	defer func() {
		cancelActionAck = previousAck
	}()

	command.handleCancelAction(&socketmode.Event{
		Data: slack.InteractionCallback{
			ResponseURL: "https://hooks.slack.test/fallback",
			ActionCallback: slack.ActionCallbacks{
				BlockActions: []*slack.BlockAction{
					{ActionID: kedaCancelActionID, Value: "not-json"},
				},
			},
		},
		Request: &socketmode.Request{EnvelopeID: "envelope-1"},
	}, client)

	if launcher.deleteCalls != 0 {
		t.Fatalf("deleteCalls = %d, want 0", launcher.deleteCalls)
	}
	if !ackCalled {
		t.Fatal("ack was not invoked")
	}
	if got := len(webhooks.msgs); got != 1 {
		t.Fatalf("webhook posts = %d, want 1", got)
	}
	if got := webhooks.urls[0]; got != "https://hooks.slack.test/fallback" {
		t.Fatalf("responseURL = %q", got)
	}
	if got := webhooks.msgs[0].Text; got != "Failed to cancel launch request." {
		t.Fatalf("message text = %q", got)
	}
	if webhooks.msgs[0].ReplaceOriginal {
		t.Fatal("ReplaceOriginal = true, want false")
	}
}

func TestHandleCancelActionPostsEphemeralErrorWhenDeleteFails(t *testing.T) {
	client := socketmode.New(slack.New("test-token"))
	ackCalled := false
	launcher := &cancelLauncherSpy{
		ackCalled: &ackCalled,
		deleteErr: errors.New("receiver rejected delete"),
	}
	webhooks := &webhookSpy{}
	command := &kedaLaunchCommand{
		launcher:    launcher,
		postWebhook: webhooks.post,
	}
	previousAck := cancelActionAck
	cancelActionAck = func(evt *socketmode.Event, client *socketmode.Client, payload any) bool {
		ackCalled = true
		return true
	}
	defer func() {
		cancelActionAck = previousAck
	}()

	metadata, err := encodeAcceptedRequestMetadata(acceptedRequestMetadata{
		RequestID:   "request-1",
		Namespace:   "default",
		Name:        "worker",
		ResponseURL: "https://hooks.slack.test/original",
	})
	if err != nil {
		t.Fatalf("encodeAcceptedRequestMetadata() error = %v", err)
	}

	command.handleCancelAction(&socketmode.Event{
		Data: slack.InteractionCallback{
			ResponseURL: "https://hooks.slack.test/fallback",
			ActionCallback: slack.ActionCallbacks{
				BlockActions: []*slack.BlockAction{
					{ActionID: kedaCancelActionID, Value: metadata},
				},
			},
		},
		Request: &socketmode.Request{EnvelopeID: "envelope-1"},
	}, client)

	if got, want := launcher.deleteCalls, 1; got != want {
		t.Fatalf("deleteCalls = %d, want %d", got, want)
	}
	if !ackCalled {
		t.Fatal("ack was not invoked")
	}
	if got, want := len(webhooks.msgs), 1; got != want {
		t.Fatalf("webhook posts = %d, want %d", got, want)
	}
	if got := webhooks.urls[0]; got != "https://hooks.slack.test/original" {
		t.Fatalf("responseURL = %q", got)
	}
	if webhooks.msgs[0].ReplaceOriginal {
		t.Fatal("ReplaceOriginal = true, want false")
	}
	if got := webhooks.msgs[0].Text; got != "Launch request was not canceled and might still be active." {
		t.Fatalf("message text = %q", got)
	}
}

func TestHandleCancelActionLogsWebhookPostFailureOnSuccess(t *testing.T) {
	client := socketmode.New(slack.New("test-token"))
	ackCalled := false
	launcher := &cancelLauncherSpy{
		ackCalled: &ackCalled,
		deleteResp: domainclient.DeletedRequest{
			RequestID: "request-1",
			ScaledObject: domainclient.ScaledObject{
				Namespace: "default",
				Name:      "worker",
			},
			EffectiveStart: time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC),
			EffectiveEnd:   time.Date(2026, 5, 1, 12, 10, 0, 0, time.UTC),
		},
	}
	webhooks := &webhookSpy{err: errors.New("slack unavailable")}
	command := &kedaLaunchCommand{
		launcher:    launcher,
		postWebhook: webhooks.post,
	}
	previousAck := cancelActionAck
	cancelActionAck = func(evt *socketmode.Event, client *socketmode.Client, payload any) bool {
		ackCalled = true
		return true
	}
	defer func() {
		cancelActionAck = previousAck
	}()

	metadata, err := encodeAcceptedRequestMetadata(acceptedRequestMetadata{
		RequestID:   "request-1",
		Namespace:   "default",
		Name:        "worker",
		ResponseURL: "https://hooks.slack.test/original",
	})
	if err != nil {
		t.Fatalf("encodeAcceptedRequestMetadata() error = %v", err)
	}

	var logBuf strings.Builder
	logger := slog.New(slog.NewTextHandler(&logBuf, nil))
	previousDefault := slog.Default()
	slog.SetDefault(logger)
	defer slog.SetDefault(previousDefault)

	command.handleCancelAction(&socketmode.Event{
		Data: slack.InteractionCallback{
			ResponseURL: "https://hooks.slack.test/fallback",
			ActionCallback: slack.ActionCallbacks{
				BlockActions: []*slack.BlockAction{
					{ActionID: kedaCancelActionID, Value: metadata},
				},
			},
		},
		Request: &socketmode.Request{EnvelopeID: "envelope-1"},
	}, client)

	if got, want := launcher.deleteCalls, 1; got != want {
		t.Fatalf("deleteCalls = %d, want %d", got, want)
	}
	if !launcher.ackedBeforeDelete {
		t.Fatal("delete ran before Slack ack was queued")
	}
	if got, want := len(webhooks.msgs), 1; got != want {
		t.Fatalf("webhook posts = %d, want %d", got, want)
	}
	if !strings.Contains(logBuf.String(), "failed to post cancel success response") {
		t.Fatalf("log output = %q", logBuf.String())
	}
}
