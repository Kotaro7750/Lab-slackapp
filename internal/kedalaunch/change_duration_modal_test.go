package kedalaunch

import (
	"testing"
	"time"

	"github.com/slack-go/slack"
)

func TestParseChangeSubmissionKeepsRequestIDAndScaledObject(t *testing.T) {
	metadata, err := encodeKedaRequestMetadata(kedaRequestMetadata{
		RequestID:   "request-1",
		Namespace:   "default",
		Name:        "worker",
		Duration:    "10m",
		ResponseURL: "https://hooks.slack.test/response",
	})
	if err != nil {
		t.Fatalf("encodeKedaRequestMetadata() error = %v", err)
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
