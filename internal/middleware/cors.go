package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// CORS returns a middleware that handles Cross-Origin Resource Sharing for
// credentialed requests.
//
// Browsers forbid the wildcard "*" as Access-Control-Allow-Origin whenever a
// request carries credentials (cookies, Authorization headers). Such a
// response is rejected in both the preflight and the actual request. To accept
// credentialed cross-site requests we must instead echo the request's exact
// Origin. Because the allowed origin now varies per request, we also set
// Vary: Origin so shared caches do not serve one origin's response to another.
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		// For credentialed requests browsers reject "*"; echo the request
		// Origin instead. Same-origin requests send no Origin header, in
		// which case CORS headers are unnecessary.
		if origin != "" {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
			// Append (don't clobber) so any existing Vary value is preserved.
			c.Writer.Header().Add("Vary", "Origin")
		}

		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		// Preflight request: answer the OPTIONS check without invoking the
		// downstream handler.
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
