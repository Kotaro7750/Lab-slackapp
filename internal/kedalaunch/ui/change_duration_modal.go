package ui

import (
	"fmt"
	"strings"

	domainclient "github.com/Kotaro7750/keda-launcher-scaler/pkg/client"
	"github.com/slack-go/slack"
)

// BuildChangeDurationModal creates the form used to update an accepted request duration.
func (metadata *LaunchRequestMetadata) BuildChangeDurationModal() slack.ModalViewRequest {
	return slack.ModalViewRequest{
		Type:            slack.VTModal,
		CallbackID:      KedaChangeCallbackID,
		PrivateMetadata: metadata.Encode(),
		Title:           slack.NewTextBlockObject(slack.PlainTextType, "Change duration", false, false),
		Close:           slack.NewTextBlockObject(slack.PlainTextType, "Cancel", false, false),
		Submit:          slack.NewTextBlockObject(slack.PlainTextType, "Send", false, false),
		Blocks: slack.Blocks{BlockSet: []slack.Block{
			slack.NewSectionBlock(
				slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("*ScaledObject:* `%s/%s`\n*Request ID:* `%s`", metadata.Namespace, metadata.Name, metadata.RequestID), false, false),
				nil,
				nil,
			),
			textInputBlock(KedaDurationBlockID, kedaDurationAction, "Duration", "10m", metadata.Duration),
		}},
	}
}

// ParseChangeDurationModal validates a duration change while preserving the original target.
func (metadata *LaunchRequestMetadata) ParseChangeDurationModal(view slack.View) (domainclient.LaunchRequest, map[string]string) {
	durationValue, durationOK := inputValue(view, KedaDurationBlockID, kedaDurationAction)
	duration, ok := parsePositiveDuration(strings.TrimSpace(durationValue))
	if !durationOK || !ok {
		return domainclient.LaunchRequest{}, map[string]string{
			KedaDurationBlockID: "Duration must be a positive Go duration such as 10m or 1h.",
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
	}, nil
}
