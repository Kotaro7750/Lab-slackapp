package ui

import (
	"time"

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

// parsePositiveDuration accepts only positive Go duration strings.
func parsePositiveDuration(value string) (time.Duration, bool) {
	duration, err := time.ParseDuration(value)
	if err != nil || duration <= 0 {
		return 0, false
	}
	return duration, true
}
