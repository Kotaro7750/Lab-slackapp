package ui

import (
	"fmt"
	"testing"
	"time"

	"github.com/slack-go/slack"
)

const fakeUserID = "U123"
const fakeChannelID = "C123"
const fakeResponseURL = "https://hooks.slack.test/response"

var fakeCommandInvocationMetadata = CommandInvocationMetadata{
	UserID:      fakeUserID,
	ChannelID:   fakeChannelID,
	ResponseURL: fakeResponseURL,
}

func TestParseLaunchModal_parse_correctly(t *testing.T) {
	now := time.Date(2026, 4, 25, 12, 0, 0, 123, time.UTC)
	req, fieldErrors := fakeCommandInvocationMetadata.ParseLaunchModal(slack.View{
		PrivateMetadata: fakeCommandInvocationMetadata.Encode(),
		State: &slack.ViewState{Values: map[string]map[string]slack.BlockAction{
			kedaNamespaceBlockID: {
				kedaNamespaceAction: {Value: " default "},
			},
			kedaNameBlockID: {
				kedaNameAction: {Value: " worker "},
			},
			KedaDurationBlockID: {
				kedaDurationAction: {Value: "10m"},
			},
		}},
	}, now)

	if len(fieldErrors) > 0 {
		t.Fatalf("fieldErrors = %v", fieldErrors)
	}
	if req.RequestID != fmt.Sprintf("slack:%s:%s:default/worker:1777118400000000123", fakeUserID, fakeChannelID) {
		t.Fatalf("requestID = %q", req.RequestID)
	}
	if req.ScaledObject.Namespace != "default" || req.ScaledObject.Name != "worker" {
		t.Fatalf("scaledObject = %+v", req.ScaledObject)
	}
	if req.Duration != 10*time.Minute {
		t.Fatalf("duration = %s", req.Duration)
	}
}

func TestParseLaunchModal_validate_correctly(t *testing.T) {
	_, fieldErrors := fakeCommandInvocationMetadata.ParseLaunchModal(
		slack.View{
			State: &slack.ViewState{Values: map[string]map[string]slack.BlockAction{
				kedaNamespaceBlockID: {
					kedaNamespaceAction: {Value: " "},
				},
				kedaNameBlockID: {
					kedaNameAction: {Value: "worker"},
				},
				KedaDurationBlockID: {
					kedaDurationAction: {Value: "soon"},
				},
			}},
		}, time.Now())

	if len(fieldErrors) != 2 {
		t.Fatalf("fieldErrors = %v", fieldErrors)
	}
	if fieldErrors[kedaNamespaceBlockID] == "" {
		t.Fatalf("missing namespace error: %v", fieldErrors)
	}
	if fieldErrors[KedaDurationBlockID] == "" {
		t.Fatalf("missing duration error: %v", fieldErrors)
	}
}
