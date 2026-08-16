package middleware

import "strings"

func appendVary(existing, value string) string {
	for _, item := range strings.Split(existing, ",") {
		if strings.EqualFold(strings.TrimSpace(item), value) {
			return existing
		}
	}
	if existing == "" {
		return value
	}
	return existing + ", " + value
}
