package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// User 用户模型
type User struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Username  string             `bson:"username" json:"username"`
	Email     string             `bson:"email" json:"email"`
	Password  string             `bson:"password" json:"-"`
	Nickname  string             `bson:"nickname" json:"nickname"`
	Avatar    string             `bson:"avatar" json:"avatar"`
	Status    UserStatus         `bson:"status" json:"status"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time          `bson:"updated_at" json:"updated_at"`
}

type UserStatus string

const (
	UserOnline  UserStatus = "online"
	UserOffline UserStatus = "offline"
	UserBusy    UserStatus = "busy"
)

// Message 消息模型
type Message struct {
	ID         primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	RoomID     string             `bson:"room_id" json:"room_id"`
	SenderID   string             `bson:"sender_id" json:"sender_id"`
	SenderName string             `bson:"sender_name" json:"sender_name"`
	ReceiverID string             `bson:"receiver_id,omitempty" json:"receiver_id,omitempty"`
	Content    string             `bson:"content" json:"content"`
	Type       MessageType        `bson:"type" json:"type"`
	MediaURL   string             `bson:"media_url,omitempty" json:"media_url,omitempty"`
	Read       bool               `bson:"read" json:"read"`
	ReadAt     *time.Time         `bson:"read_at,omitempty" json:"read_at,omitempty"`
	CreatedAt  time.Time          `bson:"created_at" json:"created_at"`
	Reactions  []Reaction         `bson:"reactions,omitempty" json:"reactions,omitempty"`
	Mentions   []string           `bson:"mentions,omitempty" json:"mentions,omitempty"`
}

type MessageType string

const (
	MessageTypeText  MessageType = "text"
	MessageTypeImage MessageType = "image"
	MessageTypeFile  MessageType = "file"
)

// Reaction 表情回复
type Reaction struct {
	UserID string `bson:"user_id" json:"user_id"`
	Emoji  string `bson:"emoji" json:"emoji"`
}

// Room 聊天室模型
type Room struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name        string             `bson:"name" json:"name"`
	Description string             `bson:"description" json:"description"`
	Type        RoomType           `bson:"type" json:"type"`
	OwnerID     string             `bson:"owner_id" json:"owner_id"`
	Members     []string           `bson:"members" json:"members"`
	CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time          `bson:"updated_at" json:"updated_at"`
}

type RoomType string

const (
	RoomTypePublic  RoomType = "public"
	RoomTypePrivate RoomType = "private"
)

// WebSocketMessage WebSocket消息结构
type WebSocketMessage struct {
	Type      string      `json:"type"`
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
}

// WebSocket消息类型
const (
	WSMessageTypeChat         = "chat"
	WSMessageTypePrivate      = "private"
	WSMessageTypeJoin         = "join"
	WSMessageTypeLeave        = "leave"
	WSMessageTypeTyping       = "typing"
	WSMessageTypeRead         = "read"
	WSMessageTypeReaction     = "reaction"
	WSMessageTypeOnlineUsers  = "online_users"
	WSMessageTypeHeartbeat    = "heartbeat"
	WSMessageTypeNotification = "notification"
)