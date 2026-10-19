package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"upanel/internal/service"
)

type DatabaseHandler struct {
	service *service.DatabaseService
	err     error
}

// NewDatabaseHandler 创建数据库处理器。
// Docker 不可用时仍返回处理器，保证路由注册，请求时再给出明确提示。
func NewDatabaseHandler() (*DatabaseHandler, error) {
	s, err := service.NewDatabaseService()
	return &DatabaseHandler{service: s, err: err}, nil
}

func (h *DatabaseHandler) Close() error {
	if h.service != nil {
		return h.service.Close()
	}
	return nil
}

// unavailable 服务不可用时响应明确错误
func (h *DatabaseHandler) unavailable(c *gin.Context) bool {
	if h.err != nil || h.service == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Docker 未连接，无法执行该操作: " + h.err.Error()})
		return true
	}
	return false
}

// 创建数据库
func (h *DatabaseHandler) Create(c *gin.Context) {
	if h.unavailable(c) {
		return
	}
	var req service.CreateDatabaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	database, err := h.service.CreateDatabase(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"data": database})
}

// 获取数据库列表
func (h *DatabaseHandler) List(c *gin.Context) {
	if h.unavailable(c) {
		return
	}
	databases, err := h.service.ListDatabases()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if databases == nil {
		databases = []service.Database{}
	}
	c.JSON(http.StatusOK, gin.H{"data": databases})
}

// 启动数据库
func (h *DatabaseHandler) Start(c *gin.Context) {
	if h.unavailable(c) {
		return
	}
	name := c.Param("name")
	
	if err := h.service.StartDatabase(name); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"message": "数据库已启动"})
}

// 停止数据库
func (h *DatabaseHandler) Stop(c *gin.Context) {
	if h.unavailable(c) {
		return
	}
	name := c.Param("name")
	
	if err := h.service.StopDatabase(name); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"message": "数据库已停止"})
}

// 重启数据库
func (h *DatabaseHandler) Restart(c *gin.Context) {
	if h.unavailable(c) {
		return
	}
	name := c.Param("name")
	
	if err := h.service.RestartDatabase(name); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"message": "数据库已重启"})
}

// 删除数据库
func (h *DatabaseHandler) Delete(c *gin.Context) {
	if h.unavailable(c) {
		return
	}
	name := c.Param("name")
	
	keepData := c.Query("keep_data") == "1" || c.Query("keepData") == "true"
	if err := h.service.DeleteDatabase(name, keepData); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"message": "数据库已删除"})
}

// 获取连接信息
func (h *DatabaseHandler) GetConnectionInfo(c *gin.Context) {
	if h.unavailable(c) {
		return
	}
	name := c.Param("name")
	
	info, err := h.service.ConnectionInfo(name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"data": info})
}

// Get 数据库详情
func (h *DatabaseHandler) Get(c *gin.Context) {
	if h.unavailable(c) {
		return
	}
	name := c.Param("name")

	db, err := h.service.GetDatabase(name)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": db})
}

// Logs 数据库容器日志
func (h *DatabaseHandler) Logs(c *gin.Context) {
	if h.unavailable(c) {
		return
	}
	name := c.Param("name")

	logs, err := h.service.Logs(name, 200)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": logs})
}
