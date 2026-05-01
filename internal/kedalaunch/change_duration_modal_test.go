package kedalaunch

import (
	"testing"
	"time"

	"github.com/slack-go/slack"
)

func TestAcceptedRequestActionFromChangeActionUsesSharedMetadataContract(t *testing.T) {
	metadata, err := encodeAcceptedRequestMetadata(acceptedRequestMetadata{
		RequestID:   "request-1",
		Namespace:   "default",
		Name:        "worker",
		Duration:    "10m",
		ResponseURL: "https://hooks.slack.test/original",
	})
	if err != nil {
		t.Fatalf("encodeAcceptedRequestMetadata() error = %v", err)
	}

	action, err := acceptedRequestActionFromChangeAction(slack.InteractionCallback{
		ResponseURL: "https://hooks.slack.test/fallback",
		ActionCallback: slack.ActionCallbacks{
			BlockActions: []*slack.BlockAction{
				{ActionID: kedaChangeActionID, Value: metadata},
			},
		},
	})
	if err != nil {
		t.Fatalf("acceptedRequestActionFromChangeAction() error = %v", err)
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
	if action.Metadata.Duration != "10m" {
		t.Fatalf("duration = %q", action.Metadata.Duration)
	}
	if action.Metadata.ResponseURL != "https://hooks.slack.test/original" {
		t.Fatalf("metadata responseURL = %q", action.Metadata.ResponseURL)
	}
}

func TestParseChangeSubmissionKeepsRequestIDAndScaledObject(t *testing.T) {
	metadata, err := encodeAcceptedRequestMetadata(acceptedRequestMetadata{
		RequestID:   "request-1",
		Namespace:   "default",
		Name:        "worker",
		Duration:    "10m",
		ResponseURL: "https://hooks.slack.test/response",
	})
	if err != nil {
		t.Fatalf("encodeAcceptedRequestMetadata() error = %v", err)
	}

	req, responseURL, fieldErrors := parseChangeSubmission(slack.View{
		PrivateMetadata: metadata,
		State: &slack.ViewState{Values: map[string]map[string]slack.BlockAction{
			kedaDurationBlockID: {
				kedaDurationAction: {Value: "30m"},
			},
		}},
	})

	if len(fieldErrors) > 0 {
		t.Fatalf("fieldErrors = %v", fieldErrors)
	}
	if responseURL != "https://hooks.slack.test/response" {
		t.Fatalf("responseURL = %q", responseURL)
	}
	if req.RequestID != "request-1" {
		t.Fatalf("requestID = %q", req.RequestID)
	}
	if req.ScaledObject.Namespace != "default" || req.ScaledObject.Name != "worker" {
		t.Fatalf("scaledObject = %+v", req.ScaledObject)
	}
	if req.Duration != 30*time.Minute {
		t.Fatalf("duration = %s", req.Duration)
	}
}

func TestAcceptedRequestActionFromChangeActionKeepsNotificationTargetOnDecodeFailure(t *testing.T) {
	action, err := acceptedRequestActionFromChangeAction(slack.InteractionCallback{
		ResponseURL: "https://hooks.slack.test/fallback",
		ActionCallback: slack.ActionCallbacks{
			BlockActions: []*slack.BlockAction{
				{Value: "not-json"},
			},
		},
	})

	if err == nil {
		t.Fatal("acceptedRequestActionFromChangeAction() error = nil")
	}
	if action.ResponseURL != "https://hooks.slack.test/fallback" {
		t.Fatalf("responseURL = %q", action.ResponseURL)
	}
	if action.Metadata.RequestID != "" || action.Metadata.Namespace != "" || action.Metadata.Name != "" {
		t.Fatalf("metadata = %+v", action.Metadata)
	}
}
