package repository

import (
	"context"
	"encoding/json"
	"time"

	"we-chat/internal/config"
	"we-chat/internal/models"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
)

type RedisRepository struct {
	client *redis.Client
}

func NewRedisRepository() *RedisRepository {
	rdb := redis.NewClient(&redis.Options{
		Addr:     config.AppConfig.Redis.Addr,
		Password: config.AppConfig.Redis.Password,
		DB:       config.AppConfig.Redis.DB,
	})

	return &RedisRepository{client: rdb}
}

// 在线用户管理
func (r *RedisRepository) SetUserOnline(ctx context.Context, userID string) error {
	key := "online_users:" + userID
	return r.client.Set(ctx, key, time.Now().Unix(), 24*time.Hour).Err()
}

func (r *RedisRepository) SetUserOffline(ctx context.Context, userID string) error {
	key := "online_users:" + userID
	return r.client.Del(ctx, key).Err()
}

func (r *RedisRepository) IsUserOnline(ctx context.Context, userID string) (bool, error) {
	key := "online_users:" + userID
	exists, err := r.client.Exists(ctx, key).Result()
	return exists > 0, err
}

func (r *RedisRepository) GetOnlineUsers(ctx context.Context) ([]string, error) {
	pattern := "online_users:*"
	keys, err := r.client.Keys(ctx, pattern).Result()
	if err != nil {
		return nil, err
	}

	users := make([]string, 0, len(keys))
	for _, key := range keys {
		users = append(users, key[13:]) // Remove "online_users:" prefix
	}
	return users, nil
}

// 消息缓存
func (r *RedisRepository) CacheMessage(ctx context.Context, roomID string, message *models.Message) error {
	key := "messages:" + roomID
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}

	// 使用LPUSH添加到列表头部
	if err := r.client.LPush(ctx, key, data).Err(); err != nil {
		return err
	}

	// 保持列表长度为100
	r.client.LTrim(ctx, key, 0, 99)
	return nil
}

func (r *RedisRepository) GetCachedMessages(ctx context.Context, roomID string, count int64) ([]models.Message, error) {
	key := "messages:" + roomID
	data, err := r.client.LRange(ctx, key, 0, count-1).Result()
	if err != nil {
		return nil, err
	}

	messages := make([]models.Message, 0, len(data))
	for _, item := range data {
		var msg models.Message
		if err := json.Unmarshal([]byte(item), &msg); err != nil {
			continue
		}
		messages = append(messages, msg)
	}

	return messages, nil
}

// 离线消息队列
func (r *RedisRepository) AddOfflineMessage(ctx context.Context, userID string, message *models.Message) error {
	key := "offline_messages:" + userID
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}

	return r.client.RPush(ctx, key, data).Err()
}

func (r *RedisRepository) GetOfflineMessages(ctx context.Context, userID string) ([]models.Message, error) {
	key := "offline_messages:" + userID
	data, err := r.client.LRange(ctx, key, 0, -1).Result()
	if err != nil {
		return nil, err
	}

	messages := make([]models.Message, 0, len(data))
	for _, item := range data {
		var msg models.Message
		if err := json.Unmarshal([]byte(item), &msg); err != nil {
			continue
		}
		messages = append(messages, msg)
	}

	// 清空离线消息
	r.client.Del(ctx, key)

	return messages, nil
}

// 用户会话管理
func (r *RedisRepository) SetUserSession(ctx context.Context, userID, token string, expiration time.Duration) error {
	key := "session:" + userID
	return r.client.Set(ctx, key, token, expiration).Err()
}

func (r *RedisRepository) GetUserSession(ctx context.Context, userID string) (string, error) {
	key := "session:" + userID
	return r.client.Get(ctx, key).Result()
}

func (r *RedisRepository) DeleteUserSession(ctx context.Context, userID string) error {
	key := "session:" + userID
	return r.client.Del(ctx, key).Err()
}

// 聊天室成员管理
func (r *RedisRepository) AddRoomMember(ctx context.Context, roomID, userID string) error {
	key := "room_members:" + roomID
	return r.client.SAdd(ctx, key, userID).Err()
}

func (r *RedisRepository) RemoveRoomMember(ctx context.Context, roomID, userID string) error {
	key := "room_members:" + roomID
	return r.client.SRem(ctx, key, userID).Err()
}

func (r *RedisRepository) GetRoomMembers(ctx context.Context, roomID string) ([]string, error) {
	key := "room_members:" + roomID
	return r.client.SMembers(ctx, key).Result()
}

// 心跳检测
func (r *RedisRepository) SetHeartbeat(ctx context.Context, userID string) error {
	key := "heartbeat:" + userID
	return r.client.Set(ctx, key, time.Now().Unix(), 30*time.Second).Err()
}

func (r *RedisRepository) CheckHeartbeat(ctx context.Context, userID string) (bool, error) {
	key := "heartbeat:" + userID
	exists, err := r.client.Exists(ctx, key).Result()
	return exists > 0, err
}

func (r *RedisRepository) Close() error {
	return r.client.Close()
}

// ClaimOfflineMessage removes the oldest queued offline message and stores it in
// a short-lived claim key until the websocket writer acknowledges it. The claim
// prevents a full client-side send buffer from silently discarding the message.
func (r *RedisRepository) ClaimOfflineMessage(ctx context.Context, userID string) (string, *models.Message, error) {
	queueKey := "offline_messages:" + userID
	raw, err := r.client.LPop(ctx, queueKey).Result()
	if err == redis.Nil {
		return "", nil, nil
	}
	if err != nil {
		return "", nil, err
	}

	var message models.Message
	if err := json.Unmarshal([]byte(raw), &message); err != nil {
		// Keep malformed data observable instead of treating the queue as empty.
		r.client.LPush(ctx, queueKey, raw)
		return "", nil, err
	}

	claimID := uuid.NewString()
	claimKey := "offline_message_claim:" + userID + ":" + claimID
	if err := r.client.Set(ctx, claimKey, raw, 5*time.Minute).Err(); err != nil {
		r.client.LPush(ctx, queueKey, raw)
		return "", nil, err
	}
	return claimID, &message, nil
}

// AcknowledgeOfflineMessage permanently removes a claim after a websocket
// writer accepted its frame.
func (r *RedisRepository) AcknowledgeOfflineMessage(ctx context.Context, userID, claimID string) error {
	return r.client.Del(ctx, "offline_message_claim:"+userID+":"+claimID).Err()
}

// ReleaseOfflineMessage returns an unaccepted claim to the head of its queue,
// preserving the original ordering for the next reconnect attempt. Claims are
// taken from the head (LPop in ClaimOfflineMessage), so they must be returned
// to the head (LPush) — pushing to the tail would requeue the message behind
// newer entries and reorder delivery.
func (r *RedisRepository) ReleaseOfflineMessage(ctx context.Context, userID, claimID string) error {
	claimKey := "offline_message_claim:" + userID + ":" + claimID
	raw, err := r.client.Get(ctx, claimKey).Result()
	if err == redis.Nil {
		return nil
	}
	if err != nil {
		return err
	}
	pipe := r.client.TxPipeline()
	pipe.LPush(ctx, "offline_messages:"+userID, raw)
	pipe.Del(ctx, claimKey)
	_, err = pipe.Exec(ctx)
	return err
}
