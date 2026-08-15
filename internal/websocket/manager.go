package websocket

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"we-chat/internal/models"
	"we-chat/internal/repository"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Client struct {
	ID       string
	UserID   string
	Username string
	Conn     *websocket.Conn
	Send     chan []byte
	Rooms    map[string]bool
	mu       sync.Mutex
}

type WebSocketManager struct {
	connections *connectionRegistry
	register    chan *Client
	unregister  chan *Client
	broadcast   chan []byte
	mongoRepo   *repository.MongoDBRepository
	redisRepo   *repository.RedisRepository
}

func NewWebSocketManager(mongoRepo *repository.MongoDBRepository, redisRepo *repository.RedisRepository) *WebSocketManager {
	return &WebSocketManager{
		connections: newConnectionRegistry(),
		register:    make(chan *Client),
		unregister:  make(chan *Client),
		broadcast:   make(chan []byte, 100),
		mongoRepo:   mongoRepo,
		redisRepo:   redisRepo,
	}
}

func (m *WebSocketManager) Run() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case client := <-m.register:
			firstConnection := m.connections.Add(client)
			if firstConnection {
				m.redisRepo.SetUserOnline(context.Background(), client.UserID)
			}
			m.redisRepo.SetUserSession(context.Background(), client.UserID, client.ID, 24*time.Hour)
			m.notifyOnlineUsers()

		case client := <-m.unregister:
			removed, lastConnection := m.connections.Remove(client.ID)
			if removed {
				close(client.Send)
			}
			if lastConnection {
				m.redisRepo.SetUserOffline(context.Background(), client.UserID)
				m.redisRepo.DeleteUserSession(context.Background(), client.UserID)
			}
			m.notifyOnlineUsers()

		case message := <-m.broadcast:
			for _, client := range m.connections.All() {
				select {
				case client.Send <- message:
				default:
					if removed, lastConnection := m.connections.Remove(client.ID); removed {
						close(client.Send)
						if lastConnection {
							m.redisRepo.SetUserOffline(context.Background(), client.UserID)
							m.redisRepo.DeleteUserSession(context.Background(), client.UserID)
						}
					}
				}
			}

		case <-ticker.C:
			m.checkHeartbeats()
		}
	}
}

func (m *WebSocketManager) HandleWebSocket(c *gin.Context) {
	userID := c.Query("user_id")
	username := c.Query("username")

	if userID == "" || username == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id and username are required"})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("WebSocket upgrade error:", err)
		return
	}

	client := &Client{
		ID:       uuid.NewString(),
		UserID:   userID,
		Username: username,
		Conn:     conn,
		Send:     make(chan []byte, 256),
		Rooms:    make(map[string]bool),
	}

	m.register <- client

	// 发送离线消息
	go m.sendOfflineMessages(client)

	go client.writePump()
	go client.readPump(m)
}

func (c *Client) readPump(m *WebSocketManager) {
	defer func() {
		m.unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(5120)
	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		m.redisRepo.SetHeartbeat(context.Background(), c.UserID)
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket read error: %v", err)
			}
			break
		}

		m.handleMessage(c, message)
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			n := len(c.Send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.Send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (m *WebSocketManager) handleMessage(client *Client, data []byte) {
	var wsMsg models.WebSocketMessage
	if err := json.Unmarshal(data, &wsMsg); err != nil {
		log.Printf("Failed to unmarshal message: %v", err)
		return
	}

	ctx := context.Background()

	switch wsMsg.Type {
	case models.WSMessageTypeChat:
		m.handleChatMessage(ctx, client, wsMsg)
	case models.WSMessageTypePrivate:
		m.handlePrivateMessage(ctx, client, wsMsg)
	case models.WSMessageTypeJoin:
		m.handleJoinRoom(ctx, client, wsMsg)
	case models.WSMessageTypeLeave:
		m.handleLeaveRoom(ctx, client, wsMsg)
	case models.WSMessageTypeTyping:
		m.handleTyping(client, wsMsg)
	case models.WSMessageTypeRead:
		m.handleReadMessage(ctx, wsMsg)
	case models.WSMessageTypeReaction:
		m.handleReaction(ctx, wsMsg)
	case models.WSMessageTypeHeartbeat:
		m.redisRepo.SetHeartbeat(ctx, client.UserID)
	}
}

func (m *WebSocketManager) handleChatMessage(ctx context.Context, client *Client, wsMsg models.WebSocketMessage) {
	data, _ := json.Marshal(wsMsg.Data)
	var msg models.Message
	json.Unmarshal(data, &msg)

	msg.SenderID = client.UserID
	msg.SenderName = client.Username
	msg.Type = models.MessageTypeText

	if err := m.mongoRepo.SaveMessage(ctx, &msg); err != nil {
		log.Printf("Failed to save message: %v", err)
		return
	}

	m.redisRepo.CacheMessage(ctx, msg.RoomID, &msg)

	m.broadcastToRoom(msg.RoomID, &wsMsg)
}

func (m *WebSocketManager) handlePrivateMessage(ctx context.Context, client *Client, wsMsg models.WebSocketMessage) {
	data, _ := json.Marshal(wsMsg.Data)
	var msg models.Message
	json.Unmarshal(data, &msg)

	msg.SenderID = client.UserID
	msg.SenderName = client.Username
	msg.Type = models.MessageTypeText

	if err := m.mongoRepo.SaveMessage(ctx, &msg); err != nil {
		log.Printf("Failed to save private message: %v", err)
		return
	}

	m.sendToUser(msg.ReceiverID, &wsMsg)

	if isOnline, _ := m.redisRepo.IsUserOnline(ctx, msg.ReceiverID); !isOnline {
		m.redisRepo.AddOfflineMessage(ctx, msg.ReceiverID, &msg)
	}
}

func (m *WebSocketManager) handleJoinRoom(ctx context.Context, client *Client, wsMsg models.WebSocketMessage) {
	data, _ := json.Marshal(wsMsg.Data)
	var joinData struct {
		RoomID string `json:"room_id"`
	}
	json.Unmarshal(data, &joinData)

	client.mu.Lock()
	client.Rooms[joinData.RoomID] = true
	client.mu.Unlock()

	m.redisRepo.AddRoomMember(ctx, joinData.RoomID, client.UserID)

	notification := models.WebSocketMessage{
		Type: models.WSMessageTypeNotification,
		Data: map[string]string{
			"user_id":  client.UserID,
			"username": client.Username,
			"room_id":  joinData.RoomID,
			"action":   "joined",
		},
		Timestamp: time.Now(),
	}
	m.broadcastToRoom(joinData.RoomID, &notification)
}

func (m *WebSocketManager) handleLeaveRoom(ctx context.Context, client *Client, wsMsg models.WebSocketMessage) {
	data, _ := json.Marshal(wsMsg.Data)
	var leaveData struct {
		RoomID string `json:"room_id"`
	}
	json.Unmarshal(data, &leaveData)

	client.mu.Lock()
	delete(client.Rooms, leaveData.RoomID)
	client.mu.Unlock()

	m.redisRepo.RemoveRoomMember(ctx, leaveData.RoomID, client.UserID)

	notification := models.WebSocketMessage{
		Type: models.WSMessageTypeNotification,
		Data: map[string]string{
			"user_id":  client.UserID,
			"username": client.Username,
			"room_id":  leaveData.RoomID,
			"action":   "left",
		},
		Timestamp: time.Now(),
	}
	m.broadcastToRoom(leaveData.RoomID, &notification)
}

func (m *WebSocketManager) handleTyping(client *Client, wsMsg models.WebSocketMessage) {
	data, _ := json.Marshal(wsMsg.Data)
	var typingData struct {
		RoomID string `json:"room_id"`
	}
	json.Unmarshal(data, &typingData)

	wsMsg.Data = map[string]string{
		"user_id":  client.UserID,
		"username": client.Username,
		"room_id":  typingData.RoomID,
	}

	m.broadcastToRoom(typingData.RoomID, &wsMsg)
}

func (m *WebSocketManager) handleReadMessage(ctx context.Context, wsMsg models.WebSocketMessage) {
	data, _ := json.Marshal(wsMsg.Data)
	var readData struct {
		MessageID string `json:"message_id"`
	}
	json.Unmarshal(data, &readData)

	m.mongoRepo.MarkMessageAsRead(ctx, readData.MessageID)
}

func (m *WebSocketManager) handleReaction(ctx context.Context, wsMsg models.WebSocketMessage) {
	data, _ := json.Marshal(wsMsg.Data)
	var reactionData struct {
		MessageID string `json:"message_id"`
		UserID    string `json:"user_id"`
		Emoji     string `json:"emoji"`
	}
	json.Unmarshal(data, &reactionData)

	reaction := models.Reaction{
		UserID: reactionData.UserID,
		Emoji:  reactionData.Emoji,
	}

	m.mongoRepo.AddReaction(ctx, reactionData.MessageID, reaction)
}

func (m *WebSocketManager) broadcastToRoom(roomID string, message *models.WebSocketMessage) {
	data, _ := json.Marshal(message)

	for _, client := range m.connections.All() {
		client.mu.Lock()
		_, joined := client.Rooms[roomID]
		client.mu.Unlock()
		if joined {
			select {
			case client.Send <- data:
			default:
			}
		}
	}
}

func (m *WebSocketManager) sendToUser(userID string, message *models.WebSocketMessage) {
	data, _ := json.Marshal(message)

	for _, client := range m.connections.ForUser(userID) {
		select {
		case client.Send <- data:
		default:
		}
	}
}

func (m *WebSocketManager) sendOfflineMessages(client *Client) {
	ctx := context.Background()
	messages, err := m.redisRepo.GetOfflineMessages(ctx, client.UserID)
	if err != nil {
		return
	}

	for _, msg := range messages {
		wsMsg := models.WebSocketMessage{
			Type:      models.WSMessageTypePrivate,
			Data:      msg,
			Timestamp: time.Now(),
		}
		data, _ := json.Marshal(wsMsg)
		client.Send <- data
	}
}

func (m *WebSocketManager) notifyOnlineUsers() {
	ctx := context.Background()
	users, _ := m.redisRepo.GetOnlineUsers(ctx)

	message := models.WebSocketMessage{
		Type:      models.WSMessageTypeOnlineUsers,
		Data:      users,
		Timestamp: time.Now(),
	}

	data, _ := json.Marshal(message)
	m.broadcast <- data
}

func (m *WebSocketManager) checkHeartbeats() {
	ctx := context.Background()
	for _, client := range m.connections.All() {
		if ok, _ := m.redisRepo.CheckHeartbeat(ctx, client.UserID); !ok {
			m.unregister <- client
		}
	}
}

func (m *WebSocketManager) GetOnlineUsers() []string {
	return m.connections.Usernames()
}
