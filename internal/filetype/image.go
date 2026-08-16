package filetype

import (
	"io"
	"net/http"
)

// DetectImageContentType identifies an image from its bytes instead of a
// caller-controlled multipart header. The stream is reset before returning so
// the caller can persist the complete payload.
func DetectImageContentType(src io.ReadSeeker) (string, error) {
	header := make([]byte, 512)
	n, err := src.Read(header)
	if err != nil && err != io.EOF {
		return "", err
	}
	if _, err := src.Seek(0, io.SeekStart); err != nil {
		return "", err
	}
	return http.DetectContentType(header[:n]), nil
}
