package handler

import (
    "net/http"
    "strconv"

    "github.com/gin-gonic/gin"
    "upanel/internal/service"
)

type ContainerHandler struct {
    service *service.ContainerService
    err     error
}

// NewContainerHandler 创建容器处理器。
// Docker 不可用时仍返回处理器，保证路由注册，请求时再给出明确提示。
func NewContainerHandler() (*ContainerHandler, error) {
    s, err := service.NewContainerService()
    return &ContainerHandler{service: s, err: err}, nil
}

func (h *ContainerHandler) Close() {
    if h.service != nil {
        h.service.Close()
    }
}

// unavailable 服务不可用时响应明确错误
func (h *ContainerHandler) unavailable(c *gin.Context) bool {
    if h.err != nil || h.service == nil {
        c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Docker 未连接，无法执行该操作: " + h.err.Error()})
        return true
    }
    return false
}

// 获取容器列表
func (h *ContainerHandler) List(c *gin.Context) {
    if h.unavailable(c) {
        return
    }
    all := c.Query("all") == "true"
    containers, err := h.service.ListContainers(all)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, gin.H{"data": containers})
}

// 启动容器
func (h *ContainerHandler) Start(c *gin.Context) {
    if h.unavailable(c) {
        return
    }
    id := c.Param("id")
    err := h.service.StartContainer(id)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, gin.H{"message": "容器已启动"})
}

// 停止容器
func (h *ContainerHandler) Stop(c *gin.Context) {
    if h.unavailable(c) {
        return
    }
    id := c.Param("id")
    err := h.service.StopContainer(id)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, gin.H{"message": "容器已停止"})
}

// 重启容器
func (h *ContainerHandler) Restart(c *gin.Context) {
    if h.unavailable(c) {
        return
    }
    id := c.Param("id")
    err := h.service.RestartContainer(id)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, gin.H{"message": "容器已重启"})
}

// 删除容器
func (h *ContainerHandler) Remove(c *gin.Context) {
    if h.unavailable(c) {
        return
    }
    id := c.Param("id")
    force := c.Query("force") == "true"
    err := h.service.RemoveContainer(id, force)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, gin.H{"message": "容器已删除"})
}

// 获取容器日志
func (h *ContainerHandler) Logs(c *gin.Context) {
    if h.unavailable(c) {
        return
    }
    id := c.Param("id")
    tail := 100
    if t := c.Query("tail"); t != "" {
        if parsed, err := strconv.Atoi(t); err == nil {
            tail = parsed
        }
    }
    logs, err := h.service.GetContainerLogs(id, tail)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, gin.H{"logs": logs})
}
