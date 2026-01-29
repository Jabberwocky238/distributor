package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	var adminListen, storeFile string
	flag.StringVar(&adminListen, "admin", "localhost:8081", "admin listen address")
	flag.StringVar(&storeFile, "store", "/data/workers.json", "workers store file path")
	flag.Parse()

	memStore, err := NewMemoryStore(storeFile)
	if err != nil {
		log.Fatalf("failed to initialize memory store: %v", err)
	}
	var k8sClient *K8sClient
	k8sClient, err = NewK8sClient()
	if err != nil {
		log.Printf("k8s client init failed (running outside cluster?): %v", err)
	}

	registerHandler := NewRegisterHandler(memStore, k8sClient)

	// 管理服务器 (8081) - API + 健康检查
	admin := gin.Default()
	admin.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "text/html", `<html>
	<head><title>Distributor Admin</title></head>
	<body>
	<h1>WELCOME TO JW238 Distributor Admin</h1>
	</body>
	<body>`)
	})
	admin.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	api := admin.Group("/api")
	{
		api.POST("/register", registerHandler.Register)
		api.DELETE("/register/:domain", registerHandler.Delete)
		api.GET("/workers", registerHandler.List)
	}

	// 启动管理服务器
	log.Printf("starting admin server on %s", adminListen)
	if err := admin.Run(adminListen); err != nil {
		log.Fatalf("admin server error: %v", err)
	}
}
