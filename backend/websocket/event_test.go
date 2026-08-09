package websocket

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestMarshalEvent(t *testing.T) {
	payload, err := MarshalEvent(EventConnected, ConnectedData{ConnectionID: "conn-1"}, testNow())
	if err != nil {
		t.Fatalf("MarshalEvent: %v", err)
	}
	var envelope Envelope
	if err := json.Unmarshal(payload, &envelope); err != nil {
		t.Fatalf("unmarshal envelope: %v", err)
	}
	if envelope.Type != EventConnected {
		t.Fatalf("type = %q", envelope.Type)
	}
}

func TestMarshalEvent_DataMarshalError(t *testing.T) {
	orig := jsonMarshalFn
	jsonMarshalFn = func(any) ([]byte, error) { return nil, errors.New("marshal failed") }
	t.Cleanup(func() { jsonMarshalFn = orig })

	_, err := MarshalEvent(EventError, ErrorData{Code: "X", Message: "y"}, testNow())
	if err == nil {
		t.Fatal("expected marshal error")
	}
}
