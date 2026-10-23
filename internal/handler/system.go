package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"upanel/internal/service"
)

type SystemHandler struct {
	service *service.SystemService
	err     error
}

func NewSystemHandler() (*SystemHandler, error) {
	s, err := service.NewSystemService()
	return &SystemHandler{service: s, err: err}, nil
}

func (h *SystemHandler) Close() error {
	if h.service != nil {
		return h.service.Close()
	}
	return nil
}

// 获取系统信息
func (h *SystemHandler) GetInfo(c *gin.Context) {
	if h.err != nil || h.service == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "系统信息服务不可用"})
		return
	}
	info, err := h.service.GetSystemInfo()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": info})
}

// SSH 状态（只读）
func (h *SystemHandler) SSH(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"data": service.GetSSHStatus()})
}

// 防火墙状态（只读）
func (h *SystemHandler) Firewall(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"data": service.GetFirewallStatus()})
}
