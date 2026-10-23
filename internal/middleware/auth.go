package middleware

import (
	"net/http"
	"strings"
	"sync"
	"time"

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

// AuthMiddleware JWT 认证中间件
func AuthMiddleware() gin.HandlerFunc {
	auth := getAuthService()
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			unauthorized(c, "未提供认证令牌")
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
			unauthorized(c, "认证格式错误")
			return
		}

		username, err := auth.ValidateToken(strings.TrimSpace(parts[1]))
		if err != nil {
			unauthorized(c, "登录已失效，请重新登录")
			return
		}

		c.Set("username", username)
		c.Next()
	}
}

func unauthorized(c *gin.Context, message string) {
	c.Header("WWW-Authenticate", `Bearer realm="upanel"`)
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": message})
}

// OptionalAuthMiddleware 可选认证（不强制登录，但如果有 token 则解析）
func OptionalAuthMiddleware() gin.HandlerFunc {
	auth := getAuthService()
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
				username, err := auth.ValidateToken(strings.TrimSpace(parts[1]))
				if err == nil {
					c.Set("username", username)
				}
			}
		}
		c.Next()
	}
}

// CurrentUsername 从上下文读取当前管理员用户名
func CurrentUsername(c *gin.Context) string {
	if v, ok := c.Get("username"); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// 登录失败限流：同一 IP 10 分钟内最多 8 次失败，超过后锁定 15 分钟。
// 这是一道针对公网面板暴力破解的兜底防线。
const (
	loginWindow      = 10 * time.Minute
	loginMaxFailures = 8
	loginBlockFor    = 15 * time.Minute
	loginGlobalMax   = 200
)

type loginBucket struct {
	failures     int
	firstFailure time.Time
	blockedUntil time.Time
}

type loginLimiter struct {
	mu        sync.Mutex
	buckets   map[string]*loginBucket
	global    int
	globalSet time.Time
}

var limiter = &loginLimiter{buckets: make(map[string]*loginBucket)}

// LoginRateLimit 返回登录接口限流中间件
func LoginRateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		if retry, blocked := limiter.blocked(ip); blocked {
			c.Header("Retry-After", retry)
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "登录尝试过于频繁，请稍后再试",
			})
			return
		}

		c.Next()

		switch c.Writer.Status() {
		case http.StatusOK:
			limiter.reset(ip)
		case http.StatusUnauthorized, http.StatusBadRequest:
			limiter.fail(ip)
		}
	}
}

func (l *loginLimiter) blocked(ip string) (string, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	l.pruneLocked(now)

	if now.Sub(l.globalSet) > loginWindow {
		l.global = 0
		l.globalSet = now
	}
	if l.global >= loginGlobalMax {
		return "60", true
	}

	b, ok := l.buckets[ip]
	if !ok {
		return "", false
	}
	if now.Before(b.blockedUntil) {
		secs := int(time.Until(b.blockedUntil).Seconds())
		if secs < 1 {
			secs = 1
		}
		return itoa(secs), true
	}
	return "", false
}

func (l *loginLimiter) fail(ip string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()

	if now.Sub(l.globalSet) > loginWindow {
		l.global = 0
		l.globalSet = now
	}
	l.global++

	b, ok := l.buckets[ip]
	if !ok {
		b = &loginBucket{firstFailure: now}
		l.buckets[ip] = b
	}
	if now.Sub(b.firstFailure) > loginWindow {
		b.failures = 0
		b.firstFailure = now
		b.blockedUntil = time.Time{}
	}
	b.failures++
	if b.failures >= loginMaxFailures {
		b.blockedUntil = now.Add(loginBlockFor)
		b.failures = 0
		b.firstFailure = now
	}
}

func (l *loginLimiter) reset(ip string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.buckets, ip)
}

func (l *loginLimiter) pruneLocked(now time.Time) {
	if len(l.buckets) < 5000 {
		return
	}
	for ip, b := range l.buckets {
		if now.After(b.blockedUntil) && now.Sub(b.firstFailure) > loginWindow {
			delete(l.buckets, ip)
		}
	}
}

func itoa(n int) string {
	if n <= 0 {
		return "1"
	}
	var buf [12]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
