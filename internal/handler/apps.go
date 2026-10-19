package handler

import (
	"net/http"

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
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": 503, "error": "Docker 未连接，无法执行该操作: " + h.err.Error()})
		return true
	}
	return false
}

// List 应用列表
func (h *AppsHandler) List(c *gin.Context) {
	if h.unavailable(c) {
		return
	}
	apps := h.service.GetApps()
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": apps, "msg": "success"})
}

// Install 安装应用
func (h *AppsHandler) Install(c *gin.Context) {
	if h.unavailable(c) {
		return
	}
	var req service.InstallAppRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "error": "请求参数有误: " + err.Error()})
		return
	}

	app, err := h.service.InstallApp(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": app, "message": "安装成功"})
}

// Installed 已安装应用列表
func (h *AppsHandler) Installed(c *gin.Context) {
	if h.unavailable(c) {
		return
	}
	list, err := h.service.ListInstalled()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "error": err.Error()})
		return
	}
	if list == nil {
		list = []service.InstalledApp{}
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": list, "msg": "success"})
}

// Start 启动应用
func (h *AppsHandler) Start(c *gin.Context) {
	if h.unavailable(c) {
		return
	}
	name := c.Param("name")
	if err := h.service.StartApp(name); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "应用已启动"})
}

// Stop 停止应用
func (h *AppsHandler) Stop(c *gin.Context) {
	if h.unavailable(c) {
		return
	}
	name := c.Param("name")
	if err := h.service.StopApp(name); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "应用已停止"})
}

// Restart 重启应用
func (h *AppsHandler) Restart(c *gin.Context) {
	if h.unavailable(c) {
		return
	}
	name := c.Param("name")
	if err := h.service.RestartApp(name); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "应用已重启"})
}

// Uninstall 卸载应用
func (h *AppsHandler) Uninstall(c *gin.Context) {
	if h.unavailable(c) {
		return
	}
	name := c.Param("name")
	keepData := c.Query("keep_data") == "1" || c.Query("keepData") == "true"
	if err := h.service.UninstallApp(name, keepData); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "应用已卸载"})
}
