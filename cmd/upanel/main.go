package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"upanel/internal/config"
	"upanel/internal/handler"
	"upanel/internal/middleware"
	"upanel/internal/service"
)

// Version 面板版本（编译时通过 -ldflags 注入）
var Version = "v0.1.6-lts"

func main() {
	// 加载配置
	if err := config.LoadConfig(); err != nil {
		log.Printf("配置加载失败: %v", err)
	}
	cfg := config.Current()

	// 设置 Gin 模式
	gin.SetMode(cfg.Mode)

	// 创建必要的目录
	_ = cfg.EnsureDirs()

	r := gin.Default()

	// 跨域（本地开发用；生产环境建议由网关代理同域访问）
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type,Authorization")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// ========== 安全入口 ==========
	// 若配置了安全入口，所有页面/接口都必须带该前缀访问
	entry := cfg.GetSecurityEntry()
	var staticGroup *gin.RouterGroup
	if entry != "" {
		staticGroup = r.Group(entry)
	} else {
		staticGroup = r.Group("/")
	}

	// ========== 静态文件服务 ==========
	staticPath := cfg.StaticPath
	if _, err := os.Stat(staticPath); err == nil {
		staticGroup.Static("/assets", filepath.Join(staticPath, "assets"))
		staticGroup.StaticFile("/favicon.ico", filepath.Join(staticPath, "favicon.ico"))
		staticGroup.StaticFile("/favicon.svg", filepath.Join(staticPath, "favicon.svg"))
		staticGroup.StaticFile("/vite.svg", filepath.Join(staticPath, "vite.svg"))
		staticGroup.StaticFile("/icons.svg", filepath.Join(staticPath, "icons.svg"))

		indexFile := filepath.Join(staticPath, "index.html")
		staticGroup.GET("/", func(c *gin.Context) {
			c.File(indexFile)
		})
	}

	// 未知路径：API 返回 JSON 404，其余回落到前端路由
	r.NoRoute(func(c *gin.Context) {
		p := c.Request.URL.Path
		if len(p) >= 4 && p[:4] == "/api" {
			c.JSON(http.StatusNotFound, gin.H{"error": "API not found"})
			return
		}
		// 安全入口生效时，未带前缀的访问直接拒绝
		if entry != "" && !strings.HasPrefix(p, entry) {
			c.Status(http.StatusNotFound)
			return
		}
		c.File(filepath.Join(staticPath, "index.html"))
	})

	// API 路由始终挂载在 /api 下（不受安全入口影响）
	api := r.Group("/api")
	{
		// 认证
		authHandler := handler.NewAuthHandler()
		api.POST("/auth/login", authHandler.Login)
		api.POST("/auth/logout", authHandler.Logout)

		// 面板基础信息（无需登录，登录页需要展示版本）
		api.GET("/panel/info", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"version":  Version,
				"entry":    entry,
				"username": cfg.Username,
			})
		})

		// 需要认证的路由组
		authorized := api.Group("/")
		authorized.Use(middleware.AuthMiddleware())
		{
			authorized.GET("/auth/userinfo", authHandler.GetUserInfo)
			authorized.POST("/auth/change-password", authHandler.ChangePassword)

			// 系统信息
			systemHandler, err := handler.NewSystemHandler()
			if err != nil {
				log.Printf("系统服务初始化失败: %v", err)
			} else {
				defer systemHandler.Close()
				authorized.GET("/system/info", systemHandler.GetInfo)
			}

			// Docker 版本
			authorized.GET("/docker/version", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"version": "Docker v29.4.0"})
			})

			// 容器管理
			containerHandler, err := handler.NewContainerHandler()
			if err != nil {
				log.Printf("容器服务初始化失败: %v", err)
			}
			defer containerHandler.Close()
			containers := authorized.Group("/containers")
			{
				containers.GET("/", containerHandler.List)
				containers.GET("/all", containerHandler.List)
				containers.POST("/:id/start", containerHandler.Start)
				containers.POST("/:id/stop", containerHandler.Stop)
				containers.POST("/:id/restart", containerHandler.Restart)
				containers.DELETE("/:id", containerHandler.Remove)
				containers.GET("/:id/logs", containerHandler.Logs)
			}

			// 文件管理
			fileHandler := handler.NewFileHandler()
			files := authorized.Group("/files")
			{
				files.GET("/list", fileHandler.List)
				files.GET("/content", fileHandler.GetContent)
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
			websiteHandler, err := handler.NewWebsiteHandler()
			if err != nil {
				log.Printf("网站服务初始化失败: %v", err)
			}
			defer websiteHandler.Close()
			websites := authorized.Group("/websites")
			{
				websites.POST("/", websiteHandler.Create)
				websites.GET("/", websiteHandler.List)
				websites.GET("/:domain", websiteHandler.Get)
				websites.PUT("/:domain", websiteHandler.Update)
				websites.DELETE("/:domain", websiteHandler.Delete)
				websites.POST("/:domain/start", websiteHandler.Start)
				websites.POST("/:domain/stop", websiteHandler.Stop)
				websites.POST("/:domain/restart", websiteHandler.Restart)
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
			databaseHandler, err := handler.NewDatabaseHandler()
			if err != nil {
				log.Printf("数据库服务初始化失败: %v", err)
			}
			defer databaseHandler.Close()
			databases := authorized.Group("/databases")
			{
				databases.POST("/", databaseHandler.Create)
				databases.GET("/", databaseHandler.List)
				databases.GET("/:name", databaseHandler.Get)
				databases.POST("/:name/start", databaseHandler.Start)
				databases.POST("/:name/stop", databaseHandler.Stop)
				databases.POST("/:name/restart", databaseHandler.Restart)
				databases.DELETE("/:name", databaseHandler.Delete)
				databases.GET("/:name/connection", databaseHandler.GetConnectionInfo)
				databases.GET("/:name/logs", databaseHandler.Logs)
			}

			// 应用商店
			appsHandler, err := handler.NewAppsHandler()
			if err != nil {
				log.Printf("应用商店服务初始化失败: %v", err)
			}
			defer appsHandler.Close()
			apps := authorized.Group("/apps")
			{
				apps.GET("/", appsHandler.List)
				apps.GET("/installed", appsHandler.Installed)
				apps.POST("/install", appsHandler.Install)
				apps.POST("/:name/start", appsHandler.Start)
				apps.POST("/:name/stop", appsHandler.Stop)
				apps.POST("/:name/restart", appsHandler.Restart)
				apps.DELETE("/:name", appsHandler.Uninstall)
			}

			// 系统设置
			settingsHandler := handler.NewSettingsHandler()
			authorized.GET("/settings", settingsHandler.GetAll)
			authorized.PUT("/settings", settingsHandler.Update)

			// Docker 配置
			dockerHandler := handler.NewDockerHandler()
			docker := authorized.Group("/docker")
			{
				docker.GET("/config", dockerHandler.GetConfig)
				docker.PUT("/config", dockerHandler.UpdateConfig)
				docker.POST("/restart", dockerHandler.RestartDocker)
			}
		}
	}

	addr := cfg.BindIP + ":" + cfg.Port
	fmt.Println("════════════════════════════════════════════════")
	fmt.Printf("  UPanel %s 启动成功\n", Version)
	fmt.Printf("  面板地址: http://<服务器IP>:%s%s/\n", cfg.Port, entry)
	fmt.Printf("  数据目录: %s\n", cfg.DataPath)
	fmt.Printf("  前端目录: %s\n", cfg.StaticPath)
	fmt.Printf("  %s\n", dockerHint())
	fmt.Println("════════════════════════════════════════════════")
	if err := r.Run(addr); err != nil {
		log.Fatalf("服务启动失败: %v", err)
	}
}

func dockerHint() string {
	if rt, err := service.NewRuntime(); err == nil {
		defer rt.Close()
		return "Docker: 已连接"
	}
	return "Docker: 未连接（网站/数据库功能不可用）"
}
