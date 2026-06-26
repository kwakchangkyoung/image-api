package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func APIKeyAuth(apiKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if apiKey == "" {
			c.Next()
			return
		}

		if c.GetHeader("X-API-Key") != apiKey {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "유효하지 않은 API Key 입니다.",
			})
			return
		}

		c.Next()
	}
}
