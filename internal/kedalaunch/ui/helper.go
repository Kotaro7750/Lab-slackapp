package ui

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	domainclient "github.com/Kotaro7750/keda-launcher-scaler/pkg/client"
	"github.com/slack-go/slack"
)

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

// selectValue reads the selected option value from a Slack modal state.
func selectValue(view slack.View, blockID, actionID string) (string, bool) {
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
	value := strings.TrimSpace(action.SelectedOption.Value)
	if value == "" {
		return "", false
	}
	return value, true
}

// decodeLaunchTarget restores a ScaledObject from the select option JSON payload.
func decodeLaunchTarget(value string) (domainclient.ScaledObject, error) {
	var target domainclient.ScaledObject
	if strings.TrimSpace(value) == "" {
		return domainclient.ScaledObject{}, fmt.Errorf("launch target is empty")
	}
	if err := json.Unmarshal([]byte(value), &target); err != nil {
		return domainclient.ScaledObject{}, err
	}
	target.Namespace = strings.TrimSpace(target.Namespace)
	target.Name = strings.TrimSpace(target.Name)
	if target.Namespace == "" || target.Name == "" {
		return domainclient.ScaledObject{}, fmt.Errorf("launch target is incomplete")
	}
	return target, nil
}

// parsePositiveDuration accepts only positive Go duration strings.
func parsePositiveDuration(value string) (time.Duration, bool) {
	duration, err := time.ParseDuration(value)
	if err != nil || duration <= 0 {
		return 0, false
	}
	return duration, true
}
