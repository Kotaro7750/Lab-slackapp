package ui

import "testing"

func TestCommandInvocationMetadataEncodeDecodeRoundTrip(t *testing.T) {
	metadata := CommandInvocationMetadata{
		UserID:      fakeUserID,
		ChannelID:   fakeChannelID,
		ResponseURL: fakeResponseURL,
	}

	decoded, err := DecodeCommandInvocationMetadata(metadata.Encode())
	if err != nil {
		t.Fatalf("DecodeCommandInvocationMetadata() error = %v", err)
	}

	if decoded != metadata {
		t.Fatalf("decoded = %+v", decoded)
	}
}

func TestDecodeLaunchRequestMetadataRoundTrip(t *testing.T) {
	metadata := LaunchRequestMetadata{
		RequestID:   "request-1",
		Namespace:   "default",
		Name:        "worker",
		Duration:    "10m",
		ResponseURL: fakeResponseURL,
	}

	decoded, err := DecodeLaunchRequestMetadata(metadata.Encode())
	if err != nil {
		t.Fatalf("DecodeLaunchRequestMetadata() error = %v", err)
	}

	if decoded != metadata {
		t.Fatalf("decoded = %+v", decoded)
	}
}

func TestDecodeLaunchRequestMetadataFailsWhenRequiredFieldIsMissing(t *testing.T) {
	_, err := DecodeLaunchRequestMetadata(`{"request_id":"request-1","namespace":"default","name":"worker"}`)
	if err == nil {
		t.Fatal("DecodeLaunchRequestMetadata() error = nil")
	}
}
