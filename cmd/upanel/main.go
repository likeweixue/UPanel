package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"upanel/internal/config"
	"upanel/internal/handler"
	"upanel/internal/middleware"
	"upanel/internal/service"
)

// Version 面板版本（编译时通过 -ldflags 注入）
var Version = "v0.1.8"

func main() {
	// 加载配置
	if err := config.LoadConfig(); err != nil {
		log.Fatalf("配置加载失败: %v", err)
	}
	cfg := config.Current()

	gin.SetMode(cfg.Mode)

	if err := cfg.EnsureDirs(); err != nil {
		log.Printf("警告: 创建数据目录失败: %v", err)
	}

	// 启动计划任务调度器（无任务时开销可忽略）
	service.StartCronScheduler()

	r := gin.New()
	r.SetTrustedProxies(nil) // 不信任 X-Forwarded-For，避免限流被伪造 IP 绕过
	r.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		return fmt.Sprintf("[GIN] %s | %3d | %10s | %-15s | %s %s\n",
			param.TimeStamp.Format("2006-01-02 15:04:05"),
			param.StatusCode,
			param.Latency.Round(time.Millisecond),
			param.ClientIP,
			param.Method,
			param.Path,
		)
	}))
	r.Use(gin.Recovery())
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.BodyLimit())

	entry := cfg.GetSecurityEntry()
	apiPrefix := entry + "/api"

	// ========== 前端静态资源 ==========
	assetsDir := filepath.Join(cfg.StaticPath, "assets")

	assetHandler := func(c *gin.Context) {
		rel := strings.TrimPrefix(c.Param("filepath"), "/")
		clean := filepath.Clean(rel)
		if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) || filepath.IsAbs(clean) {
			c.Status(http.StatusNotFound)
			return
		}
		full := filepath.Join(assetsDir, clean)
		info, err := os.Stat(full)
		if err != nil || info.IsDir() {
			c.Status(http.StatusNotFound)
			return
		}
		c.Header("Cache-Control", "public, max-age=31536000, immutable")
		c.File(full)
	}
	r.GET(entry+"/assets/*filepath", assetHandler)
	r.HEAD(entry+"/assets/*filepath", assetHandler)

	for _, name := range []string{"favicon.svg", "favicon.ico"} {
		file := filepath.Join(cfg.StaticPath, name)
		if _, err := os.Stat(file); err != nil {
			continue
		}
		r.GET(entry+"/"+name, func(c *gin.Context) { c.File(file) })
	}

	serveIndex := func(c *gin.Context) {
		indexHTML := loadIndexHTML(cfg.StaticPath, entry)
		if indexHTML == nil {
			c.Data(http.StatusServiceUnavailable, "text/html; charset=utf-8",
				[]byte("<h1>UPanel 前端资源缺失</h1><p>请确认 web/ 目录存在，或重新解压发布包。</p>"))
			return
		}
		c.Header("Cache-Control", "no-cache")
		c.Data(http.StatusOK, "text/html; charset=utf-8", indexHTML)
	}
	r.GET(entry+"/", serveIndex)
	r.HEAD(entry+"/", serveIndex)
	if entry != "" {
		r.GET(entry, func(c *gin.Context) {
			c.Redirect(http.StatusPermanentRedirect, entry+"/")
		})
	}

	// ========== API ==========
	api := r.Group(apiPrefix)
	api.Use(middleware.NoCacheAPI())
	{
		authHandler := handler.NewAuthHandler()
		api.POST("/auth/login", middleware.LoginRateLimit(), authHandler.Login)

		api.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok", "version": Version})
		})

		// 面板基础信息（无需登录，登录页需要展示版本/名称）
		api.GET("/panel/info", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"version": Version,
				"entry":   entry,
				"name":    service.NewSettingsService().GetAll().PanelName,
			})
		})

		// 需要认证的路由组
		authorized := api.Group("/")
		authorized.Use(middleware.AuthMiddleware())
		{
			authorized.GET("/auth/userinfo", authHandler.GetUserInfo)
			authorized.POST("/auth/logout", authHandler.Logout)
			authorized.POST("/auth/change-password", authHandler.ChangePassword)

			// 系统信息 / SSH / 防火墙
			systemHandler, err := handler.NewSystemHandler()
			if err != nil {
				log.Printf("系统服务初始化失败: %v", err)
			}
			if systemHandler != nil {
				defer systemHandler.Close()
				authorized.GET("/system/info", systemHandler.GetInfo)
				authorized.GET("/system/ssh", systemHandler.SSH)
				authorized.GET("/system/firewall", systemHandler.Firewall)
			}

			// Docker
			dockerHandler := handler.NewDockerHandler()
			docker := authorized.Group("/docker")
			{
				docker.GET("/version", dockerHandler.Version)
				docker.GET("/config", dockerHandler.GetConfig)
				docker.PUT("/config", dockerHandler.UpdateConfig)
				docker.POST("/restart", dockerHandler.RestartDocker)
			}

			// 容器管理
			containerHandler, _ := handler.NewContainerHandler()
			defer containerHandler.Close()
			containers := authorized.Group("/containers")
			{
				containers.GET("", containerHandler.List)
				containers.GET("/all", containerHandler.List)
				containers.GET("/:id/logs", containerHandler.Logs)
				containers.POST("/:id/start", containerHandler.Start)
				containers.POST("/:id/stop", containerHandler.Stop)
				containers.POST("/:id/restart", containerHandler.Restart)
				containers.DELETE("/:id", containerHandler.Remove)
			}
			authorized.GET("/docker/images", containerHandler.Images)
			authorized.GET("/docker/volumes", containerHandler.Volumes)
			authorized.GET("/docker/networks", containerHandler.Networks)

			// 文件管理
			fileHandler := handler.NewFileHandler()
			files := authorized.Group("/files")
			{
				files.GET("/list", fileHandler.List)
				files.GET("/content", fileHandler.GetContent)
				files.GET("/download", fileHandler.Download)
				files.POST("/content", fileHandler.SaveContent)
				files.POST("/folder", fileHandler.CreateFolder)
				files.POST("/file", fileHandler.CreateFile)
				files.PUT("/rename", fileHandler.Rename)
				files.DELETE("/delete", fileHandler.Delete)
				files.POST("/upload", fileHandler.Upload)
				files.GET("/root", func(c *gin.Context) {
					c.JSON(http.StatusOK, gin.H{"root": fileHandler.RootPath()})
				})
			}

			// 网站管理
			websiteHandler, _ := handler.NewWebsiteHandler()
			defer websiteHandler.Close()
			websites := authorized.Group("/websites")
			{
				websites.POST("", websiteHandler.Create)
				websites.GET("", websiteHandler.List)
				websites.GET("/:domain", websiteHandler.Get)
				websites.PUT("/:domain", websiteHandler.Update)
				websites.DELETE("/:domain", websiteHandler.Delete)
				websites.POST("/:domain/start", websiteHandler.Start)
				websites.POST("/:domain/stop", websiteHandler.Stop)
				websites.POST("/:domain/restart", websiteHandler.Restart)
				websites.GET("/:domain/logs", websiteHandler.Logs)
				websites.POST("/:domain/ssl", websiteHandler.ApplySSL)
				websites.DELETE("/:domain/ssl", websiteHandler.DisableSSL)
			}

			// 网关
			gateway := authorized.Group("/gateway")
			{
				gateway.GET("/status", websiteHandler.GatewayStatus)
				gateway.POST("/ensure", websiteHandler.EnsureGateway)
			}

			// 数据库管理
			databaseHandler, _ := handler.NewDatabaseHandler()
			defer databaseHandler.Close()
			databases := authorized.Group("/databases")
			{
				databases.POST("", databaseHandler.Create)
				databases.GET("", databaseHandler.List)
				databases.GET("/:name", databaseHandler.Get)
				databases.POST("/:name/start", databaseHandler.Start)
				databases.POST("/:name/stop", databaseHandler.Stop)
				databases.POST("/:name/restart", databaseHandler.Restart)
				databases.DELETE("/:name", databaseHandler.Delete)
				databases.GET("/:name/connection", databaseHandler.GetConnectionInfo)
				databases.GET("/:name/logs", databaseHandler.Logs)
			}

			// 应用商店
			appsHandler, _ := handler.NewAppsHandler()
			defer appsHandler.Close()
			apps := authorized.Group("/apps")
			{
				apps.GET("", appsHandler.List)
				apps.GET("/installed", appsHandler.Installed)
				apps.POST("/install", appsHandler.Install)
				apps.GET("/:name/credentials", appsHandler.Credentials)
				apps.POST("/:name/start", appsHandler.Start)
				apps.POST("/:name/stop", appsHandler.Stop)
				apps.POST("/:name/restart", appsHandler.Restart)
				apps.DELETE("/:name", appsHandler.Uninstall)
			}

			// 计划任务
			cronHandler := handler.NewCronHandler()
			cron := authorized.Group("/cron")
			{
				cron.GET("", cronHandler.List)
				cron.POST("", cronHandler.Create)
				cron.PUT("/:id", cronHandler.Update)
				cron.DELETE("/:id", cronHandler.Delete)
				cron.POST("/:id/run", cronHandler.Run)
			}

			// 邮件通知
			emailHandler := handler.NewEmailHandler()
			emailGroup := authorized.Group("/email")
			{
				emailGroup.GET("/config", emailHandler.Get)
				emailGroup.PUT("/config", emailHandler.Update)
				emailGroup.POST("/test", emailHandler.Test)
			}

			// 安全中心
			securityHandler := handler.NewSecurityHandler()
			authorized.GET("/security/overview", securityHandler.Overview)
			authorized.GET("/security/audit", securityHandler.Audit)

			// 系统设置
			settingsHandler := handler.NewSettingsHandler()
			authorized.GET("/settings", settingsHandler.GetAll)
			authorized.PUT("/settings", settingsHandler.Update)
		}
	}

	// 未知路径：API 返回 JSON 404，其余回落到前端路由
	r.NoRoute(func(c *gin.Context) {
		p := c.Request.URL.Path
		if p == apiPrefix || strings.HasPrefix(p, apiPrefix+"/") {
			c.JSON(http.StatusNotFound, gin.H{"error": "API not found"})
			return
		}
		// 安全入口生效时，未带前缀的访问一律 404，不泄露面板存在
		if entry != "" && !strings.HasPrefix(p, entry+"/") && p != entry {
			c.Status(http.StatusNotFound)
			return
		}
		if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
			c.Status(http.StatusNotFound)
			return
		}
		serveIndex(c)
	})

	addr := cfg.BindIP + ":" + cfg.Port
	printBanner(cfg, entry)

	srv := &http.Server{
		Addr:              addr,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Minute,
		WriteTimeout:      30 * time.Minute,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("服务启动失败: %v", err)
		}
	}()

	// 优雅退出
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("正在关闭面板服务 ...")
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("关闭超时: %v", err)
	}
}

func printBanner(cfg *config.Config, entry string) {
	fmt.Println("════════════════════════════════════════════════")
	fmt.Printf("  UPanel %s 启动成功\n", Version)
	fmt.Printf("  面板地址: http://<服务器IP>:%s%s/\n", cfg.Port, entry)
	fmt.Printf("  数据目录: %s\n", cfg.DataPath)
	fmt.Printf("  前端目录: %s\n", cfg.StaticPath)
	if _, err := os.Stat(cfg.InitialPasswordFile()); err == nil {
		fmt.Printf("  初始密码: %s（登录后请删除）\n", cfg.InitialPasswordFile())
	}
	fmt.Printf("  %s\n", dockerHint())
	fmt.Println("════════════════════════════════════════════════")
}

// loadIndexHTML 读取 index.html 并注入 <base>，让前端资源在安全入口下也能正确加载。
func loadIndexHTML(staticPath, entry string) []byte {
	raw, err := os.ReadFile(filepath.Join(staticPath, "index.html"))
	if err != nil {
		log.Printf("警告: 未找到前端 index.html (%v)，面板界面不可用", err)
		return nil
	}
	href := "/"
	if entry != "" {
		href = entry + "/"
	}
	baseTag := []byte(`<base href="` + href + `">`)
	if bytes.Contains(raw, []byte("<!--UPANEL_BASE-->")) {
		raw = bytes.Replace(raw, []byte("<!--UPANEL_BASE-->"), baseTag, 1)
	} else if !bytes.Contains(raw, []byte("<base ")) {
		raw = bytes.Replace(raw, []byte("<head>"), []byte("<head>\n    "+string(baseTag)), 1)
	}
	return raw
}

func dockerHint() string {
	if rt, err := service.NewRuntime(); err == nil {
		defer rt.Close()
		return "Docker: 已连接"
	}
	return "Docker: 未连接（网站/数据库功能不可用）"
}
