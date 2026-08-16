package filetype

// AllowedImageType reports whether an inspected MIME type is accepted by the
// upload policy. Keeping the policy separate makes validation reusable by all
// upload endpoints.
func AllowedImageType(contentType string, allowed []string) bool {
	for _, candidate := range allowed {
		if contentType == candidate {
			return true
		}
	}
	return false
}
