package ui

import (
	"fmt"
	"strings"
	"time"

	domainclient "github.com/Kotaro7750/keda-launcher-scaler/pkg/client"
	"github.com/slack-go/slack"
)

// These IDs define the single launch form artifact: its fields, validation keys, and state lookup.
const (
	kedaNamespaceBlockID = "keda_namespace"
	kedaNamespaceAction  = "namespace"
	kedaNameBlockID      = "keda_scaled_object_name"
	kedaNameAction       = "scaled_object_name"
	KedaDurationBlockID  = "keda_duration"
	kedaDurationAction   = "duration"
)

// BuildLaunchModal creates the initial modal shown after the slash command entrypoint.
func (metadata *CommandInvocationMetadata) BuildLaunchModal() slack.ModalViewRequest {
	return slack.ModalViewRequest{
		Type:            slack.VTModal,
		CallbackID:      KedaLaunchCallbackID,
		PrivateMetadata: metadata.Encode(),
		Title:           slack.NewTextBlockObject(slack.PlainTextType, "KEDA launch", false, false),
		Close:           slack.NewTextBlockObject(slack.PlainTextType, "Cancel", false, false),
		Submit:          slack.NewTextBlockObject(slack.PlainTextType, "Send", false, false),
		Blocks: slack.Blocks{BlockSet: []slack.Block{
			textInputBlock(kedaNamespaceBlockID, kedaNamespaceAction, "Namespace", "default", ""),
			textInputBlock(kedaNameBlockID, kedaNameAction, "ScaledObject name", "worker", ""),
			textInputBlock(KedaDurationBlockID, kedaDurationAction, "Duration", "10m", ""),
		}},
	}
}

// ParseLaunchModal validates the launch form and converts it into a launcher request.
func (metadata *CommandInvocationMetadata) ParseLaunchModal(view slack.View, now time.Time) (domainclient.LaunchRequest, map[string]string) {
	// Collect all field errors so Slack can render them beside each input at once.
	fieldErrors := make(map[string]string)

	namespace, namespaceOK := inputValue(view, kedaNamespaceBlockID, kedaNamespaceAction)
	namespace = strings.TrimSpace(namespace)
	if !namespaceOK || namespace == "" {
		fieldErrors[kedaNamespaceBlockID] = "Namespace is required."
	}

	name, nameOK := inputValue(view, kedaNameBlockID, kedaNameAction)
	name = strings.TrimSpace(name)
	if !nameOK || name == "" {
		fieldErrors[kedaNameBlockID] = "ScaledObject name is required."
	}

	durationValue, durationOK := inputValue(view, KedaDurationBlockID, kedaDurationAction)
	durationValue = strings.TrimSpace(durationValue)
	duration, ok := parsePositiveDuration(durationValue)
	if !durationOK || !ok {
		fieldErrors[KedaDurationBlockID] = "Duration must be a positive Go duration such as 10m or 1h."
	}

	if len(fieldErrors) > 0 {
		return domainclient.LaunchRequest{}, fieldErrors
	}

	return domainclient.LaunchRequest{
		RequestID: generateRequestID(metadata.UserID, metadata.ChannelID, namespace, name, now),
		ScaledObject: domainclient.ScaledObject{
			Namespace: namespace,
			Name:      name,
		},
		Duration: duration,
	}, nil
}

// generateRequestID creates a stable-enough request id for Slack-originated launches.
func generateRequestID(userID, channelID, namespace, name string, now time.Time) string {
	return fmt.Sprintf(
		"slack:%s:%s:%s/%s:%d",
		strings.TrimSpace(userID),
		strings.TrimSpace(channelID),
		strings.TrimSpace(namespace),
		strings.TrimSpace(name),
		now.UnixNano(),
	)
}
