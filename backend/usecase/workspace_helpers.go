package usecase

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/koezuka404/notehub/entity"
)

const (
	maxWorkspaceNameLength  = 100
	minDeleteReasonLength   = 1
	maxDeleteReasonLength   = 500
	unavailableReasonNone   = ""
	unavailableHostSuspended = "HOST_SUSPENDED"
	unavailableHostDeleted   = "HOST_DELETED"
)

func normalizeWorkspaceName(name string) string {
	return strings.TrimSpace(name)
}

func validateWorkspaceName(name string) error {
	name = normalizeWorkspaceName(name)
	if name == "" {
		return ErrValidation
	}
	if utf8.RuneCountInString(name) > maxWorkspaceNameLength {
		return ErrValidation
	}
	return nil
}

func validateDeleteReason(reason string) error {
	reason = strings.TrimSpace(reason)
	length := utf8.RuneCountInString(reason)
	if length < minDeleteReasonLength || length > maxDeleteReasonLength {
		return ErrValidation
	}
	return nil
}

type auditDeleteMetadata struct {
	Reason     string `json:"reason"`
	DeleteType string `json:"delete_type"`
}

func workspaceDeleteMetadata(reason string) (json.RawMessage, error) {
	raw, err := json.Marshal(auditDeleteMetadata{Reason: reason, DeleteType: "logical"})
	if err != nil {
		return nil, fmt.Errorf("marshal delete metadata: %w", err)
	}
	return raw, nil
}

func workspaceAvailability(host *entity.User) (bool, string) {
	if host.CanAuthenticate() {
		return true, unavailableReasonNone
	}
	if host.IsSuspended() {
		return false, unavailableHostSuspended
	}
	if host.IsDeleted() {
		return false, unavailableHostDeleted
	}
	return false, unavailableHostSuspended
}
