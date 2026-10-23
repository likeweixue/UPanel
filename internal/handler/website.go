package handler

import (
	"net/http"
	"strconv"
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
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Docker 未连接，该功能暂不可用",
			"code":  "docker_unavailable",
		})
		return true
	}
	return false
}

// domainParam 读取并校验路径中的域名，阻断路径穿越与非法字符
func domainParam(c *gin.Context) (string, bool) {
	domain := strings.ToLower(strings.TrimSpace(c.Param("domain")))
	if err := service.ValidateDomain(domain); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "域名不合法: " + err.Error()})
		return "", false
	}
	return domain, true
}

func isSiteBadRequest(err error) bool {
	msg := err.Error()
	for _, prefix := range []string{
		"域名", "不支持的网站类型", "网站 ", "别名", "PHP 版本", "镜像", "端口",
		"暂不支持", "站点目录", "证书", "请求参数",
	} {
		if strings.HasPrefix(msg, prefix) || strings.Contains(msg, prefix) {
			return true
		}
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
	if strings.TrimSpace(req.Domain) == "" {
		req.Domain = c.PostForm("domain")
	}

	website, err := h.service.CreateWebsite(&req)
	if err != nil {
		status := http.StatusInternalServerError
		if isSiteBadRequest(err) {
			status = http.StatusBadRequest
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": website, "message": "网站创建成功"})
}

// List 网站列表
func (h *WebsiteHandler) List(c *gin.Context) {
	if h.service == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "网站服务不可用", "code": "docker_unavailable"})
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
	if h.service == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "网站服务不可用", "code": "docker_unavailable"})
		return
	}
	domain, ok := domainParam(c)
	if !ok {
		return
	}
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
	domain, ok := domainParam(c)
	if !ok {
		return
	}
	var patch service.Website
	if err := c.ShouldBindJSON(&patch); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数有误: " + err.Error()})
		return
	}
	w, err := h.service.UpdateWebsite(domain, &patch)
	if err != nil {
		status := http.StatusInternalServerError
		if isSiteBadRequest(err) {
			status = http.StatusBadRequest
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": w, "message": "网站已更新"})
}

// Start 启动
func (h *WebsiteHandler) Start(c *gin.Context) {
	if h.unavailable(c) {
		return
	}
	domain, ok := domainParam(c)
	if !ok {
		return
	}
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
	domain, ok := domainParam(c)
	if !ok {
		return
	}
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
	domain, ok := domainParam(c)
	if !ok {
		return
	}
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
	domain, ok := domainParam(c)
	if !ok {
		return
	}
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
	domain, ok := domainParam(c)
	if !ok {
		return
	}
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
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "证书已部署"})
}

// DisableSSL 关闭 SSL
func (h *WebsiteHandler) DisableSSL(c *gin.Context) {
	if h.unavailable(c) {
		return
	}
	domain, ok := domainParam(c)
	if !ok {
		return
	}
	if err := h.service.DisableSSL(domain); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "SSL 已关闭"})
}

// Logs 站点访问/错误日志
func (h *WebsiteHandler) Logs(c *gin.Context) {
	if h.service == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "网站服务不可用", "code": "docker_unavailable"})
		return
	}
	domain, ok := domainParam(c)
	if !ok {
		return
	}
	lines := 200
	if v := c.Query("lines"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			lines = n
		}
	}
	kind := c.DefaultQuery("kind", "access")
	logs, err := h.service.GetSiteLogs(domain, kind, lines)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": logs, "kind": kind, "domain": domain})
}

// GatewayStatus 网关状态（Docker 不可用时返回明确状态而非报错）
func (h *WebsiteHandler) GatewayStatus(c *gin.Context) {
	if h.err != nil || h.service == nil {
		c.JSON(http.StatusOK, gin.H{"data": gin.H{
			"state":     "docker_unavailable",
			"httpPort":  config.Current().GatewayHTTPPort,
			"httpsPort": config.Current().GatewayHTTPSPort,
			"container": service.GatewayName,
			"error":     "Docker 未连接，网站网关状态暂不可用",
		}})
		return
	}
	state, port := h.service.GatewayStatus()
	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"state":     state,
		"httpPort":  port,
		"httpsPort": config.Current().GatewayHTTPSPort,
		"container": service.GatewayName,
		"image":     config.Current().GatewayImage,
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
