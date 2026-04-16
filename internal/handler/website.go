package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"upanel/internal/service"
)

type WebsiteHandler struct {
	service *service.WebsiteService
}

func NewWebsiteHandler() (*WebsiteHandler, error) {
	s, err := service.NewWebsiteService()
	if err != nil {
		return nil, err
	}
	return &WebsiteHandler{service: s}, nil
}

func (h *WebsiteHandler) Close() error {
	return h.service.Close()
}

// 创建网站
func (h *WebsiteHandler) Create(c *gin.Context) {
	var req service.CreateWebsiteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	website, err := h.service.CreateWebsite(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"data": website})
}

// 获取网站列表
func (h *WebsiteHandler) List(c *gin.Context) {
	websites, err := h.service.ListWebsites()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"data": websites})
}

// 启动网站
func (h *WebsiteHandler) Start(c *gin.Context) {
	domain := c.Param("domain")
	
	if err := h.service.StartWebsite(domain); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"message": "网站已启动"})
}

// 停止网站
func (h *WebsiteHandler) Stop(c *gin.Context) {
	domain := c.Param("domain")
	
	if err := h.service.StopWebsite(domain); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"message": "网站已停止"})
}

// 删除网站
func (h *WebsiteHandler) Delete(c *gin.Context) {
	domain := c.Param("domain")
	
	if err := h.service.DeleteWebsite(domain); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"message": "网站已删除"})
}

// 获取网站URL
func (h *WebsiteHandler) GetURL(c *gin.Context) {
	domain := c.Param("domain")
	
	url, err := h.service.GetWebsiteURL(domain)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	// 直接返回字符串，不包装成对象
	c.String(http.StatusOK, url)
}
