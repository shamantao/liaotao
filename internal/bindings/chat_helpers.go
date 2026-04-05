/*
  chat_helpers.go -- Internal helpers for chat bindings.
  Responsibilities: Wails event emission, cancellation token management,
  assistant message persistence.
*/

package bindings

import (
	"context"
	"fmt"
	"strconv"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// emitStructuredError emits a chat:error event with a user-friendly payload.
func (s *Service) emitStructuredError(convID string, err error) {
	var payload map[string]any
	if pe, ok := err.(ProviderError); ok {
		payload = map[string]any{
			"conversation_id": convID,
			"code":            pe.Code,
			"message":         pe.Message,
			"retryable":       pe.Retryable,
		}
	} else {
		payload = map[string]any{
			"conversation_id": convID,
			"code":            "unknown",
			"message":         fmt.Sprintf("Generation failed: %s", err.Error()),
			"retryable":       false,
		}
	}
	s.emit("chat:error", payload)
}

func (s *Service) emit(name string, payload any) {
	app := application.Get()
	if app == nil || app.Event == nil {
		return
	}
	app.Event.Emit(name, payload)
}

func (s *Service) setCancel(convID string, cancel context.CancelFunc) {
	s.cancelMu.Lock()
	defer s.cancelMu.Unlock()

	if previous, ok := s.cancels[convID]; ok {
		previous()
	}
	s.cancels[convID] = cancel
}

func (s *Service) clearCancel(convID string) {
	s.cancelMu.Lock()
	defer s.cancelMu.Unlock()
	delete(s.cancels, convID)
}

func (s *Service) cancelStream(convID string) bool {
	s.cancelMu.Lock()
	defer s.cancelMu.Unlock()

	cancel, ok := s.cancels[convID]
	if !ok {
		return false
	}
	cancel()
	delete(s.cancels, convID)
	return true
}

// persistAssistantMessage saves a completed assistant reply to the conversation DB.
// Non-numeric or zero conversation IDs (e.g. "default") are silently skipped.
func (s *Service) persistAssistantMessage(ctx context.Context, convID, content string) {
	if content == "" {
		return
	}
	id, err := strconv.ParseInt(convID, 10, 64)
	if err != nil || id <= 0 {
		return
	}
	_ = s.SaveMessage(ctx, MessagePayload{
		ConversationID: id,
		Role:           "assistant",
		Content:        content,
	})
}
