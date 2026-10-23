package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// SecurityHeaders 为所有响应补充基础安全响应头。
// 面板是同源 SPA，不需要放宽 CORS；静态资源与 API 均由同一个服务提供。
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.Writer.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=()")
		h.Set("Cross-Origin-Opener-Policy", "same-origin")
		h.Set("X-Permitted-Cross-Domain-Policies", "none")

		if strings.HasPrefix(c.Request.URL.Path, "/api") || strings.Contains(c.Request.URL.Path, "/api/") {
			h.Set("Cache-Control", "no-store")
		}

		// 注意：Vite 构建产物只使用外部脚本与样式，因此不需要 'unsafe-eval'。
		// style-src 允许 inline，用于 Vue 动态主题变量与组件内联样式。
		h.Set("Content-Security-Policy",
			"default-src 'self'; "+
				"script-src 'self'; "+
				"style-src 'self' 'unsafe-inline'; "+
				"img-src 'self' data:; "+
				"font-src 'self' data:; "+
				"connect-src 'self'; "+
				"object-src 'none'; "+
				"base-uri 'self'; "+
				"form-action 'self'; "+
				"frame-ancestors 'none'")

		c.Next()
	}
}

// BodyLimit 限制请求体大小。除文件上传外，JSON 请求体最大 2MB，
// 避免超大请求耗尽内存。文件上传单独放宽到 2GB。
func BodyLimit() gin.HandlerFunc {
	const (
		defaultMax int64 = 2 << 20 // 2MB
		uploadMax  int64 = 2 << 30 // 2GB
	)
	return func(c *gin.Context) {
		max := defaultMax
		if strings.HasSuffix(c.Request.URL.Path, "/files/upload") {
			max = uploadMax
		}
		if c.Request.ContentLength > max {
			c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{
				"error": "请求体过大",
			})
			return
		}
		if c.Request.Body != nil {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, max)
		}
		c.Next()
	}
}

// NoCacheAPI 禁止缓存 API 响应（尤其是登录与凭据相关接口）
func NoCacheAPI() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Cache-Control", "no-store")
		c.Header("Pragma", "no-cache")
		c.Next()
	}
}
