package ui

import (
	"testing"
	"time"

	domainclient "github.com/Kotaro7750/keda-launcher-scaler/pkg/client"
	"github.com/slack-go/slack"
)

func TestBuildAcceptedMessageEmbedsSharedMetadataIntoFollowUpButtons(t *testing.T) {
	metadata := LaunchRequestMetadata{
		RequestID:   "request-1",
		Namespace:   "default",
		Name:        "worker",
		Duration:    "10m",
		ResponseURL: fakeResponseURL,
	}
	req := domainclient.LaunchRequest{
		RequestID: "request-1",
		ScaledObject: domainclient.ScaledObject{
			Namespace: "default",
			Name:      "worker",
		},
		Duration: 10 * time.Minute,
	}
	accepted := domainclient.AcceptedRequest{
		RequestID: "request-1",
		ScaledObject: domainclient.ScaledObject{
			Namespace: "default",
			Name:      "worker",
		},
		EffectiveStart: time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC),
		EffectiveEnd:   time.Date(2026, 5, 1, 12, 10, 0, 0, time.UTC),
	}

	message := metadata.BuildAcceptedMessage(accepted, req, true)

	if !message.ReplaceOriginal {
		t.Fatal("ReplaceOriginal = false")
	}
	if message.ResponseType != slack.ResponseTypeEphemeral {
		t.Fatalf("ResponseType = %q", message.ResponseType)
	}
	if message.Blocks == nil || len(message.Blocks.BlockSet) != 2 {
		t.Fatalf("blocks = %+v", message.Blocks)
	}

	actionBlock, ok := message.Blocks.BlockSet[1].(*slack.ActionBlock)
	if !ok {
		t.Fatalf("action block type = %T", message.Blocks.BlockSet[1])
	}
	if len(actionBlock.Elements.ElementSet) != 2 {
		t.Fatalf("elements = %d", len(actionBlock.Elements.ElementSet))
	}

	for _, element := range actionBlock.Elements.ElementSet {
		button, ok := element.(*slack.ButtonBlockElement)
		if !ok {
			t.Fatalf("button type = %T", element)
		}
		if button.Value != metadata.Encode() {
			t.Fatalf("button value = %q", button.Value)
		}
	}
}
