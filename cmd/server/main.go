package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"we-chat/internal/config"
	"we-chat/internal/handlers"
	"we-chat/internal/middleware"
	"we-chat/internal/repository"
	"we-chat/internal/websocket"

	"github.com/gin-gonic/gin"
)

func main() {
	// 加载配置
	config.LoadConfig()

	// 初始化数据库
	mongoRepo := repository.NewMongoDBRepository()
	defer mongoRepo.Disconnect()

	redisRepo := repository.NewRedisRepository()
	defer redisRepo.Close()

	// 初始化WebSocket管理器
	wsManager := websocket.NewWebSocketManager(mongoRepo, redisRepo)
	go wsManager.Run()

	// 初始化Handler
	userHandler := handlers.NewUserHandler(mongoRepo, redisRepo)
	roomHandler := handlers.NewRoomHandler(mongoRepo, redisRepo)
	uploadHandler := handlers.NewUploadHandler()

	// 设置Gin
	if config.AppConfig.Server.GinMode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()

	// 中间件
	r.Use(middleware.CORS())

	// 静态文件
	r.Static("/uploads", config.AppConfig.Upload.Path)
	r.StaticFile("/", "./static/index.html")

	// 公开路由
	api := r.Group("/api")
	{
		api.POST("/register", userHandler.Register)
		api.POST("/login", userHandler.Login)
	}

	// WebSocket路由
	r.GET("/ws", wsManager.HandleWebSocket)

	// 需要认证的路由
	protected := r.Group("/api")
	protected.Use(middleware.AuthMiddleware())
	{
		// 用户相关
		protected.GET("/profile", userHandler.GetProfile)
		protected.PUT("/profile", userHandler.UpdateProfile)
		protected.POST("/logout", userHandler.Logout)
		protected.GET("/online-users", userHandler.GetOnlineUsers)

		// 聊天室相关
		protected.POST("/rooms", roomHandler.CreateRoom)
		protected.GET("/rooms", roomHandler.GetRooms)
		protected.GET("/rooms/:id", roomHandler.GetRoomInfo)
		protected.POST("/rooms/:id/join", roomHandler.JoinRoom)
		protected.POST("/rooms/:id/leave", roomHandler.LeaveRoom)
		protected.GET("/rooms/:id/messages", roomHandler.GetRoomMessages)

		// 文件上传
		protected.POST("/upload/file", uploadHandler.UploadFile)
		protected.POST("/upload/image", uploadHandler.UploadImage)
	}

	// 启动服务器
	go func() {
		if err := r.Run(":" + config.AppConfig.Server.Port); err != nil {
			log.Fatal("Failed to start server:", err)
		}
	}()

	log.Printf("Server started on port %s", config.AppConfig.Server.Port)

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
}