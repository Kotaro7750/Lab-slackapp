package kedalaunch

import (
	"testing"
	"time"

	domainclient "github.com/Kotaro7750/keda-launcher-scaler/pkg/client"
	"github.com/slack-go/slack"
)

func TestAcceptedLaunchMessageIncludesSharedMetadataForChangeAndCancelButtons(t *testing.T) {
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

	changeButton := findButton(t, msg.Blocks, kedaChangeActionID)
	if changeButton.ActionID != kedaChangeActionID {
		t.Fatalf("actionID = %q", changeButton.ActionID)
	}

	changeMetadata, err := decodeAcceptedRequestMetadata(changeButton.Value)
	if err != nil {
		t.Fatalf("decodeAcceptedRequestMetadata() error = %v", err)
	}
	if changeMetadata.RequestID != "request-1" || changeMetadata.Namespace != "default" || changeMetadata.Name != "worker" {
		t.Fatalf("metadata = %+v", changeMetadata)
	}
	if changeMetadata.Duration != "10m0s" {
		t.Fatalf("duration metadata = %q", changeMetadata.Duration)
	}
	if changeMetadata.ResponseURL != "https://hooks.slack.test/response" {
		t.Fatalf("responseURL metadata = %q", changeMetadata.ResponseURL)
	}

	cancelButton := findButton(t, msg.Blocks, kedaCancelActionID)
	if cancelButton.ActionID != kedaCancelActionID {
		t.Fatalf("cancel actionID = %q", cancelButton.ActionID)
	}
	cancelMetadata, err := decodeAcceptedRequestMetadata(cancelButton.Value)
	if err != nil {
		t.Fatalf("decodeAcceptedRequestMetadata(cancel) error = %v", err)
	}
	if cancelMetadata != changeMetadata {
		t.Fatalf("cancel metadata = %+v, want %+v", cancelMetadata, changeMetadata)
	}
}

func TestCanceledLaunchMessageReplacesAcceptedActions(t *testing.T) {
	deleted := domainclient.DeletedRequest{
		RequestID: "request-1",
		ScaledObject: domainclient.ScaledObject{
			Namespace: "default",
			Name:      "worker",
		},
		EffectiveStart: time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC),
		EffectiveEnd:   time.Date(2026, 4, 25, 12, 10, 0, 0, time.UTC),
	}

	msg := canceledLaunchMessage(deleted, "https://hooks.slack.test/response")
	if msg.ResponseType != slack.ResponseTypeEphemeral {
		t.Fatalf("responseType = %q", msg.ResponseType)
	}
	if !msg.ReplaceOriginal {
		t.Fatal("ReplaceOriginal = false")
	}
	if msg.Blocks == nil {
		t.Fatalf("blocks = %+v", msg.Blocks)
	}

	if hasActionBlock(msg.Blocks) {
		t.Fatalf("canceled message unexpectedly contains follow-up actions: %+v", msg.Blocks.BlockSet)
	}

	section := findSectionText(t, msg.Blocks)
	if got, want := section.Text, "*Launch request canceled*\n*Request ID:* `request-1`\n*ScaledObject:* `default/worker`\n*Effective window:* `2026-04-25T12:00:00Z` - `2026-04-25T12:10:00Z`"; got != want {
		t.Fatalf("section text = %q, want %q", got, want)
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

func hasActionBlock(blocks *slack.Blocks) bool {
	for _, block := range blocks.BlockSet {
		if _, ok := block.(*slack.ActionBlock); ok {
			return true
		}
	}
	return false
}

func findSectionText(t *testing.T, blocks *slack.Blocks) *slack.TextBlockObject {
	t.Helper()

	for _, block := range blocks.BlockSet {
		section, ok := block.(*slack.SectionBlock)
		if ok && section.Text != nil {
			return section.Text
		}
	}
	t.Fatal("section block was not found")
	return nil
}
