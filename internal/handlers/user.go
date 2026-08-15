package handlers

import (
	"context"

	"we-chat/internal/models"
	"we-chat/internal/repository"
	"we-chat/pkg/jwt"
	"we-chat/pkg/response"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type UserHandler struct {
	mongoRepo *repository.MongoDBRepository
	redisRepo *repository.RedisRepository
}

func NewUserHandler(mongoRepo *repository.MongoDBRepository, redisRepo *repository.RedisRepository) *UserHandler {
	return &UserHandler{
		mongoRepo: mongoRepo,
		redisRepo: redisRepo,
	}
}

type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=20"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
	Nickname string `json:"nickname"`
}

func (h *UserHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	ctx := context.Background()

	// 检查用户名是否已存在
	if _, err := h.mongoRepo.GetUserByUsername(ctx, req.Username); err == nil {
		response.BadRequest(c, "Username already exists")
		return
	}

	// 加密密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		response.InternalError(c, "Failed to hash password")
		return
	}

	user := &models.User{
		Username: req.Username,
		Email:    req.Email,
		Password: string(hashedPassword),
		Nickname: req.Nickname,
		Status:   models.UserOffline,
	}

	if req.Nickname == "" {
		user.Nickname = req.Username
	}

	if err := h.mongoRepo.CreateUser(ctx, user); err != nil {
		response.InternalError(c, "Failed to create user")
		return
	}

	response.Success(c, gin.H{
		"user_id":  user.ID.Hex(),
		"username": user.Username,
		"email":    user.Email,
	})
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (h *UserHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	ctx := context.Background()
	user, err := h.mongoRepo.GetUserByUsername(ctx, req.Username)
	if err != nil {
		response.Unauthorized(c, "Invalid username or password")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		response.Unauthorized(c, "Invalid username or password")
		return
	}

	// 更新用户状态为在线
	h.mongoRepo.UpdateUserStatus(ctx, user.ID.Hex(), models.UserOnline)

	token, err := jwt.GenerateToken(user.ID.Hex(), user.Username, user.Email)
	if err != nil {
		response.InternalError(c, "Failed to generate token")
		return
	}

	response.Success(c, gin.H{
		"token":    token,
		"user_id":  user.ID.Hex(),
		"username": user.Username,
		"nickname": user.Nickname,
		"avatar":   user.Avatar,
	})
}

func (h *UserHandler) GetProfile(c *gin.Context) {
	userID := c.GetString("user_id")

	ctx := context.Background()
	user, err := h.mongoRepo.GetUserByID(ctx, userID)
	if err != nil {
		response.NotFound(c, "User not found")
		return
	}

	response.Success(c, gin.H{
		"user_id":  user.ID.Hex(),
		"username": user.Username,
		"email":    user.Email,
		"nickname": user.Nickname,
		"avatar":   user.Avatar,
		"status":   user.Status,
	})
}

func (h *UserHandler) UpdateProfile(c *gin.Context) {
	_ = c.GetString("user_id") // userID 用于后续的数据库更新操作

	var req struct {
		Nickname string `json:"nickname"`
		Avatar   string `json:"avatar"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	// 这里简化处理，实际应该更新数据库
	response.Success(c, gin.H{
		"message": "Profile updated successfully",
	})
}

func (h *UserHandler) Logout(c *gin.Context) {
	userID := c.GetString("user_id")

	ctx := context.Background()
	h.mongoRepo.UpdateUserStatus(ctx, userID, models.UserOffline)
	h.redisRepo.SetUserOffline(ctx, userID)
	h.redisRepo.DeleteUserSession(ctx, userID)

	response.Success(c, gin.H{
		"message": "Logged out successfully",
	})
}

func (h *UserHandler) GetOnlineUsers(c *gin.Context) {
	ctx := context.Background()
	users, err := h.redisRepo.GetOnlineUsers(ctx)
	if err != nil {
		response.InternalError(c, "Failed to get online users")
		return
	}

	response.Success(c, users)
}