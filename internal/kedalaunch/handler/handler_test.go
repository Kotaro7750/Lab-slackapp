package handler

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Kotaro7750/Lab-slackapp/internal/kedalaunch/slack_responder"
	"github.com/Kotaro7750/Lab-slackapp/internal/kedalaunch/ui"
	domainclient "github.com/Kotaro7750/keda-launcher-scaler/pkg/client"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/socketmode"
)

type fakeKedaLauncher struct {
	launchReq   domainclient.LaunchRequest
	launchResp  domainclient.AcceptedRequest
	launchErr   error
	launchCalls int

	listResp  []domainclient.ScaledObject
	listErr   error
	listCalls int

	cancelReq   domainclient.DeleteRequest
	cancelResp  domainclient.DeletedRequest
	cancelErr   error
	cancelCalls int
}

func (f *fakeKedaLauncher) Launch(ctx context.Context, req domainclient.LaunchRequest) (domainclient.AcceptedRequest, error) {
	f.launchCalls++
	f.launchReq = req
	return f.launchResp, f.launchErr
}

func (f *fakeKedaLauncher) ListScaledObjects(ctx context.Context) ([]domainclient.ScaledObject, error) {
	f.listCalls++
	return f.listResp, f.listErr
}

func (f *fakeKedaLauncher) DeleteRequest(ctx context.Context, req domainclient.DeleteRequest) (domainclient.DeletedRequest, error) {
	f.cancelCalls++
	f.cancelReq = req
	return f.cancelResp, f.cancelErr
}

type fakeSlackResponder struct {
	ackSuccessCalls int
	ackViewCalls    int
	ackErrorCalls   int
	lastViewResp    *slack.ViewSubmissionResponse
	lastAckError    error

	openViewTriggerID string
	openView          slack.ModalViewRequest
	openViewCalls     int
	openViewResp      *slack.ViewResponse
	openViewErr       error

	errorPosts []errorPost
	webhooks   []webhookPost
}

type errorPost struct {
	responseURL     string
	text            string
	replaceOriginal bool
}

type webhookPost struct {
	responseURL string
	message     *slack.WebhookMessage
}

func (f *fakeSlackResponder) AckWithSuccess(evt *socketmode.Event, client *socketmode.Client) error {
	f.ackSuccessCalls++
	return nil
}

func (f *fakeSlackResponder) AckWithViewResponse(evt *socketmode.Event, client *socketmode.Client, viewError *slack.ViewSubmissionResponse) error {
	f.ackViewCalls++
	f.lastViewResp = viewError
	return nil
}

func (f *fakeSlackResponder) AckWithUnrecoverableError(evt *socketmode.Event, client *socketmode.Client, err error) error {
	f.ackErrorCalls++
	f.lastAckError = err
	return nil
}

func (f *fakeSlackResponder) PostEphemeralError(ctx context.Context, responseURL, text string, replaceOriginal bool) {
	f.errorPosts = append(f.errorPosts, errorPost{
		responseURL:     responseURL,
		text:            text,
		replaceOriginal: replaceOriginal,
	})
}

func (f *fakeSlackResponder) PostWebhook(ctx context.Context, responseURL string, message *slack.WebhookMessage) error {
	f.webhooks = append(f.webhooks, webhookPost{
		responseURL: responseURL,
		message:     message,
	})
	return nil
}

func (f *fakeSlackResponder) OpenViewContext(ctx context.Context, triggerID string, view slack.ModalViewRequest) (*slack.ViewResponse, error) {
	f.openViewCalls++
	f.openViewTriggerID = triggerID
	f.openView = view
	return f.openViewResp, f.openViewErr
}

func newTestHandler(launcher *fakeKedaLauncher, responder slack_responder.SlackResponderIF) *KedaLaunchHandler {
	handler := NewKedaLaunchHandler(launcher, responder)
	handler.now = func() time.Time {
		return time.Date(2026, 5, 3, 12, 0, 0, 123, time.UTC)
	}
	return handler
}

func mustEncodeCommandMetadata(t *testing.T) string {
	t.Helper()
	return (&ui.CommandInvocationMetadata{
		UserID:      "U123",
		ChannelID:   "C123",
		ResponseURL: "https://hooks.slack.test/response",
	}).Encode()
}

func mustEncodeLaunchMetadata(t *testing.T) string {
	t.Helper()
	return (&ui.LaunchRequestMetadata{
		RequestID:   "request-1",
		Namespace:   "default",
		Name:        "worker",
		Duration:    "10m",
		ResponseURL: "https://hooks.slack.test/response",
	}).Encode()
}

func findActionButtons(t *testing.T, message *slack.WebhookMessage) []*slack.ButtonBlockElement {
	t.Helper()
	if message.Blocks == nil || len(message.Blocks.BlockSet) < 2 {
		t.Fatalf("blocks = %+v", message.Blocks)
	}
	actionBlock, ok := message.Blocks.BlockSet[1].(*slack.ActionBlock)
	if !ok {
		t.Fatalf("action block type = %T", message.Blocks.BlockSet[1])
	}

	buttons := make([]*slack.ButtonBlockElement, 0, len(actionBlock.Elements.ElementSet))
	for _, element := range actionBlock.Elements.ElementSet {
		button, ok := element.(*slack.ButtonBlockElement)
		if !ok {
			t.Fatalf("button type = %T", element)
		}
		buttons = append(buttons, button)
	}
	return buttons
}

func findSectionText(t *testing.T, message *slack.WebhookMessage) string {
	t.Helper()
	if message.Blocks == nil || len(message.Blocks.BlockSet) == 0 {
		t.Fatalf("blocks = %+v", message.Blocks)
	}
	section, ok := message.Blocks.BlockSet[0].(*slack.SectionBlock)
	if !ok {
		t.Fatalf("section type = %T", message.Blocks.BlockSet[0])
	}
	if section.Text == nil {
		t.Fatal("section text = nil")
	}
	return section.Text.Text
}

func TestHandleSlashCommandAcknowledgesAndOpensLaunchModal(t *testing.T) {
	launcher := &fakeKedaLauncher{
		listResp: []domainclient.ScaledObject{
			{Namespace: "default", Name: "worker"},
		},
	}
	responder := &fakeSlackResponder{}
	handler := newTestHandler(launcher, responder)

	handler.HandleSlashCommand(&socketmode.Event{
		Data: slack.SlashCommand{
			UserID:      "U123",
			ChannelID:   "C123",
			TriggerID:   "trigger-1",
			ResponseURL: "https://hooks.slack.test/response",
		},
	}, nil)

	if responder.ackSuccessCalls != 1 {
		t.Fatalf("ackSuccessCalls = %d", responder.ackSuccessCalls)
	}
	if responder.openViewCalls != 1 {
		t.Fatalf("openViewCalls = %d", responder.openViewCalls)
	}
	if launcher.listCalls != 1 {
		t.Fatalf("listCalls = %d", launcher.listCalls)
	}
	if responder.openViewTriggerID != "trigger-1" {
		t.Fatalf("triggerID = %q", responder.openViewTriggerID)
	}
	metadata, err := ui.DecodeCommandInvocationMetadata(responder.openView.PrivateMetadata)
	if err != nil {
		t.Fatalf("DecodeCommandInvocationMetadata() error = %v", err)
	}
	if metadata.UserID != "U123" || metadata.ChannelID != "C123" || metadata.ResponseURL != "https://hooks.slack.test/response" {
		t.Fatalf("metadata = %+v", metadata)
	}
	if len(responder.errorPosts) != 0 {
		t.Fatalf("errorPosts = %d", len(responder.errorPosts))
	}
}

func TestHandleSlashCommandPostsEphemeralErrorWhenListFails(t *testing.T) {
	launcher := &fakeKedaLauncher{listErr: errors.New("boom")}
	responder := &fakeSlackResponder{}
	handler := newTestHandler(launcher, responder)

	handler.HandleSlashCommand(&socketmode.Event{
		Data: slack.SlashCommand{
			UserID:      "U123",
			ChannelID:   "C123",
			TriggerID:   "trigger-1",
			ResponseURL: "https://hooks.slack.test/response",
		},
	}, nil)

	if responder.ackSuccessCalls != 1 {
		t.Fatalf("ackSuccessCalls = %d", responder.ackSuccessCalls)
	}
	if responder.openViewCalls != 0 {
		t.Fatalf("openViewCalls = %d", responder.openViewCalls)
	}
	if len(responder.errorPosts) != 1 {
		t.Fatalf("errorPosts = %d", len(responder.errorPosts))
	}
	if responder.errorPosts[0].text != "Failed to load launch targets." {
		t.Fatalf("error text = %q", responder.errorPosts[0].text)
	}
}

func TestHandleSlashCommandPostsEphemeralErrorWhenNoTargetsExist(t *testing.T) {
	launcher := &fakeKedaLauncher{listResp: []domainclient.ScaledObject{}}
	responder := &fakeSlackResponder{}
	handler := newTestHandler(launcher, responder)

	handler.HandleSlashCommand(&socketmode.Event{
		Data: slack.SlashCommand{
			UserID:      "U123",
			ChannelID:   "C123",
			TriggerID:   "trigger-1",
			ResponseURL: "https://hooks.slack.test/response",
		},
	}, nil)

	if responder.ackSuccessCalls != 1 {
		t.Fatalf("ackSuccessCalls = %d", responder.ackSuccessCalls)
	}
	if responder.openViewCalls != 0 {
		t.Fatalf("openViewCalls = %d", responder.openViewCalls)
	}
	if len(responder.errorPosts) != 1 {
		t.Fatalf("errorPosts = %d", len(responder.errorPosts))
	}
	if responder.errorPosts[0].text != "No launch targets are currently available." {
		t.Fatalf("error text = %q", responder.errorPosts[0].text)
	}
}

func TestHandleSlashCommandPostsEphemeralErrorWhenOpenViewFails(t *testing.T) {
	launcher := &fakeKedaLauncher{
		listResp: []domainclient.ScaledObject{
			{Namespace: "default", Name: "worker"},
		},
	}
	responder := &fakeSlackResponder{openViewErr: errors.New("boom")}
	handler := newTestHandler(launcher, responder)

	handler.HandleSlashCommand(&socketmode.Event{
		Data: slack.SlashCommand{
			UserID:      "U123",
			ChannelID:   "C123",
			TriggerID:   "trigger-1",
			ResponseURL: "https://hooks.slack.test/response",
		},
	}, nil)

	if len(responder.errorPosts) != 1 {
		t.Fatalf("errorPosts = %d", len(responder.errorPosts))
	}
	if responder.errorPosts[0].text != "Failed to open launch form." {
		t.Fatalf("error text = %q", responder.errorPosts[0].text)
	}
}

func TestHandleLaunchSubmissionReturnsViewErrorsForInvalidInput(t *testing.T) {
	launcher := &fakeKedaLauncher{}
	responder := &fakeSlackResponder{}
	handler := newTestHandler(launcher, responder)

	handler.HandleLaunchSubmission(&socketmode.Event{
		Data: slack.InteractionCallback{
			View: slack.View{
				PrivateMetadata: mustEncodeCommandMetadata(t),
				State: &slack.ViewState{Values: map[string]map[string]slack.BlockAction{
					"keda_target": {
						"target": {SelectedOption: slack.OptionBlockObject{Value: ""}},
					},
					"keda_duration": {
						"duration": {Value: "soon"},
					},
				}},
			},
		},
	}, nil)

	if responder.ackViewCalls != 1 {
		t.Fatalf("ackViewCalls = %d", responder.ackViewCalls)
	}
	if launcher.launchCalls != 0 {
		t.Fatalf("launchCalls = %d", launcher.launchCalls)
	}
	if responder.lastViewResp == nil || responder.lastViewResp.ResponseAction != slack.RAErrors {
		t.Fatalf("viewResponse = %+v", responder.lastViewResp)
	}
}

func TestHandleLaunchSubmissionLaunchesAndPostsAcceptedMessage(t *testing.T) {
	launcher := &fakeKedaLauncher{
		launchResp: domainclient.AcceptedRequest{
			RequestID: "slack:U123:C123:default/worker:1777809600000000123",
			ScaledObject: domainclient.ScaledObject{
				Namespace: "default",
				Name:      "worker",
			},
			EffectiveStart: time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC),
			EffectiveEnd:   time.Date(2026, 5, 3, 12, 10, 0, 0, time.UTC),
		},
	}
	responder := &fakeSlackResponder{}
	handler := newTestHandler(launcher, responder)

	handler.HandleLaunchSubmission(&socketmode.Event{
		Data: slack.InteractionCallback{
			View: slack.View{
				PrivateMetadata: mustEncodeCommandMetadata(t),
				State: &slack.ViewState{Values: map[string]map[string]slack.BlockAction{
					"keda_target": {
						"target": {SelectedOption: slack.OptionBlockObject{Value: `{"namespace":" default ","name":" worker "}`}},
					},
					"keda_duration": {
						"duration": {Value: "10m"},
					},
				}},
			},
		},
	}, nil)

	if responder.ackSuccessCalls != 1 {
		t.Fatalf("ackSuccessCalls = %d", responder.ackSuccessCalls)
	}
	if launcher.launchCalls != 1 {
		t.Fatalf("launchCalls = %d", launcher.launchCalls)
	}
	if launcher.launchReq.ScaledObject.Namespace != "default" || launcher.launchReq.ScaledObject.Name != "worker" {
		t.Fatalf("launchReq = %+v", launcher.launchReq)
	}
	if launcher.launchReq.Duration != 10*time.Minute {
		t.Fatalf("duration = %s", launcher.launchReq.Duration)
	}
	if len(responder.webhooks) != 1 {
		t.Fatalf("webhooks = %d", len(responder.webhooks))
	}
	if responder.webhooks[0].responseURL != "https://hooks.slack.test/response" {
		t.Fatalf("responseURL = %q", responder.webhooks[0].responseURL)
	}
	if responder.webhooks[0].message.ReplaceOriginal {
		t.Fatal("ReplaceOriginal = true")
	}
	buttons := findActionButtons(t, responder.webhooks[0].message)
	if len(buttons) != 2 {
		t.Fatalf("buttons = %d", len(buttons))
	}
}

func TestHandleLaunchSubmissionPostsEphemeralErrorWhenLaunchFails(t *testing.T) {
	launcher := &fakeKedaLauncher{launchErr: errors.New("boom")}
	responder := &fakeSlackResponder{}
	handler := newTestHandler(launcher, responder)

	handler.HandleLaunchSubmission(&socketmode.Event{
		Data: slack.InteractionCallback{
			View: slack.View{
				PrivateMetadata: mustEncodeCommandMetadata(t),
				State: &slack.ViewState{Values: map[string]map[string]slack.BlockAction{
					"keda_target": {
						"target": {SelectedOption: slack.OptionBlockObject{Value: `{"namespace":"default","name":"worker"}`}},
					},
					"keda_duration": {
						"duration": {Value: "10m"},
					},
				}},
			},
		},
	}, nil)

	if len(responder.errorPosts) != 1 {
		t.Fatalf("errorPosts = %d", len(responder.errorPosts))
	}
	if responder.errorPosts[0].text != "Launch request failed." {
		t.Fatalf("error text = %q", responder.errorPosts[0].text)
	}
}

func TestHandleChangeActionAcknowledgesAndOpensModal(t *testing.T) {
	launcher := &fakeKedaLauncher{}
	responder := &fakeSlackResponder{}
	handler := newTestHandler(launcher, responder)

	handler.HandleChangeAction(&socketmode.Event{
		Data: slack.InteractionCallback{
			TriggerID:   "trigger-1",
			ResponseURL: "https://hooks.slack.test/fallback",
			ActionCallback: slack.ActionCallbacks{
				BlockActions: []*slack.BlockAction{
					{ActionID: ui.KedaChangeActionID, Value: mustEncodeLaunchMetadata(t)},
				},
			},
		},
	}, nil)

	if responder.ackSuccessCalls != 1 {
		t.Fatalf("ackSuccessCalls = %d", responder.ackSuccessCalls)
	}
	if responder.openViewCalls != 1 {
		t.Fatalf("openViewCalls = %d", responder.openViewCalls)
	}
	if responder.openView.CallbackID != ui.KedaChangeCallbackID {
		t.Fatalf("callbackID = %q", responder.openView.CallbackID)
	}
	metadata, err := ui.DecodeLaunchRequestMetadata(responder.openView.PrivateMetadata)
	if err != nil {
		t.Fatalf("DecodeLaunchRequestMetadata() error = %v", err)
	}
	if metadata.ResponseURL != "https://hooks.slack.test/fallback" {
		t.Fatalf("responseURL = %q", metadata.ResponseURL)
	}
}

func TestHandleChangeSubmissionReturnsViewErrorsForInvalidDuration(t *testing.T) {
	launcher := &fakeKedaLauncher{}
	responder := &fakeSlackResponder{}
	handler := newTestHandler(launcher, responder)

	handler.HandleChangeSubmission(&socketmode.Event{
		Data: slack.InteractionCallback{
			View: slack.View{
				PrivateMetadata: mustEncodeLaunchMetadata(t),
				State: &slack.ViewState{Values: map[string]map[string]slack.BlockAction{
					"keda_duration": {
						"duration": {Value: "soon"},
					},
				}},
			},
		},
	}, nil)

	if responder.ackViewCalls != 1 {
		t.Fatalf("ackViewCalls = %d", responder.ackViewCalls)
	}
	if launcher.launchCalls != 0 {
		t.Fatalf("launchCalls = %d", launcher.launchCalls)
	}
}

func TestHandleChangeSubmissionKeepsRequestTargetAndReplacesOriginalMessage(t *testing.T) {
	launcher := &fakeKedaLauncher{
		launchResp: domainclient.AcceptedRequest{
			RequestID: "request-1",
			ScaledObject: domainclient.ScaledObject{
				Namespace: "default",
				Name:      "worker",
			},
			EffectiveStart: time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC),
			EffectiveEnd:   time.Date(2026, 5, 3, 12, 30, 0, 0, time.UTC),
		},
	}
	responder := &fakeSlackResponder{}
	handler := newTestHandler(launcher, responder)

	handler.HandleChangeSubmission(&socketmode.Event{
		Data: slack.InteractionCallback{
			View: slack.View{
				PrivateMetadata: mustEncodeLaunchMetadata(t),
				State: &slack.ViewState{Values: map[string]map[string]slack.BlockAction{
					"keda_duration": {
						"duration": {Value: "30m"},
					},
				}},
			},
		},
	}, nil)

	if responder.ackSuccessCalls != 1 {
		t.Fatalf("ackSuccessCalls = %d", responder.ackSuccessCalls)
	}
	if launcher.launchReq.RequestID != "request-1" {
		t.Fatalf("requestID = %q", launcher.launchReq.RequestID)
	}
	if launcher.launchReq.ScaledObject.Namespace != "default" || launcher.launchReq.ScaledObject.Name != "worker" {
		t.Fatalf("scaledObject = %+v", launcher.launchReq.ScaledObject)
	}
	if launcher.launchReq.Duration != 30*time.Minute {
		t.Fatalf("duration = %s", launcher.launchReq.Duration)
	}
	if len(responder.webhooks) != 1 {
		t.Fatalf("webhooks = %d", len(responder.webhooks))
	}
	if responder.webhooks[0].responseURL != "https://hooks.slack.test/response" {
		t.Fatalf("responseURL = %q", responder.webhooks[0].responseURL)
	}
	if !responder.webhooks[0].message.ReplaceOriginal {
		t.Fatal("ReplaceOriginal = false")
	}
}

func TestHandleCancelActionRejectsInvalidMetadataWithoutCallingCancel(t *testing.T) {
	launcher := &fakeKedaLauncher{}
	responder := &fakeSlackResponder{}
	handler := newTestHandler(launcher, responder)

	handler.HandleCancelAction(&socketmode.Event{
		Data: slack.InteractionCallback{
			ResponseURL: "https://hooks.slack.test/fallback",
			ActionCallback: slack.ActionCallbacks{
				BlockActions: []*slack.BlockAction{
					{ActionID: ui.KedaCancelActionID, Value: "not-json"},
				},
			},
		},
	}, nil)

	if responder.ackErrorCalls != 1 {
		t.Fatalf("ackErrorCalls = %d", responder.ackErrorCalls)
	}
	if launcher.cancelCalls != 0 {
		t.Fatalf("cancelCalls = %d", launcher.cancelCalls)
	}
}

func TestHandleCancelActionPostsCanceledMessageOnSuccess(t *testing.T) {
	launcher := &fakeKedaLauncher{
		cancelResp: domainclient.DeletedRequest{
			RequestID: "request-1",
			ScaledObject: domainclient.ScaledObject{
				Namespace: "default",
				Name:      "worker",
			},
			EffectiveStart: time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC),
			EffectiveEnd:   time.Date(2026, 5, 3, 12, 10, 0, 0, time.UTC),
		},
	}
	responder := &fakeSlackResponder{}
	handler := newTestHandler(launcher, responder)

	handler.HandleCancelAction(&socketmode.Event{
		Data: slack.InteractionCallback{
			ResponseURL: "https://hooks.slack.test/fallback",
			ActionCallback: slack.ActionCallbacks{
				BlockActions: []*slack.BlockAction{
					{ActionID: ui.KedaCancelActionID, Value: mustEncodeLaunchMetadata(t)},
				},
			},
		},
	}, nil)

	if responder.ackSuccessCalls != 1 {
		t.Fatalf("ackSuccessCalls = %d", responder.ackSuccessCalls)
	}
	if launcher.cancelCalls != 1 {
		t.Fatalf("cancelCalls = %d", launcher.cancelCalls)
	}
	if launcher.cancelReq.RequestID != "request-1" {
		t.Fatalf("requestID = %q", launcher.cancelReq.RequestID)
	}
	if len(responder.webhooks) != 1 {
		t.Fatalf("webhooks = %d", len(responder.webhooks))
	}
	if responder.webhooks[0].responseURL != "https://hooks.slack.test/fallback" {
		t.Fatalf("responseURL = %q", responder.webhooks[0].responseURL)
	}
	if !responder.webhooks[0].message.ReplaceOriginal {
		t.Fatal("ReplaceOriginal = false")
	}
	if responder.webhooks[0].message.Text != "Launch request canceled." {
		t.Fatalf("message text = %q", responder.webhooks[0].message.Text)
	}
	if section := findSectionText(t, responder.webhooks[0].message); section == "" {
		t.Fatal("section text is empty")
	}
}

func TestHandleCancelActionPostsEphemeralErrorWhenCancelFails(t *testing.T) {
	launcher := &fakeKedaLauncher{cancelErr: errors.New("boom")}
	responder := &fakeSlackResponder{}
	handler := newTestHandler(launcher, responder)

	handler.HandleCancelAction(&socketmode.Event{
		Data: slack.InteractionCallback{
			ResponseURL: "https://hooks.slack.test/fallback",
			ActionCallback: slack.ActionCallbacks{
				BlockActions: []*slack.BlockAction{
					{ActionID: ui.KedaCancelActionID, Value: mustEncodeLaunchMetadata(t)},
				},
			},
		},
	}, nil)

	if len(responder.errorPosts) != 1 {
		t.Fatalf("errorPosts = %d", len(responder.errorPosts))
	}
	if responder.errorPosts[0].responseURL != "https://hooks.slack.test/fallback" {
		t.Fatalf("responseURL = %q", responder.errorPosts[0].responseURL)
	}
	if responder.errorPosts[0].text != "Launch request was not canceled and might still be active." {
		t.Fatalf("error text = %q", responder.errorPosts[0].text)
	}
}
