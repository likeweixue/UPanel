package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"upanel/internal/service"
)

type DockerHandler struct {
	service *service.DockerService
}

func NewDockerHandler() *DockerHandler {
	return &DockerHandler{
		service: service.NewDockerService(),
	}
}

// 获取 Docker 配置
func (h *DockerHandler) GetConfig(c *gin.Context) {
	config, err := h.service.GetConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	mirrors := h.service.GetMirrors()
	
	c.JSON(http.StatusOK, gin.H{
		"data":    config,
		"mirrors": mirrors,
	})
}

// 更新 Docker 配置
func (h *DockerHandler) UpdateConfig(c *gin.Context) {
	var req service.DockerConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	if err := h.service.SaveConfig(&req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"message": "配置已保存，请重启 Docker 生效"})
}

// 重启 Docker
func (h *DockerHandler) RestartDocker(c *gin.Context) {
	if err := h.service.RestartDocker(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"message": "Docker 正在重启"})
}
