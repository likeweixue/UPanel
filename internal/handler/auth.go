package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"upanel/internal/service"
)

type AuthHandler struct {
	service *service.AuthService
}

func NewAuthHandler() *AuthHandler {
	return &AuthHandler{
		service: service.NewAuthService(),
	}
}

// 登录请求
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// 修改密码请求
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required"`
}

// 登录
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	token, err := h.service.Login(req.Username, req.Password)
	if err != nil {
		service.RecordLoginAttempt(c.ClientIP(), c.GetHeader("User-Agent"), req.Username, false, err.Error())
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	service.RecordLoginAttempt(c.ClientIP(), c.GetHeader("User-Agent"), req.Username, true, "")
	c.JSON(http.StatusOK, gin.H{
		"token":   token,
		"message": "登录成功",
	})
}

// 登出：把当前 JWT 加入黑名单，立即失效
func (h *AuthHandler) Logout(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
		_ = h.service.RevokeToken(strings.TrimSpace(parts[1]))
	}
	// 递增会话版本：当前进程内立即生效，并在重启后依然阻止旧令牌
	if err := h.service.RevokeAllSessions(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "登出失败，请重试"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "已登出"})
}

// 获取当前用户信息
func (h *AuthHandler) GetUserInfo(c *gin.Context) {
	username, exists := c.Get("username")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{"username": username, "role": "administrator"},
	})
}

// pickString 从 map 中按顺序取第一个非空字符串值
func pickString(m map[string]interface{}, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			if s, ok := v.(string); ok && s != "" {
				return s
			}
		}
	}
	return ""
}

// 修改密码
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	username, exists := c.Get("username")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}

	// 同时兼容 snake_case 与 camelCase 字段命名
	var raw map[string]interface{}
	if err := c.ShouldBindJSON(&raw); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req := ChangePasswordRequest{
		OldPassword: pickString(raw, "old_password", "oldPassword"),
		NewPassword: pickString(raw, "new_password", "newPassword"),
	}
	if req.OldPassword == "" || req.NewPassword == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "旧密码与新密码均不能为空"})
		return
	}

	err := h.service.ChangePassword(username.(string), req.OldPassword, req.NewPassword)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "密码修改成功"})
}
