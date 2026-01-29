package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jabberwocky238/distributor/handler"
	"github.com/jabberwocky238/distributor/k8s"
	"github.com/jabberwocky238/distributor/store"
)

func main() {
	var adminListen string
	flag.StringVar(&adminListen, "admin", "localhost:8081", "admin listen address")
	flag.Parse()

	memStore := store.NewMemoryStore()
	var err error
	var k8sClient *k8s.Client
	k8sClient, err = k8s.NewClient()
	if err != nil {
		log.Printf("k8s client init failed (running outside cluster?): %v", err)
	}

	registerHandler := handler.NewRegisterHandler(memStore, k8sClient)

	// 管理服务器 (8081) - API + 健康检查
	admin := gin.Default()
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
