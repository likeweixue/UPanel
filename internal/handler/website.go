package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"upanel/internal/config"
	"upanel/internal/service"
)

// WebsiteHandler 网站管理接口
type WebsiteHandler struct {
	service *service.WebsiteService
	err     error // Docker 不可用等原因导致的初始化失败
}

// NewWebsiteHandler 创建网站处理器。
// 即使 Docker 暂时不可用也不返回错误，路由照常注册，
// 请求时再给出明确的错误提示，避免出现难以理解的 404。
func NewWebsiteHandler() (*WebsiteHandler, error) {
	s, err := service.NewWebsiteService()
	return &WebsiteHandler{service: s, err: err}, nil
}

func (h *WebsiteHandler) Close() error {
	if h.service != nil {
		return h.service.Close()
	}
	return nil
}

// unavailable 判断服务是否可用，不可用时直接响应错误
func (h *WebsiteHandler) unavailable(c *gin.Context) bool {
	if h.err != nil || h.service == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Docker 未连接，无法执行该操作: " + h.err.Error()})
		return true
	}
	return false
}

// Create 创建网站
func (h *WebsiteHandler) Create(c *gin.Context) {
	if h.unavailable(c) {
		return
	}
	var req service.CreateWebsiteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数有误: " + err.Error()})
		return
	}
	if req.Domain == "" {
		req.Domain = c.PostForm("domain")
	}

	website, err := h.service.CreateWebsite(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": website, "message": "网站创建成功"})
}

// List 网站列表
func (h *WebsiteHandler) List(c *gin.Context) {
	if h.unavailable(c) {
		return
	}
	websites, err := h.service.ListWebsites()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if websites == nil {
		websites = []service.Website{}
	}
	c.JSON(http.StatusOK, gin.H{"data": websites})
}

// Get 网站详情
func (h *WebsiteHandler) Get(c *gin.Context) {
	if h.unavailable(c) {
		return
	}
	domain := strings.TrimSpace(c.Param("domain"))
	w, err := h.service.GetWebsite(domain)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": w})
}

// Update 更新网站
func (h *WebsiteHandler) Update(c *gin.Context) {
	if h.unavailable(c) {
		return
	}
	domain := strings.TrimSpace(c.Param("domain"))
	var patch service.Website
	if err := c.ShouldBindJSON(&patch); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数有误: " + err.Error()})
		return
	}
	w, err := h.service.UpdateWebsite(domain, &patch)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": w, "message": "网站已更新"})
}

// Start 启动
func (h *WebsiteHandler) Start(c *gin.Context) {
	if h.unavailable(c) {
		return
	}
	domain := strings.TrimSpace(c.Param("domain"))
	if err := h.service.StartWebsite(domain); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "网站已启动"})
}

// Stop 停止
func (h *WebsiteHandler) Stop(c *gin.Context) {
	if h.unavailable(c) {
		return
	}
	domain := strings.TrimSpace(c.Param("domain"))
	if err := h.service.StopWebsite(domain); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "网站已停止"})
}

// Restart 重启
func (h *WebsiteHandler) Restart(c *gin.Context) {
	if h.unavailable(c) {
		return
	}
	domain := strings.TrimSpace(c.Param("domain"))
	if err := h.service.StopWebsite(domain); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err := h.service.StartWebsite(domain); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "网站已重启"})
}

// Delete 删除
func (h *WebsiteHandler) Delete(c *gin.Context) {
	if h.unavailable(c) {
		return
	}
	domain := strings.TrimSpace(c.Param("domain"))
	keepFiles := c.Query("keep_files") == "1" || c.Query("keepFiles") == "true"
	if err := h.service.DeleteWebsite(domain, keepFiles); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "网站已删除"})
}

// ApplySSL 部署证书
func (h *WebsiteHandler) ApplySSL(c *gin.Context) {
	if h.unavailable(c) {
		return
	}
	domain := strings.TrimSpace(c.Param("domain"))
	var req struct {
		Fullchain string `json:"fullchain"`
		Privkey   string `json:"privkey"`
		ForceSSL  bool   `json:"forceSSL"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数有误: " + err.Error()})
		return
	}
	if strings.TrimSpace(req.Fullchain) == "" || strings.TrimSpace(req.Privkey) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "证书内容和私钥不能为空"})
		return
	}
	if err := h.service.ApplySSL(domain, req.Fullchain, req.Privkey, req.ForceSSL); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "证书已部署"})
}

// DisableSSL 关闭 SSL
func (h *WebsiteHandler) DisableSSL(c *gin.Context) {
	if h.unavailable(c) {
		return
	}
	domain := strings.TrimSpace(c.Param("domain"))
	if err := h.service.DisableSSL(domain); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "SSL 已关闭"})
}

// GatewayStatus 网关状态（Docker 不可用时返回明确状态而非报错）
func (h *WebsiteHandler) GatewayStatus(c *gin.Context) {
	if h.err != nil || h.service == nil {
		c.JSON(http.StatusOK, gin.H{"data": gin.H{
			"state":     "docker_unavailable",
			"httpPort":  config.Current().GatewayHTTPPort,
			"container": service.GatewayName,
			"error":     h.err.Error(),
		}})
		return
	}
	state, port := h.service.GatewayStatus()
	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"state":     state,
		"httpPort":  port,
		"container": service.GatewayName,
	}})
}

// EnsureGateway 启动/修复网关
func (h *WebsiteHandler) EnsureGateway(c *gin.Context) {
	if h.unavailable(c) {
		return
	}
	if err := h.service.EnsureGateway(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "网关已就绪"})
}
