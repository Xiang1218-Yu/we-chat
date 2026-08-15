package handlers

import (
	"context"

	"we-chat/internal/models"
	"we-chat/internal/repository"
	"we-chat/pkg/response"

	"github.com/gin-gonic/gin"
)

type RoomHandler struct {
	mongoRepo *repository.MongoDBRepository
	redisRepo *repository.RedisRepository
}

func NewRoomHandler(mongoRepo *repository.MongoDBRepository, redisRepo *repository.RedisRepository) *RoomHandler {
	return &RoomHandler{
		mongoRepo: mongoRepo,
		redisRepo: redisRepo,
	}
}

type CreateRoomRequest struct {
	Name        string        `json:"name" binding:"required"`
	Description string        `json:"description"`
	Type        models.RoomType `json:"type"`
}

func (h *RoomHandler) CreateRoom(c *gin.Context) {
	userID := c.GetString("user_id")

	var req CreateRoomRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if req.Type == "" {
		req.Type = models.RoomTypePublic
	}

	room := &models.Room{
		Name:        req.Name,
		Description: req.Description,
		Type:        req.Type,
		OwnerID:     userID,
		Members:     []string{userID},
	}

	ctx := context.Background()
	if err := h.mongoRepo.CreateRoom(ctx, room); err != nil {
		response.InternalError(c, "Failed to create room")
		return
	}

	response.Success(c, gin.H{
		"room_id": room.ID.Hex(),
		"name":    room.Name,
		"type":    room.Type,
	})
}

func (h *RoomHandler) GetRooms(c *gin.Context) {
	userID := c.GetString("user_id")

	ctx := context.Background()
	rooms, err := h.mongoRepo.GetUserRooms(ctx, userID)
	if err != nil {
		response.InternalError(c, "Failed to get rooms")
		return
	}

	result := make([]gin.H, 0)
	for _, room := range rooms {
		result = append(result, gin.H{
			"room_id":     room.ID.Hex(),
			"name":        room.Name,
			"description": room.Description,
			"type":        room.Type,
			"members":     len(room.Members),
		})
	}

	response.Success(c, result)
}

func (h *RoomHandler) GetRoomInfo(c *gin.Context) {
	roomID := c.Param("id")

	ctx := context.Background()
	room, err := h.mongoRepo.GetRoomByID(ctx, roomID)
	if err != nil {
		response.NotFound(c, "Room not found")
		return
	}

	response.Success(c, gin.H{
		"room_id":     room.ID.Hex(),
		"name":        room.Name,
		"description": room.Description,
		"type":        room.Type,
		"owner_id":    room.OwnerID,
		"members":     room.Members,
	})
}

func (h *RoomHandler) JoinRoom(c *gin.Context) {
	userID := c.GetString("user_id")
	roomID := c.Param("id")

	ctx := context.Background()
	if err := h.mongoRepo.AddMemberToRoom(ctx, roomID, userID); err != nil {
		response.InternalError(c, "Failed to join room")
		return
	}

	h.redisRepo.AddRoomMember(ctx, roomID, userID)

	response.Success(c, gin.H{
		"message": "Joined room successfully",
	})
}

func (h *RoomHandler) LeaveRoom(c *gin.Context) {
	userID := c.GetString("user_id")
	roomID := c.Param("id")

	ctx := context.Background()
	if err := h.mongoRepo.RemoveMemberFromRoom(ctx, roomID, userID); err != nil {
		response.InternalError(c, "Failed to leave room")
		return
	}

	h.redisRepo.RemoveRoomMember(ctx, roomID, userID)

	response.Success(c, gin.H{
		"message": "Left room successfully",
	})
}

func (h *RoomHandler) GetRoomMessages(c *gin.Context) {
	roomID := c.Param("id")

	ctx := context.Background()

	// 先从Redis缓存获取
	messages, err := h.redisRepo.GetCachedMessages(ctx, roomID, 50)
	if err == nil && len(messages) > 0 {
		response.Success(c, messages)
		return
	}

	// 从MongoDB获取
	messages, err = h.mongoRepo.GetRoomMessages(ctx, roomID, 50)
	if err != nil {
		response.InternalError(c, "Failed to get messages")
		return
	}

	response.Success(c, messages)
}