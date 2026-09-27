package sharedgin

import (
	"crypto/subtle"
	"net/http"

	"github.com/gin-gonic/gin"
)

const (
	APIKeyHeader          = "X-API-Key"
	ErrorCodeUnauthorized = "unauthorized"
)

// RequireAPIKey rejects requests whose X-API-Key header doesn't match key.
// An empty key disables the check (local development only; main refuses to
// start in release mode without one).
func RequireAPIKey(key string) gin.HandlerFunc {
	if key == "" {
		return func(c *gin.Context) { c.Next() }
	}
	expected := []byte(key)
	return func(c *gin.Context) {
		provided := []byte(c.GetHeader(APIKeyHeader))
		if subtle.ConstantTimeCompare(provided, expected) != 1 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{
				Code:    ErrorCodeUnauthorized,
				Message: "missing or invalid " + APIKeyHeader + " header",
			})
			return
		}
		c.Next()
	}
}
