package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"upanel/internal/service"
)

type AppsHandler struct {
	service *service.AppService
}

func NewAppsHandler() (*AppsHandler, error) {
	s, err := service.NewAppService()
	if err != nil {
		return nil, err
	}
	return &AppsHandler{service: s}, nil
}

func (h *AppsHandler) Close() error {
	return h.service.Close()
}

// 获取应用列表
func (h *AppsHandler) List(c *gin.Context) {
	apps := h.service.GetApps()
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": apps, "msg": "success"})
}

// 安装应用
func (h *AppsHandler) Install(c *gin.Context) {
	var req service.InstallAppRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "error": err.Error()})
		return
	}
	
	if err := h.service.InstallApp(&req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "安装成功"})
}
