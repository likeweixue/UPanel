package service

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type FileInfo struct {
	Name     string    `json:"name"`
	Path     string    `json:"path"`
	Size     int64     `json:"size"`
	IsDir    bool      `json:"is_dir"`
	ModTime  time.Time `json:"mod_time"`
	Extension string   `json:"extension"`
}

type FileService struct {
	rootPath string
}

func NewFileService() *FileService {
	// 默认根目录为用户目录
	homeDir, _ := os.UserHomeDir()
	return &FileService{
		rootPath: homeDir,
	}
}

// 设置根目录
func (s *FileService) SetRootPath(path string) {
	s.rootPath = path
}

// 获取文件列表
func (s *FileService) ListFiles(path string) ([]FileInfo, error) {
	// 安全检查，防止路径穿越
	fullPath := filepath.Join(s.rootPath, path)
	if !strings.HasPrefix(fullPath, s.rootPath) {
		return nil, os.ErrPermission
	}

	entries, err := os.ReadDir(fullPath)
	if err != nil {
		return nil, err
	}

	var files []FileInfo
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}
		
		fileInfo := FileInfo{
			Name:    entry.Name(),
			Path:    filepath.Join(path, entry.Name()),
			Size:    info.Size(),
			IsDir:   entry.IsDir(),
			ModTime: info.ModTime(),
		}
		
		if !entry.IsDir() {
			ext := filepath.Ext(entry.Name())
			if len(ext) > 0 {
				fileInfo.Extension = ext[1:]
			}
		}
		
		files = append(files, fileInfo)
	}
	
	return files, nil
}

// 获取文件内容
func (s *FileService) GetFileContent(path string) (string, error) {
	fullPath := filepath.Join(s.rootPath, path)
	if !strings.HasPrefix(fullPath, s.rootPath) {
		return "", os.ErrPermission
	}
	
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// 保存文件内容
func (s *FileService) SaveFileContent(path, content string) error {
	fullPath := filepath.Join(s.rootPath, path)
	if !strings.HasPrefix(fullPath, s.rootPath) {
		return os.ErrPermission
	}
	
	return os.WriteFile(fullPath, []byte(content), 0644)
}

// 创建文件夹
func (s *FileService) CreateFolder(path, name string) error {
	fullPath := filepath.Join(s.rootPath, path, name)
	if !strings.HasPrefix(fullPath, s.rootPath) {
		return os.ErrPermission
	}
	
	return os.MkdirAll(fullPath, 0755)
}

// 创建文件
func (s *FileService) CreateFile(path, name string) error {
	fullPath := filepath.Join(s.rootPath, path, name)
	if !strings.HasPrefix(fullPath, s.rootPath) {
		return os.ErrPermission
	}
	
	file, err := os.Create(fullPath)
	if err != nil {
		return err
	}
	return file.Close()
}

// 重命名文件/文件夹
func (s *FileService) Rename(oldPath, newName string) error {
	fullOldPath := filepath.Join(s.rootPath, oldPath)
	if !strings.HasPrefix(fullOldPath, s.rootPath) {
		return os.ErrPermission
	}
	
	dir := filepath.Dir(fullOldPath)
	fullNewPath := filepath.Join(dir, newName)
	
	return os.Rename(fullOldPath, fullNewPath)
}

// 删除文件/文件夹
func (s *FileService) Delete(path string) error {
	fullPath := filepath.Join(s.rootPath, path)
	if !strings.HasPrefix(fullPath, s.rootPath) {
		return os.ErrPermission
	}
	
	return os.RemoveAll(fullPath)
}

// 上传文件
func (s *FileService) UploadFile(path string, filename string, reader io.Reader) error {
	fullPath := filepath.Join(s.rootPath, path)
	if !strings.HasPrefix(fullPath, s.rootPath) {
		return os.ErrPermission
	}
	
	// 确保目录存在
	if err := os.MkdirAll(fullPath, 0755); err != nil {
		return err
	}
	
	filePath := filepath.Join(fullPath, filename)
	dst, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer dst.Close()
	
	_, err = io.Copy(dst, reader)
	return err
}

// 下载文件
func (s *FileService) DownloadFile(path string) (string, error) {
	fullPath := filepath.Join(s.rootPath, path)
	if !strings.HasPrefix(fullPath, s.rootPath) {
		return "", os.ErrPermission
	}
	return fullPath, nil
}

// 获取当前路径信息
func (s *FileService) GetPathInfo(path string) (string, error) {
	fullPath := filepath.Join(s.rootPath, path)
	if !strings.HasPrefix(fullPath, s.rootPath) {
		return "", os.ErrPermission
	}
	
	absPath, err := filepath.Abs(fullPath)
	if err != nil {
		return "", err
	}
	return absPath, nil
}
