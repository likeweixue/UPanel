package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"upanel/internal/service"
)

var authService = service.NewAuthService()

// JWT 认证中间件
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从 Header 获取 token
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "未提供认证令牌"})
			c.Abort()
			return
		}
		
		// 解析 Bearer token
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "认证格式错误"})
			c.Abort()
			return
		}
		
		token := parts[1]
		
		// 验证 token
		username, err := authService.ValidateToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的认证令牌"})
			c.Abort()
			return
		}
		
		// 将用户名存入上下文
		c.Set("username", username)
		c.Next()
	}
}

// 可选认证（不强制登录，但如果有 token 则解析）
func OptionalAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && parts[0] == "Bearer" {
				username, err := authService.ValidateToken(parts[1])
				if err == nil {
					c.Set("username", username)
				}
			}
		}
		c.Next()
	}
}
