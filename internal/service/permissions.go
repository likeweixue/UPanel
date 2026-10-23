package service

import "os"

// 面板以 root 运行，但容器内的 PHP / Redis / MinIO 等进程通常以非 root 用户运行。
// 宿主机 bind mount 不会自动继承容器内用户权限，因此需要对数据目录做最小限度的
// ownership 调整，否则应用会因无法写入数据目录而启动失败。
// 非 root 环境（例如开发机）静默跳过。
func prepareWritableDir(path string, uid, gid int) {
	if path == "" {
		return
	}
	_ = os.Chmod(path, 0775)
	if os.Geteuid() != 0 {
		return
	}
	_ = os.Chown(path, uid, gid)
}

func preparePHPDir(path string)       { prepareWritableDir(path, 33, 33) }     // Debian php-fpm: www-data
func prepareRedisDir(path string)     { prepareWritableDir(path, 999, 999) }   // redis:alpine
func prepareMySQLDir(path string)     { prepareWritableDir(path, 999, 999) }   // mysql / mariadb
func preparePostgresDir(path string)  { prepareWritableDir(path, 999, 999) }   // postgres
func prepareMinioDir(path string)     { prepareWritableDir(path, 1000, 1000) } // minio
func prepareWordpressDir(path string) { prepareWritableDir(path, 33, 33) }     // wordpress: www-data
