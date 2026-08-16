package handlers

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"os"
	"path/filepath"
	"testing"

	"we-chat/internal/config"

	"github.com/gin-gonic/gin"
)

func TestUploadImageRejectsSpoofedContentType(t *testing.T) {
	gin.SetMode(gin.TestMode)
	uploadDir := t.TempDir()
	config.AppConfig = &config.Config{Upload: config.UploadConfig{Path: uploadDir, MaxSize: 1024 * 1024}}

	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	part, err := writer.CreatePart(textprotoHeader("image", "avatar.png", "image/png"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write([]byte("<script>alert('not an image')</script>")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	router := gin.New()
	router.POST("/upload/image", NewUploadHandler().UploadImage)
	req := httptest.NewRequest(http.MethodPost, "/upload/image", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("spoofed image upload returned %d, want %d; body=%s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	entries, err := os.ReadDir(filepath.Join(uploadDir, "images"))
	if err == nil && len(entries) != 0 {
		t.Fatalf("spoofed upload was persisted: %d files", len(entries))
	}
}

func textprotoHeader(field, filename, contentType string) textproto.MIMEHeader {
	return textproto.MIMEHeader{
		"Content-Disposition": {`form-data; name="` + field + `"; filename="` + filename + `"`},
		"Content-Type":        {contentType},
	}
}
