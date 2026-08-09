package dto

import (
	"encoding/json"
	"testing"
)

func TestNewErrorResponse(t *testing.T) {
	t.Run("without request id", func(t *testing.T) {
		resp := NewErrorResponse("INVALID_REQUEST", "bad input", "")
		if resp.Error.Code != "INVALID_REQUEST" {
			t.Fatalf("Code = %q", resp.Error.Code)
		}
		if resp.Error.Message != "bad input" {
			t.Fatalf("Message = %q", resp.Error.Message)
		}
		if resp.Error.RequestID != "" {
			t.Fatalf("RequestID = %q", resp.Error.RequestID)
		}

		raw, err := json.Marshal(resp)
		if err != nil {
			t.Fatalf("Marshal: %v", err)
		}
		if string(raw) != `{"error":{"code":"INVALID_REQUEST","message":"bad input"}}` {
			t.Fatalf("json = %s", raw)
		}
	})

	t.Run("with request id", func(t *testing.T) {
		resp := NewErrorResponse("UNAUTHORIZED", "login required", "req-123")
		if resp.Error.RequestID != "req-123" {
			t.Fatalf("RequestID = %q", resp.Error.RequestID)
		}

		raw, err := json.Marshal(resp)
		if err != nil {
			t.Fatalf("Marshal: %v", err)
		}
		if string(raw) != `{"error":{"code":"UNAUTHORIZED","message":"login required","requestId":"req-123"}}` {
			t.Fatalf("json = %s", raw)
		}
	})
}
