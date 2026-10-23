package handler

import (
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"upanel/internal/service"
)

type CronHandler struct{}

func NewCronHandler() *CronHandler { return &CronHandler{} }

var cronIDRe = regexp.MustCompile(`^[a-f0-9]{16}$`)

func cronID(c *gin.Context) (string, bool) {
	id := strings.TrimSpace(c.Param("id"))
	if !cronIDRe.MatchString(id) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "任务 ID 不合法"})
		return "", false
	}
	return id, true
}

func (h *CronHandler) List(c *gin.Context) {
	jobs, err := service.ListCronJobs()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": jobs})
}

func (h *CronHandler) Create(c *gin.Context) {
	var req service.CronJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数有误: " + err.Error()})
		return
	}
	job, err := service.CreateCronJob(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": job, "message": "计划任务已创建"})
}

func (h *CronHandler) Update(c *gin.Context) {
	id, ok := cronID(c)
	if !ok {
		return
	}
	var req service.CronJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数有误: " + err.Error()})
		return
	}
	job, err := service.UpdateCronJob(id, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": job, "message": "计划任务已更新"})
}

func (h *CronHandler) Delete(c *gin.Context) {
	id, ok := cronID(c)
	if !ok {
		return
	}
	if err := service.DeleteCronJob(id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "计划任务已删除"})
}

func (h *CronHandler) Run(c *gin.Context) {
	id, ok := cronID(c)
	if !ok {
		return
	}
	if err := service.RunCronJob(id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "任务已开始执行"})
}
