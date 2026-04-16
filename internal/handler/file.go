package handler

import (
	"net/http"

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

// 获取文件列表
func (h *FileHandler) List(c *gin.Context) {
	path := c.DefaultQuery("path", "")
	
	files, err := h.service.ListFiles(path)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	// 获取当前路径的绝对路径
	currentPath, _ := h.service.GetPathInfo(path)
	
	c.JSON(http.StatusOK, gin.H{
		"data": files,
		"current_path": currentPath,
		"relative_path": path,
	})
}

// 获取文件内容
func (h *FileHandler) GetContent(c *gin.Context) {
	path := c.Query("path")
	
	content, err := h.service.GetFileContent(path)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
	
	if err := h.service.SaveFileContent(req.Path, req.Content); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
	
	if err := h.service.Rename(req.Path, req.NewName); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"message": "重命名成功"})
}

// 删除文件/文件夹
func (h *FileHandler) Delete(c *gin.Context) {
	path := c.Query("path")
	
	if err := h.service.Delete(path); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

// 上传文件
func (h *FileHandler) Upload(c *gin.Context) {
	path := c.PostForm("path")
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// 打开上传的文件
	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer src.Close()
	
	// 保存文件
	if err := h.service.UploadFile(path, file.Filename, src); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"message": "上传成功"})
}
