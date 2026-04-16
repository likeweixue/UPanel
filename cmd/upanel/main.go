package main

import (
	"fmt"
	"log"

	"upanel/internal/handler"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	// 允许跨域（开发用）
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// API 路由
	api := r.Group("/api")
	{
		// 系统信息
		systemHandler, err := handler.NewSystemHandler()
		if err != nil {
			log.Printf("系统服务初始化失败: %v", err)
		} else {
			defer systemHandler.Close()
			api.GET("/system/info", systemHandler.GetInfo)
		}

		// Docker 版本
		api.GET("/docker/version", func(c *gin.Context) {
			c.JSON(200, gin.H{"version": "Docker v29.4.0"})
		})

		// 容器管理
		containerHandler, err := handler.NewContainerHandler()
		if err != nil {
			log.Printf("容器服务初始化失败: %v", err)
		} else {
			defer containerHandler.Close()
			containers := api.Group("/containers")
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
		files := api.Group("/files")
		{
			files.GET("/list", fileHandler.List)
			files.GET("/content", fileHandler.GetContent)
			files.POST("/content", fileHandler.SaveContent)
			files.POST("/folder", fileHandler.CreateFolder)
			files.POST("/file", fileHandler.CreateFile)
			files.PUT("/rename", fileHandler.Rename)
			files.DELETE("/delete", fileHandler.Delete)
			files.POST("/upload", fileHandler.Upload)
		} // 网站管理
		websiteHandler, err := handler.NewWebsiteHandler()
		if err != nil {
			log.Printf("网站服务初始化失败: %v", err)
		} else {
			defer websiteHandler.Close()
			websites := api.Group("/websites")
			{
				websites.POST("/", websiteHandler.Create)
				websites.GET("/", websiteHandler.List)
				websites.POST("/:domain/start", websiteHandler.Start)
				websites.POST("/:domain/stop", websiteHandler.Stop)
				websites.DELETE("/:domain", websiteHandler.Delete)
				websites.GET("/:domain/url", websiteHandler.GetURL)
			}
		}
	}

	fmt.Println("🚀 UPanel 启动成功")
	fmt.Println("📍 http://localhost:8080")
	r.Run(":8080")
}
