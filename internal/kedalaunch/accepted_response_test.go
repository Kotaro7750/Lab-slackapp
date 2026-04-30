package kedalaunch

import (
	"testing"
	"time"

	domainclient "github.com/Kotaro7750/keda-launcher-scaler/pkg/client"
	"github.com/slack-go/slack"
)

func TestAcceptedLaunchMessageIncludesChangeButtonMetadata(t *testing.T) {
	accepted := domainclient.AcceptedRequest{
		RequestID: "request-1",
		ScaledObject: domainclient.ScaledObject{
			Namespace: "default",
			Name:      "worker",
		},
		EffectiveStart: time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC),
		EffectiveEnd:   time.Date(2026, 4, 25, 12, 10, 0, 0, time.UTC),
	}
	req := domainclient.LaunchRequest{
		RequestID: "request-1",
		ScaledObject: domainclient.ScaledObject{
			Namespace: "default",
			Name:      "worker",
		},
		Duration: 10 * time.Minute,
	}

	msg := acceptedLaunchMessage(accepted, req, "https://hooks.slack.test/response", true)
	if msg.ResponseType != slack.ResponseTypeEphemeral {
		t.Fatalf("responseType = %q", msg.ResponseType)
	}
	if !msg.ReplaceOriginal {
		t.Fatal("ReplaceOriginal = false")
	}
	if msg.Blocks == nil {
		t.Fatalf("blocks = %+v", msg.Blocks)
	}

	button := findButton(t, msg.Blocks, kedaChangeActionID)
	if button.ActionID != kedaChangeActionID {
		t.Fatalf("actionID = %q", button.ActionID)
	}

	metadata, err := decodeKedaRequestMetadata(button.Value)
	if err != nil {
		t.Fatalf("decodeKedaRequestMetadata() error = %v", err)
	}
	if metadata.RequestID != "request-1" || metadata.Namespace != "default" || metadata.Name != "worker" {
		t.Fatalf("metadata = %+v", metadata)
	}
	if metadata.Duration != "10m0s" {
		t.Fatalf("duration metadata = %q", metadata.Duration)
	}
}

func findButton(t *testing.T, blocks *slack.Blocks, actionID string) *slack.ButtonBlockElement {
	t.Helper()

	for _, block := range blocks.BlockSet {
		action, ok := block.(*slack.ActionBlock)
		if !ok {
			continue
		}
		for _, element := range action.Elements.ElementSet {
			button, ok := element.(*slack.ButtonBlockElement)
			if ok && button.ActionID == actionID {
				return button
			}
		}
	}
	t.Fatalf("button %q was not found", actionID)
	return nil
}
