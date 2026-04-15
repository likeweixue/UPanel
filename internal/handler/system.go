package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"upanel/internal/service"
)

type SystemHandler struct {
	service *service.SystemService
}

func NewSystemHandler() (*SystemHandler, error) {
	s, err := service.NewSystemService()
	if err != nil {
		return nil, err
	}
	return &SystemHandler{service: s}, nil
}

func (h *SystemHandler) Close() error {
	return h.service.Close()
}

// 获取系统信息
func (h *SystemHandler) GetInfo(c *gin.Context) {
	info, err := h.service.GetSystemInfo()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": info})
}
