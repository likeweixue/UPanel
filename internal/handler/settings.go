package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"upanel/internal/service"
)

type SettingsHandler struct {
	service *service.SettingsService
}

func NewSettingsHandler() *SettingsHandler {
	return &SettingsHandler{
		service: service.NewSettingsService(),
	}
}

// 获取所有设置
func (h *SettingsHandler) GetAll(c *gin.Context) {
	settings := h.service.GetAll()
	c.JSON(http.StatusOK, gin.H{"data": settings})
}

// 更新设置
func (h *SettingsHandler) Update(c *gin.Context) {
	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// 处理各种设置
	if mirrors, ok := updates["registry_mirrors"].([]interface{}); ok {
		var strMirrors []string
		for _, m := range mirrors {
			if s, ok := m.(string); ok {
				strMirrors = append(strMirrors, s)
			}
		}
		h.service.SetRegistryMirrors(strMirrors)
	}
	
	if theme, ok := updates["theme"].(string); ok {
		h.service.SetTheme(theme)
	}
	
	if websitePath, ok := updates["website_path"].(string); ok {
		h.service.SetWebsitePath(websitePath)
	}
	
	if databasePath, ok := updates["database_path"].(string); ok {
		h.service.SetDatabasePath(databasePath)
	}
	
	c.JSON(http.StatusOK, gin.H{"message": "设置已更新"})
}
