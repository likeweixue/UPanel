package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"upanel/internal/service"
)

// SettingsHandler 系统设置接口
type SettingsHandler struct {
	service *service.SettingsService
}

// NewSettingsHandler 创建设置处理器
func NewSettingsHandler() *SettingsHandler {
	return &SettingsHandler{service: service.NewSettingsService()}
}

// GetAll 获取所有设置
func (h *SettingsHandler) GetAll(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"data": h.service.GetAll()})
}

// Update 更新设置
func (h *SettingsHandler) Update(c *gin.Context) {
	var patch service.Settings
	if err := c.ShouldBindJSON(&patch); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数有误: " + err.Error()})
		return
	}

	restartRequired, err := h.service.Update(&patch)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	message := "设置已更新"
	if restartRequired {
		message = "设置已保存。端口/监听地址/安全入口的变更需要重启面板后生效（up 3）"
	}
	c.JSON(http.StatusOK, gin.H{
		"message":         message,
		"data":            h.service.GetAll(),
		"restartRequired": restartRequired,
	})
}
