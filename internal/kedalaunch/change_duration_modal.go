package kedalaunch

import (
	"encoding/json"
	"fmt"
	"strings"

	domainclient "github.com/Kotaro7750/keda-launcher-scaler/pkg/client"
	"github.com/slack-go/slack"
)

const (
	kedaChangeCallbackID = "keda_launch_change"
	kedaChangeActionID   = "change_keda_launch_request"
)

// kedaRequestMetadata carries an accepted request through the change-duration flow.
type kedaRequestMetadata struct {
	RequestID   string `json:"request_id"`
	Namespace   string `json:"namespace"`
	Name        string `json:"name"`
	Duration    string `json:"duration"`
	UserID      string `json:"user_id"`
	ChannelID   string `json:"channel_id"`
	ResponseURL string `json:"response_url"`
}

// buildKedaChangeModal creates the form used to update an accepted request duration.
func buildKedaChangeModal(metadata kedaRequestMetadata, encodedMetadata string) slack.ModalViewRequest {
	return slack.ModalViewRequest{
		Type:            slack.VTModal,
		CallbackID:      kedaChangeCallbackID,
		PrivateMetadata: encodedMetadata,
		Title:           slack.NewTextBlockObject(slack.PlainTextType, "Change duration", false, false),
		Close:           slack.NewTextBlockObject(slack.PlainTextType, "Cancel", false, false),
		Submit:          slack.NewTextBlockObject(slack.PlainTextType, "Send", false, false),
		Blocks: slack.Blocks{BlockSet: []slack.Block{
			slack.NewSectionBlock(
				slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("*ScaledObject:* `%s/%s`\n*Request ID:* `%s`", metadata.Namespace, metadata.Name, metadata.RequestID), false, false),
				nil,
				nil,
			),
			textInputBlock(kedaDurationBlockID, kedaDurationAction, "Duration", "10m", metadata.Duration),
		}},
	}
}

// parseChangeSubmission validates a duration change while preserving the original target.
func parseChangeSubmission(view slack.View) (domainclient.LaunchRequest, string, map[string]string) {
	metadata, err := decodeKedaRequestMetadata(view.PrivateMetadata)
	if err != nil {
		return domainclient.LaunchRequest{}, "", map[string]string{kedaDurationBlockID: "Invalid form metadata."}
	}

	durationValue, durationOK := inputValue(view, kedaDurationBlockID, kedaDurationAction)
	duration, ok := parsePositiveDuration(strings.TrimSpace(durationValue))
	if !durationOK || !ok {
		return domainclient.LaunchRequest{}, metadata.ResponseURL, map[string]string{
			kedaDurationBlockID: "Duration must be a positive Go duration such as 10m or 1h.",
		}
	}

	// The accepted message owns request id and ScaledObject; the modal may only change duration.
	return domainclient.LaunchRequest{
		RequestID: metadata.RequestID,
		ScaledObject: domainclient.ScaledObject{
			Namespace: metadata.Namespace,
			Name:      metadata.Name,
		},
		Duration: duration,
	}, metadata.ResponseURL, nil
}

// metadataFromChangeAction extracts request metadata from the pressed change button.
func metadataFromChangeAction(interaction slack.InteractionCallback) (kedaRequestMetadata, error) {
	if len(interaction.ActionCallback.BlockActions) == 0 {
		return kedaRequestMetadata{}, fmt.Errorf("missing block action")
	}
	return decodeKedaRequestMetadata(interaction.ActionCallback.BlockActions[0].Value)
}

// encodeKedaRequestMetadata serializes request context for buttons and modals.
func encodeKedaRequestMetadata(metadata kedaRequestMetadata) (string, error) {
	raw, err := json.Marshal(metadata)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

// decodeKedaRequestMetadata restores request context and checks required routing fields.
func decodeKedaRequestMetadata(value string) (kedaRequestMetadata, error) {
	var metadata kedaRequestMetadata
	if strings.TrimSpace(value) == "" {
		return kedaRequestMetadata{}, fmt.Errorf("metadata is empty")
	}
	if err := json.Unmarshal([]byte(value), &metadata); err != nil {
		return kedaRequestMetadata{}, err
	}
	if metadata.RequestID == "" || metadata.Namespace == "" || metadata.Name == "" || metadata.ResponseURL == "" {
		return kedaRequestMetadata{}, fmt.Errorf("missing required metadata")
	}
	return metadata, nil
}
