package kedalaunch

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	domainclient "github.com/Kotaro7750/keda-launcher-scaler/pkg/client"
	"github.com/slack-go/slack"
)

const (
	kedaLaunchCallbackID = "keda_launch_request"

	kedaNamespaceBlockID = "keda_namespace"
	kedaNamespaceAction  = "namespace"
	kedaNameBlockID      = "keda_scaled_object_name"
	kedaNameAction       = "scaled_object_name"
	kedaDurationBlockID  = "keda_duration"
	kedaDurationAction   = "duration"
)

// launchModalMetadata carries slash-command context through the launch modal.
type launchModalMetadata struct {
	UserID      string `json:"user_id"`
	ChannelID   string `json:"channel_id"`
	ResponseURL string `json:"response_url"`
}

// buildKedaLaunchModal creates the initial form shown by the /launch command.
func buildKedaLaunchModal(metadata string) slack.ModalViewRequest {
	return slack.ModalViewRequest{
		Type:            slack.VTModal,
		CallbackID:      kedaLaunchCallbackID,
		PrivateMetadata: metadata,
		Title:           slack.NewTextBlockObject(slack.PlainTextType, "KEDA launch", false, false),
		Close:           slack.NewTextBlockObject(slack.PlainTextType, "Cancel", false, false),
		Submit:          slack.NewTextBlockObject(slack.PlainTextType, "Send", false, false),
		Blocks: slack.Blocks{BlockSet: []slack.Block{
			textInputBlock(kedaNamespaceBlockID, kedaNamespaceAction, "Namespace", "default", ""),
			textInputBlock(kedaNameBlockID, kedaNameAction, "ScaledObject name", "worker", ""),
			textInputBlock(kedaDurationBlockID, kedaDurationAction, "Duration", "10m", ""),
		}},
	}
}

// parseLaunchSubmission validates the launch modal and converts it into a KEDA request.
func parseLaunchSubmission(view slack.View, now time.Time) (domainclient.LaunchRequest, string, map[string]string) {
	metadata, err := decodeLaunchModalMetadata(view.PrivateMetadata)
	if err != nil {
		return domainclient.LaunchRequest{}, "", map[string]string{kedaDurationBlockID: "Invalid form metadata."}
	}

	namespace, namespaceOK := inputValue(view, kedaNamespaceBlockID, kedaNamespaceAction)
	name, nameOK := inputValue(view, kedaNameBlockID, kedaNameAction)
	durationValue, durationOK := inputValue(view, kedaDurationBlockID, kedaDurationAction)

	fieldErrors := make(map[string]string)
	namespace = strings.TrimSpace(namespace)
	name = strings.TrimSpace(name)
	durationValue = strings.TrimSpace(durationValue)

	// Collect all field errors so Slack can render them beside each input at once.
	if !namespaceOK || namespace == "" {
		fieldErrors[kedaNamespaceBlockID] = "Namespace is required."
	}
	if !nameOK || name == "" {
		fieldErrors[kedaNameBlockID] = "ScaledObject name is required."
	}

	duration, ok := parsePositiveDuration(durationValue)
	if !durationOK || !ok {
		fieldErrors[kedaDurationBlockID] = "Duration must be a positive Go duration such as 10m or 1h."
	}
	if len(fieldErrors) > 0 {
		return domainclient.LaunchRequest{}, metadata.ResponseURL, fieldErrors
	}

	return domainclient.LaunchRequest{
		RequestID: generateRequestID(metadata.UserID, metadata.ChannelID, namespace, name, now),
		ScaledObject: domainclient.ScaledObject{
			Namespace: namespace,
			Name:      name,
		},
		Duration: duration,
	}, metadata.ResponseURL, nil
}

// encodeLaunchModalMetadata serializes the context needed after modal submission.
func encodeLaunchModalMetadata(metadata launchModalMetadata) (string, error) {
	raw, err := json.Marshal(metadata)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

// decodeLaunchModalMetadata restores the slash-command context from private metadata.
func decodeLaunchModalMetadata(value string) (launchModalMetadata, error) {
	var metadata launchModalMetadata
	if strings.TrimSpace(value) == "" {
		return launchModalMetadata{}, fmt.Errorf("metadata is empty")
	}
	if err := json.Unmarshal([]byte(value), &metadata); err != nil {
		return launchModalMetadata{}, err
	}
	return metadata, nil
}

// textInputBlock builds a Slack input block with an optional initial value.
func textInputBlock(blockID, actionID, label, placeholder, initialValue string) *slack.InputBlock {
	element := slack.NewPlainTextInputBlockElement(slack.NewTextBlockObject(slack.PlainTextType, placeholder, false, false), actionID)
	if initialValue != "" {
		element.WithInitialValue(initialValue)
	}
	return slack.NewInputBlock(blockID, slack.NewTextBlockObject(slack.PlainTextType, label, false, false), nil, element)
}

// inputValue reads a text input value from a Slack modal state.
func inputValue(view slack.View, blockID, actionID string) (string, bool) {
	if view.State == nil || view.State.Values == nil {
		return "", false
	}
	actions, ok := view.State.Values[blockID]
	if !ok {
		return "", false
	}
	action, ok := actions[actionID]
	if !ok {
		return "", false
	}
	return action.Value, true
}

// parsePositiveDuration accepts only positive Go duration strings.
func parsePositiveDuration(value string) (time.Duration, bool) {
	duration, err := time.ParseDuration(value)
	if err != nil || duration <= 0 {
		return 0, false
	}
	return duration, true
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
