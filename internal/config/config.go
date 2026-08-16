package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Server  ServerConfig
	MongoDB MongoDBConfig
	Redis   RedisConfig
	JWT     JWTConfig
	Upload  UploadConfig
}

type ServerConfig struct {
	Port    string
	GinMode string
}

type MongoDBConfig struct {
	URI      string
	Database string
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

type JWTConfig struct {
	Secret      string
	ExpireHours int
}

type UploadConfig struct {
	Path              string
	MaxSize           int64
	AllowedImageTypes []string
}

var AppConfig *Config

func LoadConfig() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	AppConfig = &Config{
		Server: ServerConfig{
			Port:    getEnv("SERVER_PORT", "8080"),
			GinMode: getEnv("GIN_MODE", "debug"),
		},
		MongoDB: MongoDBConfig{
			URI:      getEnv("MONGODB_URI", "mongodb://localhost:27017"),
			Database: getEnv("MONGODB_DATABASE", "wechat"),
		},
		Redis: RedisConfig{
			Addr:     getEnv("REDIS_ADDR", "localhost:6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvAsInt("REDIS_DB", 0),
		},
		JWT: JWTConfig{
			Secret:      getEnv("JWT_SECRET", "your-secret-key"),
			ExpireHours: getEnvAsInt("JWT_EXPIRE_HOURS", 24),
		},
		Upload: UploadConfig{
			Path:              getEnv("UPLOAD_PATH", "./uploads"),
			MaxSize:           getEnvAsInt64("MAX_UPLOAD_SIZE", 10485760),
			AllowedImageTypes: []string{"image/jpeg", "image/png", "image/gif"},
		},
	}
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := getEnv(key, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultValue
}

func getEnvAsInt64(key string, defaultValue int64) int64 {
	valueStr := getEnv(key, "")
	if value, err := strconv.ParseInt(valueStr, 10, 64); err == nil {
		return value
	}
	return defaultValue
}
