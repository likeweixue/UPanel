package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"upanel/internal/service"
)

type DatabaseHandler struct {
	service *service.DatabaseService
}

func NewDatabaseHandler() (*DatabaseHandler, error) {
	s, err := service.NewDatabaseService()
	if err != nil {
		return nil, err
	}
	return &DatabaseHandler{service: s}, nil
}

func (h *DatabaseHandler) Close() error {
	return h.service.Close()
}

// 创建数据库
func (h *DatabaseHandler) Create(c *gin.Context) {
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
	databases, err := h.service.ListDatabases()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"data": databases})
}

// 启动数据库
func (h *DatabaseHandler) Start(c *gin.Context) {
	name := c.Param("name")
	
	if err := h.service.StartDatabase(name); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"message": "数据库已启动"})
}

// 停止数据库
func (h *DatabaseHandler) Stop(c *gin.Context) {
	name := c.Param("name")
	
	if err := h.service.StopDatabase(name); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"message": "数据库已停止"})
}

// 重启数据库
func (h *DatabaseHandler) Restart(c *gin.Context) {
	name := c.Param("name")
	
	if err := h.service.RestartDatabase(name); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"message": "数据库已重启"})
}

// 删除数据库
func (h *DatabaseHandler) Delete(c *gin.Context) {
	name := c.Param("name")
	
	if err := h.service.DeleteDatabase(name); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"message": "数据库已删除"})
}

// 获取连接信息
func (h *DatabaseHandler) GetConnectionInfo(c *gin.Context) {
	name := c.Param("name")
	
	info, err := h.service.GetConnectionInfo(name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"data": info})
}
