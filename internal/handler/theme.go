package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"upanel/internal/service"
)

type ThemeHandler struct {
	service *service.ThemeService
}

func NewThemeHandler() *ThemeHandler {
	return &ThemeHandler{service: service.NewThemeService()}
}

func (h *ThemeHandler) List(c *gin.Context) {
	c.Header("Cache-Control", "no-cache")
	c.JSON(http.StatusOK, gin.H{"data": h.service.List()})
}

func (h *ThemeHandler) CSS(c *gin.Context) {
	data, err := h.service.CSS(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "主题不存在或缺少 theme.css"})
		return
	}
	c.Header("Cache-Control", "no-cache")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Data(http.StatusOK, "text/css; charset=utf-8", data)
}

func (h *ThemeHandler) Asset(c *gin.Context) {
	file, err := h.service.Asset(c.Param("id"), c.Param("filepath"))
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}
	c.Header("Cache-Control", "public, max-age=3600")
	c.File(file)
}
