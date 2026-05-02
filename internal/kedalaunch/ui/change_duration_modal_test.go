package ui

import (
	"testing"
	"time"

	"github.com/slack-go/slack"
)

func TestParseChangeDurationModalKeepsRequestTarget(t *testing.T) {
	metadata := LaunchRequestMetadata{
		RequestID:   "request-1",
		Namespace:   "default",
		Name:        "worker",
		Duration:    "10m",
		ResponseURL: fakeResponseURL,
	}

	req, fieldErrors := metadata.ParseChangeDurationModal(slack.View{
		State: &slack.ViewState{Values: map[string]map[string]slack.BlockAction{
			KedaDurationBlockID: {
				kedaDurationAction: {Value: "30m"},
			},
		}},
	})

	if len(fieldErrors) > 0 {
		t.Fatalf("fieldErrors = %v", fieldErrors)
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

func TestParseChangeDurationModalReturnsFieldErrorForInvalidDuration(t *testing.T) {
	metadata := LaunchRequestMetadata{
		RequestID:   "request-1",
		Namespace:   "default",
		Name:        "worker",
		Duration:    "10m",
		ResponseURL: fakeResponseURL,
	}

	_, fieldErrors := metadata.ParseChangeDurationModal(slack.View{
		State: &slack.ViewState{Values: map[string]map[string]slack.BlockAction{
			KedaDurationBlockID: {
				kedaDurationAction: {Value: "soon"},
			},
		}},
	})

	if len(fieldErrors) != 1 {
		t.Fatalf("fieldErrors = %v", fieldErrors)
	}
	if fieldErrors[KedaDurationBlockID] == "" {
		t.Fatalf("missing duration error: %v", fieldErrors)
	}
}
