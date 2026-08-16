package handlers

import (
	"io"
	"os"
	"path/filepath"
	"time"

	"we-chat/internal/config"
	"we-chat/internal/filetype"
	"we-chat/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type UploadHandler struct{}

func NewUploadHandler() *UploadHandler {
	return &UploadHandler{}
}

func (h *UploadHandler) UploadFile(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		response.BadRequest(c, "No file uploaded")
		return
	}

	// 检查文件大小
	if file.Size > config.AppConfig.Upload.MaxSize {
		response.BadRequest(c, "File size exceeds limit")
		return
	}

	// 创建上传目录
	uploadPath := config.AppConfig.Upload.Path
	if err := os.MkdirAll(uploadPath, 0755); err != nil {
		response.InternalError(c, "Failed to create upload directory")
		return
	}

	// 生成唯一文件名
	ext := filepath.Ext(file.Filename)
	filename := uuid.New().String() + ext
	filePath := filepath.Join(uploadPath, filename)

	// 保存文件
	if err := c.SaveUploadedFile(file, filePath); err != nil {
		response.InternalError(c, "Failed to save file")
		return
	}

	fileURL := "/uploads/" + filename
	response.Success(c, gin.H{
		"url":      fileURL,
		"filename": file.Filename,
		"size":     file.Size,
	})
}

func (h *UploadHandler) UploadImage(c *gin.Context) {
	file, err := c.FormFile("image")
	if err != nil {
		response.BadRequest(c, "No image uploaded")
		return
	}

	// 检查文件类型
	contentType := file.Header.Get("Content-Type")
	if contentType != "image/jpeg" && contentType != "image/png" && contentType != "image/gif" {
		response.BadRequest(c, "Invalid image format")
		return
	}

	// 检查文件大小
	if file.Size > config.AppConfig.Upload.MaxSize {
		response.BadRequest(c, "Image size exceeds limit")
		return
	}

	// 创建图片目录
	imagePath := filepath.Join(config.AppConfig.Upload.Path, "images")
	if err := os.MkdirAll(imagePath, 0755); err != nil {
		response.InternalError(c, "Failed to create image directory")
		return
	}

	// 生成唯一文件名
	ext := filepath.Ext(file.Filename)
	filename := time.Now().Format("20060102") + "_" + uuid.New().String() + ext
	filePath := filepath.Join(imagePath, filename)

	// 保存前读取文件头并按实际内容校验，不能只相信客户端声明的 Content-Type。
	src, err := file.Open()
	if err != nil {
		response.InternalError(c, "Failed to open file")
		return
	}
	defer src.Close()

	actualType, err := filetype.DetectImageContentType(src)
	if err != nil {
		response.InternalError(c, "Failed to read image")
		return
	}
	if !filetype.AllowedImageType(actualType, config.AppConfig.Upload.AllowedImageTypes) {
		response.BadRequest(c, "Invalid image format")
		return
	}

	dst, err := os.Create(filePath)
	if err != nil {
		response.InternalError(c, "Failed to create file")
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		response.InternalError(c, "Failed to save file")
		return
	}

	imageURL := "/uploads/images/" + filename
	response.Success(c, gin.H{
		"url":      imageURL,
		"filename": file.Filename,
		"size":     file.Size,
	})
}
