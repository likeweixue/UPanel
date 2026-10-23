package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"upanel/internal/service"
)

// AppsHandler 应用商店接口
type AppsHandler struct {
	service *service.AppService
	err     error
}

// NewAppsHandler 创建应用处理器。
// Docker 不可用时仍返回处理器，保证路由注册，请求时再给出明确提示。
func NewAppsHandler() (*AppsHandler, error) {
	s, err := service.NewAppService()
	return &AppsHandler{service: s, err: err}, nil
}

func (h *AppsHandler) Close() error {
	if h.service != nil {
		return h.service.Close()
	}
	return nil
}

// unavailable 服务不可用时响应明确错误
func (h *AppsHandler) unavailable(c *gin.Context) bool {
	if h.err != nil || h.service == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Docker 未连接，该功能暂不可用",
			"code":  "docker_unavailable",
		})
		return true
	}
	return false
}

func appNameParam(c *gin.Context) (string, bool) {
	name := strings.TrimSpace(c.Param("name"))
	if err := service.ValidateAppName(name); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return "", false
	}
	return name, true
}

// List 应用列表
func (h *AppsHandler) List(c *gin.Context) {
	if h.service == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "应用商店服务不可用", "code": "docker_unavailable"})
		return
	}
	apps := h.service.GetApps()
	c.JSON(http.StatusOK, gin.H{"data": apps})
}

// Install 安装应用
func (h *AppsHandler) Install(c *gin.Context) {
	if h.unavailable(c) {
		return
	}
	var req service.InstallAppRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数有误: " + err.Error()})
		return
	}

	app, err := h.service.InstallApp(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": app, "message": "安装成功"})
}

// Installed 已安装应用列表
func (h *AppsHandler) Installed(c *gin.Context) {
	if h.service == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "应用商店服务不可用", "code": "docker_unavailable"})
		return
	}
	list, err := h.service.ListInstalled()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if list == nil {
		list = []service.InstalledApp{}
	}
	c.JSON(http.StatusOK, gin.H{"data": list})
}

// Credentials 查看应用凭据
func (h *AppsHandler) Credentials(c *gin.Context) {
	if h.service == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "应用商店服务不可用", "code": "docker_unavailable"})
		return
	}
	name, ok := appNameParam(c)
	if !ok {
		return
	}
	creds, err := h.service.GetCredentials(name)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": creds})
}

// Start 启动应用
func (h *AppsHandler) Start(c *gin.Context) {
	if h.unavailable(c) {
		return
	}
	name, ok := appNameParam(c)
	if !ok {
		return
	}
	if err := h.service.StartApp(name); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "应用已启动"})
}

// Stop 停止应用
func (h *AppsHandler) Stop(c *gin.Context) {
	if h.unavailable(c) {
		return
	}
	name, ok := appNameParam(c)
	if !ok {
		return
	}
	if err := h.service.StopApp(name); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "应用已停止"})
}

// Restart 重启应用
func (h *AppsHandler) Restart(c *gin.Context) {
	if h.unavailable(c) {
		return
	}
	name, ok := appNameParam(c)
	if !ok {
		return
	}
	if err := h.service.RestartApp(name); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "应用已重启"})
}

// Uninstall 卸载应用
func (h *AppsHandler) Uninstall(c *gin.Context) {
	if h.unavailable(c) {
		return
	}
	name, ok := appNameParam(c)
	if !ok {
		return
	}
	keepData := c.Query("keep_data") == "1" || c.Query("keepData") == "true"
	if err := h.service.UninstallApp(name, keepData); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "应用已卸载"})
}
