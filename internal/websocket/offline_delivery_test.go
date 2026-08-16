package websocket

import (
	"context"
	"encoding/json"
	"testing"

	"we-chat/internal/models"
)

type memoryOfflineStore struct {
	ready   []*offlineMessageClaim
	claimed map[string]*offlineMessageClaim
}

func newMemoryOfflineStore(contents ...string) *memoryOfflineStore {
	ready := make([]*offlineMessageClaim, 0, len(contents))
	for i, content := range contents {
		ready = append(ready, &offlineMessageClaim{ID: string(rune('a' + i)), Message: models.Message{Content: content}})
	}
	return &memoryOfflineStore{ready: ready, claimed: make(map[string]*offlineMessageClaim)}
}

func (s *memoryOfflineStore) ClaimNext(_ context.Context, _ string) (*offlineMessageClaim, error) {
	if len(s.ready) == 0 {
		return nil, nil
	}
	claim := s.ready[0]
	s.ready = s.ready[1:]
	s.claimed[claim.ID] = claim
	return claim, nil
}
func (s *memoryOfflineStore) Acknowledge(_ context.Context, _ string, id string) error {
	delete(s.claimed, id)
	return nil
}
func (s *memoryOfflineStore) Release(_ context.Context, _ string, id string) error {
	claim := s.claimed[id]
	delete(s.claimed, id)
	s.ready = append([]*offlineMessageClaim{claim}, s.ready...)
	return nil
}

func TestOfflineDeliveryKeepsUnacceptedMessageForReconnect(t *testing.T) {
	store := newMemoryOfflineStore("first", "second")
	accepted := make([]string, 0, 1)
	err := deliverOfflineMessages(context.Background(), store, "u-42", func(frame []byte) bool {
		if len(accepted) == 1 {
			return false
		}
		var message models.WebSocketMessage
		if unmarshalErr := json.Unmarshal(frame, &message); unmarshalErr != nil {
			t.Fatal(unmarshalErr)
		}
		payload, ok := message.Data.(map[string]any)
		if !ok {
			t.Fatalf("message payload type = %T", message.Data)
		}
		accepted = append(accepted, payload["content"].(string))
		return true
	})
	if err != errOfflineDeliveryBackpressure {
		t.Fatalf("delivery error = %v, want backpressure", err)
	}
	if got := len(store.claimed); got != 0 {
		t.Fatalf("in-flight claims = %d, want 0", got)
	}
	if got := len(store.ready); got != 1 || store.ready[0].Message.Content != "second" {
		t.Fatalf("ready queue after backpressure = %#v, want only second", store.ready)
	}
	if got := accepted; len(got) != 1 || got[0] != "first" {
		t.Fatalf("accepted = %#v, want [first]", got)
	}
}

func TestOfflineDeliveryPreservesFIFOWhenWriterAcceptsFrames(t *testing.T) {
	store := newMemoryOfflineStore("one", "two", "three")
	var accepted []string
	err := deliverOfflineMessages(context.Background(), store, "u-42", func(frame []byte) bool {
		var message models.WebSocketMessage
		if unmarshalErr := json.Unmarshal(frame, &message); unmarshalErr != nil {
			t.Fatal(unmarshalErr)
		}
		payload := message.Data.(map[string]any)
		accepted = append(accepted, payload["content"].(string))
		return true
	})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(store.ready), 0; got != want {
		t.Fatalf("ready = %d, want %d", got, want)
	}
	if got, want := len(store.claimed), 0; got != want {
		t.Fatalf("claimed = %d, want %d", got, want)
	}
	if got := "" + accepted[0] + accepted[1] + accepted[2]; got != "onetwothree" {
		t.Fatalf("delivery order = %q", got)
	}
}
