package handlers

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"time"

	"we-chat/internal/config"
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

	// 检查文件类型（仅作初步参考，真实类型由内容魔数判定）
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

	// 保存文件
	src, err := file.Open()
	if err != nil {
		response.InternalError(c, "Failed to open file")
		return
	}
	defer src.Close()

	// 校验真实图片内容：读取头部魔数，拒绝伪装成图片的非图片文件。
	// 仅信任文件内容本身，而非客户端声明的 Content-Type 或扩展名。
	head := make([]byte, 512)
	n, err := io.ReadFull(src, head)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
		response.BadRequest(c, "Invalid image format")
		return
	}
	head = head[:n]
	if !isImageByMagic(head) {
		response.BadRequest(c, "Invalid image format")
		return
	}

	dst, err := os.Create(filePath)
	if err != nil {
		response.InternalError(c, "Failed to create file")
		return
	}
	defer dst.Close()

	// 先写入已读取的头部，再写入剩余内容
	if _, err := dst.Write(head); err != nil {
		response.BadRequest(c, "Invalid image format")
		return
	}
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

// magicHeaders 是受支持的图片格式的文件签名（魔数）。
// 匹配任一前缀即认为文件为对应类型的真实图片。
var magicHeaders = [][]byte{
	{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}, // PNG: \x89PNG\r\n\x1A\n
	{0xFF, 0xD8, 0xFF},                               // JPEG: SOI marker
	{0x47, 0x49, 0x46, 0x38, 0x37, 0x61},             // GIF87a
	{0x47, 0x49, 0x46, 0x38, 0x39, 0x61},             // GIF89a
}

// isImageByMagic 通过文件头部字节判断是否为受支持的真实图片。
// 仅依据文件内容本身判定，不信任客户端声明的 Content-Type 或扩展名，
// 从而拒绝伪装成 PNG/JPEG/GIF 的非图片文件。
func isImageByMagic(head []byte) bool {
	for _, magic := range magicHeaders {
		if bytes.HasPrefix(head, magic) {
			return true
		}
	}
	return false
}