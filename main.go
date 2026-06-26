package main

import (
	"log"
	"net/http"
	"strings"

	"image_go_api/internal/config"
	"image_go_api/internal/handler"
	"image_go_api/internal/middleware"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()

	r := gin.Default()
	r.Use(corsMiddleware(cfg.CORSOrigins))

	img := handler.NewImageHandler(cfg)
	auth := middleware.APIKeyAuth(cfg.APIKey)

	api := r.Group("/api/images")
	{
		api.POST("/upload", auth, img.Upload)
		api.GET("/health", img.Health)
		api.GET("/:subdir/:filename", img.Get)
		api.DELETE("/:subdir/:filename", auth, img.Delete)
	}

	log.Printf("image api server started on :%s", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}

func corsMiddleware(allowedOrigins []string) gin.HandlerFunc {
	allowAll := false
	allowedSet := make(map[string]struct{}, len(allowedOrigins))

	for _, origin := range allowedOrigins {
		trimmed := strings.TrimSpace(origin)
		if trimmed == "" {
			continue
		}
		if trimmed == "*" {
			allowAll = true
			break
		}
		allowedSet[trimmed] = struct{}{}
	}

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if allowAll {
			c.Header("Access-Control-Allow-Origin", "*")
		} else if origin != "" {
			if _, ok := allowedSet[origin]; ok {
				c.Header("Access-Control-Allow-Origin", origin)
				c.Header("Vary", "Origin")
			}
		}

		c.Header("Access-Control-Allow-Headers", "Content-Type, X-API-Key")
		c.Header("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
