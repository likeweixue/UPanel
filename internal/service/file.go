package service

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

// FileInfo 文件条目
type FileInfo struct {
	Name      string    `json:"name"`
	Path      string    `json:"path"`
	Size      int64     `json:"size"`
	IsDir     bool      `json:"isDir"`
	ModTime   time.Time `json:"modTime"`
	Extension string    `json:"extension"`
	Mode      string    `json:"mode"`
}

// FileService 文件管理服务
type FileService struct {
	rootPath string
}

// NewFileService 创建文件服务。
// 服务器面板以 root 运行，根目录为 / ；非 Linux（开发机）回退到用户目录。
func NewFileService() *FileService {
	root := "/"
	if runtime.GOOS != "linux" {
		if home, err := os.UserHomeDir(); err == nil {
			root = home
		}
	}
	return &FileService{rootPath: root}
}

// SetRootPath 设置根目录
func (s *FileService) SetRootPath(path string) {
	if path == "" {
		return
	}
	s.rootPath = filepath.Clean(path)
}

// RootPath 当前根目录
func (s *FileService) RootPath() string {
	return s.rootPath
}

// safeJoin 把相对路径安全地拼接到根目录，阻止 ../ 穿越
func (s *FileService) safeJoin(path string) (string, error) {
	cleaned := filepath.Clean("/" + strings.TrimPrefix(filepath.Clean(path), "/"))
	full := filepath.Join(s.rootPath, cleaned)
	root := filepath.Clean(s.rootPath)
	if full != root && !strings.HasPrefix(full, root+string(os.PathSeparator)) {
		return "", os.ErrPermission
	}
	return full, nil
}

// ListFiles 获取文件列表
func (s *FileService) ListFiles(path string) ([]FileInfo, error) {
	fullPath, err := s.safeJoin(path)
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(fullPath)
	if err != nil {
		return nil, err
	}

	files := make([]FileInfo, 0, len(entries))
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}
		fi := FileInfo{
			Name:    entry.Name(),
			Path:    filepath.Join(path, entry.Name()),
			Size:    info.Size(),
			IsDir:   entry.IsDir(),
			ModTime: info.ModTime(),
			Mode:    info.Mode().String(),
		}
		if !entry.IsDir() {
			if ext := filepath.Ext(entry.Name()); len(ext) > 0 {
				fi.Extension = ext[1:]
			}
		}
		files = append(files, fi)
	}
	sort.Slice(files, func(i, j int) bool {
		if files[i].IsDir != files[j].IsDir {
			return files[i].IsDir
		}
		return strings.ToLower(files[i].Name) < strings.ToLower(files[j].Name)
	})
	return files, nil
}

// GetFileContent 获取文件内容
func (s *FileService) GetFileContent(path string) (string, error) {
	fullPath, err := s.safeJoin(path)
	if err != nil {
		return "", err
	}
	// 简单保护：拒绝超大文件
	if info, err := os.Stat(fullPath); err == nil && info.Size() > 8*1024*1024 {
		return "", os.ErrInvalid
	}
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// SaveFileContent 保存文件内容
func (s *FileService) SaveFileContent(path, content string) error {
	fullPath, err := s.safeJoin(path)
	if err != nil {
		return err
	}
	if info, err := os.Stat(fullPath); err == nil && info.IsDir() {
		return fmt.Errorf("目标是目录，不能作为文件保存")
	}
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return err
	}
	return os.WriteFile(fullPath, []byte(content), 0644)
}

// CreateFolder 创建文件夹
func (s *FileService) CreateFolder(path, name string) error {
	if err := validateEntryName(name); err != nil {
		return err
	}
	fullPath, err := s.safeJoin(filepath.Join(path, name))
	if err != nil {
		return err
	}
	return os.Mkdir(fullPath, 0755)
}

// CreateFile 创建文件
func (s *FileService) CreateFile(path, name string) error {
	if err := validateEntryName(name); err != nil {
		return err
	}
	fullPath, err := s.safeJoin(filepath.Join(path, name))
	if err != nil {
		return err
	}
	f, err := os.OpenFile(fullPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	return f.Close()
}

func validateEntryName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" || name == "." || name == ".." {
		return fmt.Errorf("名称不能为空")
	}
	if len(name) > 255 {
		return fmt.Errorf("名称过长")
	}
	if strings.ContainsAny(name, "/\\\x00\r\n") {
		return fmt.Errorf("名称包含非法字符")
	}
	return nil
}

// Rename 重命名
func (s *FileService) Rename(oldPath, newName string) error {
	fullOld, err := s.safeJoin(oldPath)
	if err != nil {
		return err
	}
	if filepath.Clean(fullOld) == filepath.Clean(s.rootPath) {
		return os.ErrPermission
	}
	if err := validateEntryName(newName); err != nil {
		return err
	}
	fullNew := filepath.Join(filepath.Dir(fullOld), newName)
	relNew := strings.TrimPrefix(fullNew, filepath.Clean(s.rootPath))
	if _, err := s.safeJoin(relNew); err != nil {
		return err
	}
	if _, err := os.Stat(fullNew); err == nil {
		return fmt.Errorf("目标名称已存在")
	}
	return os.Rename(fullOld, fullNew)
}

// Delete 删除文件或目录
func (s *FileService) Delete(path string) error {
	fullPath, err := s.safeJoin(path)
	if err != nil {
		return err
	}
	// 禁止删除根目录
	if filepath.Clean(fullPath) == filepath.Clean(s.rootPath) {
		return os.ErrPermission
	}
	return os.RemoveAll(fullPath)
}

// UploadFile 上传文件
func (s *FileService) UploadFile(path, filename string, reader io.Reader) error {
	fullPath, err := s.safeJoin(path)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(fullPath, 0755); err != nil {
		return err
	}
	// 去掉客户端可能携带的目录部分（Windows 下为 \），只保留文件名
	filename = filepath.Base(strings.ReplaceAll(filename, "\\", "/"))
	if err := validateEntryName(filename); err != nil {
		return err
	}
	dst, err := os.OpenFile(filepath.Join(fullPath, filename), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer dst.Close()
	_, err = io.Copy(dst, reader)
	return err
}

// DownloadFile 返回可下载的绝对路径
func (s *FileService) DownloadFile(path string) (string, error) {
	return s.safeJoin(path)
}

// GetPathInfo 获取绝对路径
func (s *FileService) GetPathInfo(path string) (string, error) {
	full, err := s.safeJoin(path)
	if err != nil {
		return "", err
	}
	return filepath.Abs(full)
}
