package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"upanel/internal/config"
	"upanel/internal/handler"
	"upanel/internal/middleware"
)

var Version = "v0.1.0"

func main() {
	// 加载配置
	if err := config.LoadConfig(); err != nil {
		log.Printf("配置加载失败: %v", err)
	}
	
	// 设置 Gin 模式
	gin.SetMode(config.AppConfig.Mode)
	
	// 创建必要的目录
	os.MkdirAll(config.AppConfig.DataPath, 0755)
	os.MkdirAll(config.AppConfig.StaticPath, 0755)
	
	r := gin.Default()
	
	// 允许跨域（开发用）
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type,Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})
	
	// 静态文件服务（生产环境）
	staticPath := config.AppConfig.StaticPath
	if _, err := os.Stat(staticPath); err == nil {
		// 提供静态文件
		r.Static("/assets", filepath.Join(staticPath, "assets"))
		r.StaticFile("/favicon.ico", filepath.Join(staticPath, "favicon.ico"))
		r.StaticFile("/icons.svg", filepath.Join(staticPath, "icons.svg"))
		
		// 处理前端路由
		r.NoRoute(func(c *gin.Context) {
			// API 请求返回 404
			if len(c.Request.URL.Path) >= 4 && c.Request.URL.Path[:4] == "/api" {
				c.JSON(404, gin.H{"error": "API not found"})
				return
			}
			// 返回前端首页
			c.File(filepath.Join(staticPath, "index.html"))
		})
	}
	
	// 安全入口检查（如果配置了）
	securityEntry := config.AppConfig.GetSecurityEntry()
	if securityEntry != "" {
		// 添加安全入口中间件
		r.Use(func(c *gin.Context) {
			// 跳过登录页面和静态资源
			if c.Request.URL.Path == "/login" ||
				len(c.Request.URL.Path) >= 5 && c.Request.URL.Path[:5] == "/api/" ||
				len(c.Request.URL.Path) >= 7 && c.Request.URL.Path[:7] == "/assets" {
				c.Next()
				return
			}
			// 检查安全入口
			if c.Request.URL.Path != securityEntry && 
			   !(len(c.Request.URL.Path) > len(securityEntry) && 
			     c.Request.URL.Path[:len(securityEntry)] == securityEntry) {
				c.Header("Location", securityEntry)
				c.AbortWithStatus(302)
				return
			}
			c.Next()
		})
	}
	
	// API 路由
	api := r.Group("/api")
	{
		// 认证相关
		authHandler := handler.NewAuthHandler()
		api.POST("/auth/login", authHandler.Login)
		api.POST("/auth/logout", authHandler.Logout)
		
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
				c.JSON(200, gin.H{"version": "Docker v29.4.0"})
			})
			
			// 容器管理
			containerHandler, err := handler.NewContainerHandler()
			if err != nil {
				log.Printf("容器服务初始化失败: %v", err)
			} else {
				defer containerHandler.Close()
				containers := authorized.Group("/containers")
				{
					containers.GET("/", containerHandler.List)
					containers.POST("/:id/start", containerHandler.Start)
					containers.POST("/:id/stop", containerHandler.Stop)
					containers.POST("/:id/restart", containerHandler.Restart)
					containers.DELETE("/:id", containerHandler.Remove)
					containers.GET("/:id/logs", containerHandler.Logs)
				}
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
			}
			
			// 网站管理
			websiteHandler, err := handler.NewWebsiteHandler()
			if err != nil {
				log.Printf("网站服务初始化失败: %v", err)
			} else {
				defer websiteHandler.Close()
				websites := authorized.Group("/websites")
				{
					websites.POST("/", websiteHandler.Create)
					websites.GET("/", websiteHandler.List)
					websites.POST("/:domain/start", websiteHandler.Start)
					websites.POST("/:domain/stop", websiteHandler.Stop)
					websites.DELETE("/:domain", websiteHandler.Delete)
					websites.GET("/:domain/url", websiteHandler.GetURL)
				}
			}
			
			// 数据库管理
			databaseHandler, err := handler.NewDatabaseHandler()
			if err != nil {
				log.Printf("数据库服务初始化失败: %v", err)
			} else {
				defer databaseHandler.Close()
				databases := authorized.Group("/databases")
				{
					databases.POST("/", databaseHandler.Create)
					databases.GET("/", databaseHandler.List)
					databases.POST("/:name/start", databaseHandler.Start)
					databases.POST("/:name/stop", databaseHandler.Stop)
					databases.POST("/:name/restart", databaseHandler.Restart)
					databases.DELETE("/:name", databaseHandler.Delete)
					databases.GET("/:name/connection", databaseHandler.GetConnectionInfo)
				}
			}
			
			// 应用商店
			appsHandler, err := handler.NewAppsHandler()
			if err != nil {
				log.Printf("应用商店服务初始化失败: %v", err)
			} else {
				defer appsHandler.Close()
				apps := authorized.Group("/apps")
				{
					apps.GET("/", appsHandler.List)
					apps.POST("/install", appsHandler.Install)
				}
			}
			
			// 系统设置
			settingsHandler := handler.NewSettingsHandler()
			authorized.GET("/settings", settingsHandler.GetAll)
			authorized.PUT("/settings", settingsHandler.Update)
		}
	}
	
	// 启动服务器
	addr := ":" + config.AppConfig.Port
	fmt.Printf("🚀 UPanel %s 启动成功\n", Version)
	fmt.Printf("📍 监听地址: http://localhost%s\n", addr)
	if securityEntry != "" {
		fmt.Printf("🔐 安全入口: %s\n", securityEntry)
	}
	r.Run(addr)
}
