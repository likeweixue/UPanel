package main

import (
    "fmt"
    "net/http"
    
    "github.com/gin-gonic/gin"
    "github.com/moby/moby/client"
)

func main() {
    // 1. 测试 Docker 连接
    cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
    if err != nil {
        fmt.Println("⚠️ Docker 连接失败:", err)
    } else {
        fmt.Println("✅ Docker 连接成功")
        defer cli.Close()
    }
    
    // 2. 创建 Gin 路由
    r := gin.Default()
    
    // 3. 首页
    r.GET("/", func(c *gin.Context) {
    c.JSON(http.StatusOK, gin.H{
        "message": "UPanel API 服务运行中",
        "status":  "ok",
    })
})
    
    // 4. API 示例：获取 Docker 版本
    r.GET("/api/docker/version", func(c *gin.Context) {
        if cli == nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Docker 未连接"})
            return
        }
        version, err := cli.ServerVersion(c.Request.Context(), client.ServerVersionOptions{})
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
            return
        }
        c.JSON(http.StatusOK, gin.H{
            "version": version.Version,
            "api":     version.APIVersion,
        })
    })
    
    // 5. 加载 HTML 模板
    // r.LoadHTMLGlob("web/templates/*")  // 前端已分离，不再需要
    
    // 6. 启动服务器
    fmt.Println("🚀 UPanel 启动成功")
    fmt.Println("📍 http://localhost:8080")
    r.Run(":8080")
}