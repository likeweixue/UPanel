package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"upanel/internal/service"
)

type EmailHandler struct{}

func NewEmailHandler() *EmailHandler { return &EmailHandler{} }

// Get 读取 SMTP 配置（不含密码）
func (h *EmailHandler) Get(c *gin.Context) {
	cfg, err := service.GetEmailConfigView()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": cfg})
}

// Update 保存 SMTP 配置
func (h *EmailHandler) Update(c *gin.Context) {
	var req service.EmailConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数有误: " + err.Error()})
		return
	}
	cfg, err := service.UpdateEmailConfig(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": cfg, "message": "邮件配置已保存"})
}

// Test 发送测试邮件
func (h *EmailHandler) Test(c *gin.Context) {
	var req service.EmailConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数有误: " + err.Error()})
		return
	}
	// 允许只传空对象，使用已保存配置
	if req.Host == "" && req.From == "" && len(req.To) == 0 {
		if err := service.SendTestEmail(nil); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "测试邮件已发送"})
		return
	}
	if err := service.SendTestEmail(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "测试邮件已发送"})
}
