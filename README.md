# Go 实时聊天室应用

一个基于 Go 语言开发的实时聊天室应用，支持 WebSocket 实时通信、用户认证、聊天室管理等功能。

## 技术栈

- **Web框架**: Gin
- **实时通信**: WebSocket (gorilla/websocket)
- **数据库**: MongoDB
- **缓存**: Redis
- **认证**: JWT (golang-jwt/jwt)

## 功能特性

### 用户系统
- ✅ 用户注册、登录
- ✅ JWT认证
- ✅ 用户头像和昵称
- ✅ 在线状态显示

### 聊天功能
- ✅ 公共聊天室
- ✅ 私聊功能
- ✅ 实时消息发送和接收
- ✅ 消息历史记录
- ✅ 消息已读状态

### 扩展功能
- ✅ 图片和文件发送
- ✅ 消息表情回复
- ✅ 用户@提醒
- ✅ 聊天室创建和管理

### 技术特性
- ✅ WebSocket连接管理
- ✅ 心跳检测和断线重连
- ✅ 消息持久化
- ✅ 离线消息推送

## 项目结构

```
we-chat/
├── cmd/
│   └── server/
│       └── main.go              # 应用入口
├── internal/
│   ├── config/                  # 配置管理
│   ├── handlers/                # HTTP处理器
│   │   ├── user.go             # 用户相关
│   │   ├── room.go             # 聊天室相关
│   │   └── upload.go           # 文件上传
│   ├── middleware/              # 中间件
│   │   ├── auth.go             # JWT认证
│   │   └── cors.go             # CORS
│   ├── models/                  # 数据模型
│   ├── repository/              # 数据访问层
│   │   ├── mongodb.go          # MongoDB操作
│   │   └── redis.go            # Redis操作
│   └── websocket/               # WebSocket管理
│       └── manager.go          # 连接管理器
├── pkg/                         # 公共包
│   ├── jwt/                    # JWT工具
│   └── response/               # 响应格式
├── static/
│   └── index.html              # 前端页面
├── uploads/                     # 上传文件目录
├── .env                         # 环境配置
├── go.mod                       # Go模块定义
└── README.md                    # 项目文档
```

## 快速开始

### 前置要求

- Go 1.21 或更高版本
- MongoDB 4.0+
- Redis 5.0+

### 安装依赖

```bash
go mod download
```

### 使用 Docker Compose 启动服务（推荐）

项目提供了 Docker Compose 配置文件，可以一键启动 MongoDB 和 Redis 服务：

1. **启动数据库服务**
```bash
docker-compose up -d
```

2. **查看服务状态**
```bash
docker-compose ps
```

3. **停止服务**
```bash
docker-compose down
```

4. **查看服务日志**
```bash
docker-compose logs -f
```

服务启动后：
- MongoDB 运行在 `localhost:27017`
- Redis 运行在 `localhost:6379`

### 手动启动服务（可选）

如果你已经安装了 MongoDB 和 Redis，可以手动启动：

**启动 MongoDB**
```bash
mongod --config /usr/local/etc/mongod.conf
# 或
brew services start mongodb-community
```

**启动 Redis**
```bash
redis-server
# 或
brew services start redis
```

### 配置环境

复制 `.env` 文件并根据需要修改配置：

```env
# 服务器配置
SERVER_PORT=8080
GIN_MODE=debug

# MongoDB配置
MONGODB_URI=mongodb://localhost:27017
MONGODB_DATABASE=wechat

# Redis配置
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
REDIS_DB=0

# JWT配置
JWT_SECRET=your-secret-key-change-in-production
JWT_EXPIRE_HOURS=24

# 文件上传配置
UPLOAD_PATH=./uploads
MAX_UPLOAD_SIZE=10485760
```

### 启动服务

```bash
# 创建上传目录
mkdir -p uploads/images

# 运行应用
go run cmd/server/main.go
```

### 访问应用

打开浏览器访问：http://localhost:8080

## API 文档

### 用户相关

#### 注册
```
POST /api/register
Content-Type: application/json

{
  "username": "test",
  "email": "test@example.com",
  "password": "password123",
  "nickname": "测试用户"
}
```

#### 登录
```
POST /api/login
Content-Type: application/json

{
  "username": "test",
  "password": "password123"
}
```

#### 获取用户信息
```
GET /api/profile
Authorization: Bearer {token}
```

#### 退出登录
```
POST /api/logout
Authorization: Bearer {token}
```

### 聊天室相关

#### 创建聊天室
```
POST /api/rooms
Authorization: Bearer {token}
Content-Type: application/json

{
  "name": "聊天室名称",
  "description": "描述",
  "type": "public"
}
```

#### 获取聊天室列表
```
GET /api/rooms
Authorization: Bearer {token}
```

#### 加入聊天室
```
POST /api/rooms/{room_id}/join
Authorization: Bearer {token}
```

#### 获取聊天记录
```
GET /api/rooms/{room_id}/messages
Authorization: Bearer {token}
```

### 文件上传

#### 上传图片
```
POST /api/upload/image
Authorization: Bearer {token}
Content-Type: multipart/form-data

file: [图片文件]
```

#### 上传文件
```
POST /api/upload/file
Authorization: Bearer {token}
Content-Type: multipart/form-data

file: [文件]
```

### WebSocket

#### 连接
```
ws://localhost:8080/ws?user_id={user_id}&username={username}
```

#### 消息格式

发送聊天消息：
```json
{
  "type": "chat",
  "data": {
    "room_id": "public",
    "content": "Hello, World!",
    "type": "text"
  },
  "timestamp": "2024-01-01T00:00:00Z"
}
```

发送私聊消息：
```json
{
  "type": "private",
  "data": {
    "receiver_id": "user_id",
    "content": "Hello!",
    "type": "text"
  },
  "timestamp": "2024-01-01T00:00:00Z"
}
```

加入聊天室：
```json
{
  "type": "join",
  "data": {
    "room_id": "room_id"
  },
  "timestamp": "2024-01-01T00:00:00Z"
}
```

发送正在输入状态：
```json
{
  "type": "typing",
  "data": {
    "room_id": "room_id"
  },
  "timestamp": "2024-01-01T00:00:00Z"
}
```

消息已读：
```json
{
  "type": "read",
  "data": {
    "message_id": "message_id"
  },
  "timestamp": "2024-01-01T00:00:00Z"
}
```

添加表情回复：
```json
{
  "type": "reaction",
  "data": {
    "message_id": "message_id",
    "user_id": "user_id",
    "emoji": "👍"
  },
  "timestamp": "2024-01-01T00:00:00Z"
}
```

## 核心功能说明

### WebSocket 连接管理

- 自动处理连接建立和断开
- 心跳检测机制（30秒间隔）
- 自动重连机制（3秒延迟）
- 连接池管理

### 消息持久化

- MongoDB 存储所有消息历史
- Redis 缓存最近100条消息
- 离线消息队列

### 在线状态管理

- Redis 存储在线用户列表
- 实时推送在线用户变化
- 自动检测用户离线

### 文件上传

- 支持图片上传（JPEG、PNG、GIF）
- 支持文件上传（最大10MB）
- 自动生成唯一文件名

## 部署建议

### 生产环境配置

1. 修改 `.env` 中的敏感配置
2. 使用强密码和安全的 JWT Secret
3. 配置 MongoDB 和 Redis 的访问权限
4. 启用 HTTPS

### Docker 部署

项目已提供 `docker-compose.yml` 配置文件。

**开发环境**

使用现有的配置文件启动数据库服务：
```bash
docker-compose up -d
```

**生产环境部署**

创建 Dockerfile 用于构建应用镜像：

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o main cmd/server/main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/main .
COPY --from=builder /app/static ./static
EXPOSE 8080
CMD ["./main"]
```

然后修改 `docker-compose.yml` 添加应用服务：

```yaml
version: '3.8'

services:
  app:
    build: .
    ports:
      - "8080:8080"
    depends_on:
      - mongodb
      - redis
    environment:
      - MONGODB_URI=mongodb://mongodb:27017
      - REDIS_ADDR=redis:6379
      - JWT_SECRET=your-production-secret
      - GIN_MODE=release
    volumes:
      - ./uploads:/app/uploads

  mongodb:
    image: mongo:latest
    ports:
      - "27017:27017"
    volumes:
      - mongodb_data:/data/db

  redis:
    image: redis:latest
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data

volumes:
  mongodb_data:
  redis_data:
```

启动所有服务：
```bash
docker-compose up -d
```

## 许可证

MIT License