package ui

import (
	"fmt"
	"testing"
	"time"

	domainclient "github.com/Kotaro7750/keda-launcher-scaler/pkg/client"
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

func TestBuildLaunchModalGroupsTargetsByNamespace(t *testing.T) {
	modal := fakeCommandInvocationMetadata.BuildLaunchModal([]domainclient.ScaledObject{
		{Namespace: "zeta", Name: "beta"},
		{Namespace: "default", Name: "worker-b"},
		{Namespace: "default", Name: "worker-a"},
	})

	if len(modal.Blocks.BlockSet) != 2 {
		t.Fatalf("blocks = %d", len(modal.Blocks.BlockSet))
	}
	targetBlock, ok := modal.Blocks.BlockSet[0].(*slack.InputBlock)
	if !ok {
		t.Fatalf("target block type = %T", modal.Blocks.BlockSet[0])
	}
	selectElement, ok := targetBlock.Element.(*slack.SelectBlockElement)
	if !ok {
		t.Fatalf("target element type = %T", targetBlock.Element)
	}
	if len(selectElement.OptionGroups) != 2 {
		t.Fatalf("option groups = %d", len(selectElement.OptionGroups))
	}
	if selectElement.OptionGroups[0].Label.Text != "default" {
		t.Fatalf("first group label = %q", selectElement.OptionGroups[0].Label.Text)
	}
	if selectElement.OptionGroups[0].Options[0].Text.Text != "worker-a" {
		t.Fatalf("first option = %q", selectElement.OptionGroups[0].Options[0].Text.Text)
	}
	if selectElement.OptionGroups[0].Options[0].Value != `{"namespace":"default","name":"worker-a"}` {
		t.Fatalf("first option value = %q", selectElement.OptionGroups[0].Options[0].Value)
	}
}

func TestParseLaunchModalParseCorrectly(t *testing.T) {
	now := time.Date(2026, 4, 25, 12, 0, 0, 123, time.UTC)
	req, fieldErrors := fakeCommandInvocationMetadata.ParseLaunchModal(slack.View{
		PrivateMetadata: fakeCommandInvocationMetadata.Encode(),
		State: &slack.ViewState{Values: map[string]map[string]slack.BlockAction{
			kedaTargetBlockID: {
				kedaTargetAction: {SelectedOption: slack.OptionBlockObject{Value: `{"namespace":" default ","name":" worker "}`}},
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

func TestParseLaunchModalValidateCorrectly(t *testing.T) {
	_, fieldErrors := fakeCommandInvocationMetadata.ParseLaunchModal(
		slack.View{
			State: &slack.ViewState{Values: map[string]map[string]slack.BlockAction{
				kedaTargetBlockID: {
					kedaTargetAction: {SelectedOption: slack.OptionBlockObject{Value: ""}},
				},
				KedaDurationBlockID: {
					kedaDurationAction: {Value: "soon"},
				},
			}},
		}, time.Now())

	if len(fieldErrors) != 2 {
		t.Fatalf("fieldErrors = %v", fieldErrors)
	}
	if fieldErrors[kedaTargetBlockID] == "" {
		t.Fatalf("missing target error: %v", fieldErrors)
	}
	if fieldErrors[KedaDurationBlockID] == "" {
		t.Fatalf("missing duration error: %v", fieldErrors)
	}
}

func TestParseLaunchModalReturnsErrorForBrokenTargetValue(t *testing.T) {
	_, fieldErrors := fakeCommandInvocationMetadata.ParseLaunchModal(
		slack.View{
			State: &slack.ViewState{Values: map[string]map[string]slack.BlockAction{
				kedaTargetBlockID: {
					kedaTargetAction: {SelectedOption: slack.OptionBlockObject{Value: "not-json"}},
				},
				KedaDurationBlockID: {
					kedaDurationAction: {Value: "10m"},
				},
			}},
		}, time.Now())

	if fieldErrors[kedaTargetBlockID] != "Target selection is invalid." {
		t.Fatalf("target error = %q", fieldErrors[kedaTargetBlockID])
	}
}
