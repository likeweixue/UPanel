package handler

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"upanel/internal/service"
)

type FileHandler struct {
	service *service.FileService
}

func NewFileHandler() *FileHandler {
	return &FileHandler{
		service: service.NewFileService(),
	}
}

// RootPath 文件管理根目录
func (h *FileHandler) RootPath() string {
	return h.service.RootPath()
}

func fileError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, os.ErrPermission):
		c.JSON(http.StatusForbidden, gin.H{"error": "没有权限访问该路径"})
	case os.IsNotExist(err):
		c.JSON(http.StatusNotFound, gin.H{"error": "文件或目录不存在"})
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}
}

// 获取文件列表
func (h *FileHandler) List(c *gin.Context) {
	path := c.DefaultQuery("path", "")

	files, err := h.service.ListFiles(path)
	if err != nil {
		fileError(c, err)
		return
	}

	currentPath, _ := h.service.GetPathInfo(path)

	c.JSON(http.StatusOK, gin.H{
		"data":          files,
		"current_path":  currentPath,
		"relative_path": path,
	})
}

// 获取文件内容
func (h *FileHandler) GetContent(c *gin.Context) {
	path := c.Query("path")

	content, err := h.service.GetFileContent(path)
	if err != nil {
		fileError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"content": content})
}

// 保存文件内容
func (h *FileHandler) SaveContent(c *gin.Context) {
	var req struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Path == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "路径不能为空"})
		return
	}

	if err := h.service.SaveFileContent(req.Path, req.Content); err != nil {
		fileError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "保存成功"})
}

// 创建文件夹
func (h *FileHandler) CreateFolder(c *gin.Context) {
	var req struct {
		Path string `json:"path"`
		Name string `json:"name"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.CreateFolder(req.Path, req.Name); err != nil {
		fileError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "创建成功"})
}

// 创建文件
func (h *FileHandler) CreateFile(c *gin.Context) {
	var req struct {
		Path string `json:"path"`
		Name string `json:"name"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.CreateFile(req.Path, req.Name); err != nil {
		fileError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "创建成功"})
}

// 重命名
func (h *FileHandler) Rename(c *gin.Context) {
	var req struct {
		Path    string `json:"path"`
		NewName string `json:"new_name"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Path == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "路径不能为空"})
		return
	}

	if err := h.service.Rename(req.Path, req.NewName); err != nil {
		fileError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "重命名成功"})
}

// 删除文件/文件夹
func (h *FileHandler) Delete(c *gin.Context) {
	path := c.Query("path")
	if path == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "路径不能为空"})
		return
	}

	if err := h.service.Delete(path); err != nil {
		fileError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

// 上传文件
func (h *FileHandler) Upload(c *gin.Context) {
	path := c.PostForm("path")
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "未收到上传文件: " + err.Error()})
		return
	}
	// 清理 multipart 临时文件（超过内存阈值的上传会落盘）
	defer func() {
		if c.Request.MultipartForm != nil {
			_ = c.Request.MultipartForm.RemoveAll()
		}
	}()
	// 单文件最大 2GB（全局 BodyLimit 同一上限）
	if file.Size > 2<<30 {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "文件过大（最大 2GB）"})
		return
	}

	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer src.Close()

	if err := h.service.UploadFile(path, file.Filename, src); err != nil {
		fileError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "上传成功"})
}

// 下载文件
func (h *FileHandler) Download(c *gin.Context) {
	path := c.Query("path")
	if path == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "路径不能为空"})
		return
	}
	full, err := h.service.DownloadFile(path)
	if err != nil {
		fileError(c, err)
		return
	}
	info, err := os.Stat(full)
	if err != nil || info.IsDir() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "只能下载文件"})
		return
	}
	name := filepath.Base(full)
	if !strings.Contains(name, "\"") {
		c.FileAttachment(full, name)
		return
	}
	c.File(full)
}
