package kedalaunch

import (
	"testing"
	"time"

	"github.com/slack-go/slack"
)

func TestParseLaunchSubmissionBuildsRequest(t *testing.T) {
	now := time.Date(2026, 4, 25, 12, 0, 0, 123, time.UTC)
	metadata, err := encodeLaunchModalMetadata(launchModalMetadata{
		UserID:      "U123",
		ChannelID:   "C123",
		ResponseURL: "https://hooks.slack.test/response",
	})
	if err != nil {
		t.Fatalf("encodeLaunchModalMetadata() error = %v", err)
	}

	req, responseURL, fieldErrors := parseLaunchSubmission(slack.View{
		PrivateMetadata: metadata,
		State: &slack.ViewState{Values: map[string]map[string]slack.BlockAction{
			kedaNamespaceBlockID: {
				kedaNamespaceAction: {Value: " default "},
			},
			kedaNameBlockID: {
				kedaNameAction: {Value: " worker "},
			},
			kedaDurationBlockID: {
				kedaDurationAction: {Value: "10m"},
			},
		}},
	}, now)

	if len(fieldErrors) > 0 {
		t.Fatalf("fieldErrors = %v", fieldErrors)
	}
	if responseURL != "https://hooks.slack.test/response" {
		t.Fatalf("responseURL = %q", responseURL)
	}
	if req.RequestID != "slack:U123:C123:default/worker:1777118400000000123" {
		t.Fatalf("requestID = %q", req.RequestID)
	}
	if req.ScaledObject.Namespace != "default" || req.ScaledObject.Name != "worker" {
		t.Fatalf("scaledObject = %+v", req.ScaledObject)
	}
	if req.Duration != 10*time.Minute {
		t.Fatalf("duration = %s", req.Duration)
	}
}

func TestParseLaunchSubmissionValidation(t *testing.T) {
	metadata, err := encodeLaunchModalMetadata(launchModalMetadata{ResponseURL: "https://hooks.slack.test/response"})
	if err != nil {
		t.Fatalf("encodeLaunchModalMetadata() error = %v", err)
	}

	_, responseURL, fieldErrors := parseLaunchSubmission(slack.View{
		PrivateMetadata: metadata,
		State: &slack.ViewState{Values: map[string]map[string]slack.BlockAction{
			kedaNamespaceBlockID: {
				kedaNamespaceAction: {Value: " "},
			},
			kedaNameBlockID: {
				kedaNameAction: {Value: "worker"},
			},
			kedaDurationBlockID: {
				kedaDurationAction: {Value: "soon"},
			},
		}},
	}, time.Now())

	if responseURL != "https://hooks.slack.test/response" {
		t.Fatalf("responseURL = %q", responseURL)
	}
	if len(fieldErrors) != 2 {
		t.Fatalf("fieldErrors = %v", fieldErrors)
	}
	if fieldErrors[kedaNamespaceBlockID] == "" {
		t.Fatalf("missing namespace error: %v", fieldErrors)
	}
	if fieldErrors[kedaDurationBlockID] == "" {
		t.Fatalf("missing duration error: %v", fieldErrors)
	}
}
