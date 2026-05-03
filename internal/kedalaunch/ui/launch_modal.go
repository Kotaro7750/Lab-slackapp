package ui

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	domainclient "github.com/Kotaro7750/keda-launcher-scaler/pkg/client"
	"github.com/slack-go/slack"
)

// These IDs define the single launch form artifact: its fields, validation keys, and state lookup.
const (
	kedaTargetBlockID   = "keda_target"
	kedaTargetAction    = "target"
	KedaDurationBlockID = "keda_duration"
	kedaDurationAction  = "duration"
)

type launchTargetRef struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
}

// BuildLaunchModal creates the initial modal shown after the slash command entrypoint.
func (metadata *CommandInvocationMetadata) BuildLaunchModal(candidates []domainclient.ScaledObject) slack.ModalViewRequest {
	return slack.ModalViewRequest{
		Type:            slack.VTModal,
		CallbackID:      KedaLaunchCallbackID,
		PrivateMetadata: metadata.Encode(),
		Title:           slack.NewTextBlockObject(slack.PlainTextType, "KEDA launch", false, false),
		Close:           slack.NewTextBlockObject(slack.PlainTextType, "Cancel", false, false),
		Submit:          slack.NewTextBlockObject(slack.PlainTextType, "Send", false, false),
		Blocks: slack.Blocks{BlockSet: []slack.Block{
			selectInputBlock(kedaTargetBlockID, kedaTargetAction, "Target", "Select a ScaledObject", buildTargetOptionGroups(candidates)),
			textInputBlock(KedaDurationBlockID, kedaDurationAction, "Duration", "10m", ""),
		}},
	}
}

// ParseLaunchModal validates the launch form and converts it into a launcher request.
func (metadata *CommandInvocationMetadata) ParseLaunchModal(view slack.View, now time.Time) (domainclient.LaunchRequest, map[string]string) {
	// Collect all field errors so Slack can render them beside each input at once.
	fieldErrors := make(map[string]string)

	targetValue, targetOK := selectValue(view, kedaTargetBlockID, kedaTargetAction)
	target, err := decodeLaunchTarget(targetValue)
	if !targetOK {
		fieldErrors[kedaTargetBlockID] = "Target selection is required."
	} else if err != nil {
		fieldErrors[kedaTargetBlockID] = "Target selection is invalid."
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
		RequestID:    generateRequestID(metadata.UserID, metadata.ChannelID, target.Namespace, target.Name, now),
		ScaledObject: target,
		Duration:     duration,
	}, nil
}

func selectInputBlock(blockID, actionID, label, placeholder string, optionGroups []*slack.OptionGroupBlockObject) *slack.InputBlock {
	element := slack.NewOptionsGroupSelectBlockElement(
		slack.OptTypeStatic,
		slack.NewTextBlockObject(slack.PlainTextType, placeholder, false, false),
		actionID,
		optionGroups...,
	)
	return slack.NewInputBlock(blockID, slack.NewTextBlockObject(slack.PlainTextType, label, false, false), nil, element)
}

func buildTargetOptionGroups(candidates []domainclient.ScaledObject) []*slack.OptionGroupBlockObject {
	grouped := make(map[string][]string)
	for _, candidate := range candidates {
		namespace := strings.TrimSpace(candidate.Namespace)
		name := strings.TrimSpace(candidate.Name)
		if namespace == "" || name == "" {
			continue
		}
		grouped[namespace] = append(grouped[namespace], name)
	}

	namespaces := make([]string, 0, len(grouped))
	for namespace := range grouped {
		namespaces = append(namespaces, namespace)
	}
	sort.Strings(namespaces)

	optionGroups := make([]*slack.OptionGroupBlockObject, 0, len(namespaces))
	for _, namespace := range namespaces {
		names := grouped[namespace]
		sort.Strings(names)

		options := make([]*slack.OptionBlockObject, 0, len(names))
		for _, name := range names {
			options = append(options, slack.NewOptionBlockObject(
				mustEncodeLaunchTarget(namespace, name),
				slack.NewTextBlockObject(slack.PlainTextType, name, false, false),
				nil,
			))
		}

		optionGroups = append(optionGroups, slack.NewOptionGroupBlockElement(
			slack.NewTextBlockObject(slack.PlainTextType, namespace, false, false),
			options...,
		))
	}

	return optionGroups
}

func mustEncodeLaunchTarget(namespace, name string) string {
	encoded, err := json.Marshal(launchTargetRef{
		Namespace: strings.TrimSpace(namespace),
		Name:      strings.TrimSpace(name),
	})
	if err != nil {
		panic(fmt.Sprintf("string only struct couldn't return error on json.Marshal, but error occured: %v", err))
	}
	return string(encoded)
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
