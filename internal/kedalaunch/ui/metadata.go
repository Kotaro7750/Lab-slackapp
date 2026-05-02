package ui

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/slack-go/slack"
)

// CommandInvocationMetadata carries slash-command context into the first launch modal submit.
type CommandInvocationMetadata struct {
	UserID      string `json:"user_id"`
	ChannelID   string `json:"channel_id"`
	ResponseURL string `json:"response_url"`
}

// DecodeCommandInvocationMetadata restores the slash-command context from modal private metadata.
func DecodeCommandInvocationMetadata(value string) (CommandInvocationMetadata, error) {
	var metadata CommandInvocationMetadata
	if strings.TrimSpace(value) == "" {
		return CommandInvocationMetadata{}, fmt.Errorf("metadata is empty")
	}
	if err := json.Unmarshal([]byte(value), &metadata); err != nil {
		return CommandInvocationMetadata{}, err
	}
	return metadata, nil
}

// Encode serializes the slash-command context for modal private metadata.
func (m *CommandInvocationMetadata) Encode() string {
	encoded, err := json.Marshal(m)
	if err != nil {
		panic(fmt.Sprintf("string only struct couldn't return error on json.Marshal, but error occured: %v", err))
	}

	return string(encoded)
}

// LaunchRequestMetadata is the shared follow-up contract owned by the accepted response artifact.
type LaunchRequestMetadata struct {
	RequestID   string `json:"request_id"`
	Namespace   string `json:"namespace"`
	Name        string `json:"name"`
	Duration    string `json:"duration"`
	ResponseURL string `json:"response_url"`
}

func LaunchRequestMetadataFromAction(interaction slack.InteractionCallback) (LaunchRequestMetadata, error) {
	if len(interaction.ActionCallback.BlockActions) == 0 {
		return LaunchRequestMetadata{ResponseURL: interaction.ResponseURL}, fmt.Errorf("missing block action")
	}
	return DecodeLaunchRequestMetadata(interaction.ActionCallback.BlockActions[0].Value)
}

// DecodeLaunchRequestMetadata restores the accepted-response follow-up contract from Slack metadata.
func DecodeLaunchRequestMetadata(value string) (LaunchRequestMetadata, error) {
	var metadata LaunchRequestMetadata
	if strings.TrimSpace(value) == "" {
		return LaunchRequestMetadata{}, fmt.Errorf("metadata is empty")
	}
	if err := json.Unmarshal([]byte(value), &metadata); err != nil {
		return LaunchRequestMetadata{}, err
	}
	if metadata.RequestID == "" || metadata.Namespace == "" || metadata.Name == "" || metadata.ResponseURL == "" {
		return LaunchRequestMetadata{}, fmt.Errorf("missing required metadata")
	}
	return metadata, nil
}

// Encode serializes the accepted-response follow-up contract for buttons and modals.
func (metadata *LaunchRequestMetadata) Encode() string {
	encoded, err := json.Marshal(metadata)
	if err != nil {
		panic(fmt.Sprintf("string only struct couldn't return error on json.Marshal, but error occured: %v", err))
	}

	return string(encoded)
}
