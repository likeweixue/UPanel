package handler

import (
    "net/http"
    "strconv"

    "github.com/gin-gonic/gin"
    "upanel/internal/service"
)

type ContainerHandler struct {
    service *service.ContainerService
}

func NewContainerHandler() (*ContainerHandler, error) {
    s, err := service.NewContainerService()
    if err != nil {
        return nil, err
    }
    return &ContainerHandler{service: s}, nil
}

func (h *ContainerHandler) Close() {
    h.service.Close()
}

// 获取容器列表
func (h *ContainerHandler) List(c *gin.Context) {
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
