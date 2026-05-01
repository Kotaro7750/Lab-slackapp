package kedalaunch

import (
	"fmt"
	"strings"

	domainclient "github.com/Kotaro7750/keda-launcher-scaler/pkg/client"
	"github.com/slack-go/slack"
)

const (
	kedaChangeCallbackID = "keda_launch_change"
	kedaChangeActionID   = "change_keda_launch_request"
)

// buildKedaChangeModal creates the form used to update an accepted request duration.
func buildKedaChangeModal(metadata acceptedRequestMetadata, encodedMetadata string) slack.ModalViewRequest {
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
	metadata, err := decodeAcceptedRequestMetadata(view.PrivateMetadata)
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

// acceptedRequestActionFromChangeAction extracts the accepted-response contract from the pressed change button.
func acceptedRequestActionFromChangeAction(interaction slack.InteractionCallback) (acceptedRequestAction, error) {
	if len(interaction.ActionCallback.BlockActions) == 0 {
		return acceptedRequestAction{ResponseURL: interaction.ResponseURL}, fmt.Errorf("missing block action")
	}
	metadata, err := decodeAcceptedRequestMetadata(interaction.ActionCallback.BlockActions[0].Value)
	if err != nil {
		return acceptedRequestAction{ResponseURL: interaction.ResponseURL}, err
	}
	return acceptedRequestAction{Metadata: metadata, ResponseURL: interaction.ResponseURL}, nil
}
