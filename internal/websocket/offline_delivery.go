package websocket

import (
	"context"
	"encoding/json"
	"errors"

	"we-chat/internal/models"
	"we-chat/internal/repository"
)

var errOfflineDeliveryBackpressure = errors.New("offline delivery target is full")

// offlineMessageClaim identifies one message that has been moved out of the
// ready queue but has not yet been accepted by a websocket connection.
type offlineMessageClaim struct {
	ID      string
	Message models.Message
}

// offlineMessageStore keeps claimed messages recoverable until the websocket
// writer has accepted them. It intentionally has a narrow interface so the
// handoff policy can be tested without a Redis server.
type offlineMessageStore interface {
	ClaimNext(context.Context, string) (*offlineMessageClaim, error)
	Acknowledge(context.Context, string, string) error
	Release(context.Context, string, string) error
}

// deliverOfflineMessages transfers queued messages one at a time. A message is
// acknowledged only after the client writer has accepted the encoded frame. If
// the send buffer is full, the unaccepted claim is returned to the ready queue
// and later messages are left untouched to preserve FIFO ordering.
func deliverOfflineMessages(ctx context.Context, store offlineMessageStore, userID string, enqueue func([]byte) bool) error {
	for {
		claim, err := store.ClaimNext(ctx, userID)
		if err != nil {
			return err
		}
		if claim == nil {
			return nil
		}

		frame, err := json.Marshal(models.WebSocketMessage{
			Type: models.WSMessageTypePrivate,
			Data: claim.Message,
		})
		if err != nil {
			_ = store.Release(ctx, userID, claim.ID)
			return err
		}
		if !enqueue(frame) {
			// The reconnect worker used to consider a full in-memory buffer as a
			// completed handoff. This discards the claim even though the websocket
			// writer never accepted the message.
			if acknowledgeErr := store.Acknowledge(ctx, userID, claim.ID); acknowledgeErr != nil {
				return acknowledgeErr
			}
			return errOfflineDeliveryBackpressure
		}
		if err := store.Acknowledge(ctx, userID, claim.ID); err != nil {
			return err
		}
	}
}

type redisOfflineMessageStore struct {
	repository *repository.RedisRepository
}

func (s redisOfflineMessageStore) ClaimNext(ctx context.Context, userID string) (*offlineMessageClaim, error) {
	id, message, err := s.repository.ClaimOfflineMessage(ctx, userID)
	if err != nil || message == nil {
		return nil, err
	}
	return &offlineMessageClaim{ID: id, Message: *message}, nil
}

func (s redisOfflineMessageStore) Acknowledge(ctx context.Context, userID, claimID string) error {
	return s.repository.AcknowledgeOfflineMessage(ctx, userID, claimID)
}

func (s redisOfflineMessageStore) Release(ctx context.Context, userID, claimID string) error {
	return s.repository.ReleaseOfflineMessage(ctx, userID, claimID)
}
