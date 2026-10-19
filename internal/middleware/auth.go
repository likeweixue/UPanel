package middleware

import (
	"net/http"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"upanel/internal/service"
)

// 注意：这里不能在包初始化阶段创建 AuthService。
// 包级变量的初始化早于 main() 中的 config.LoadConfig()，
// 此时 config.Current() 返回的是默认配置，构造 AuthService 会把默认密码哈希
// 写回 config.json，从而覆盖用户设置的端口等配置。
// 因此改为懒加载，保证读取到的是已加载的配置。
var (
	authOnce    sync.Once
	authService *service.AuthService
)

func getAuthService() *service.AuthService {
	authOnce.Do(func() {
		authService = service.NewAuthService()
	})
	return authService
}

// JWT 认证中间件
func AuthMiddleware() gin.HandlerFunc {
	auth := getAuthService()
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
		username, err := auth.ValidateToken(token)
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
	auth := getAuthService()
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && parts[0] == "Bearer" {
				username, err := auth.ValidateToken(parts[1])
				if err == nil {
					c.Set("username", username)
				}
			}
		}
		c.Next()
	}
}
