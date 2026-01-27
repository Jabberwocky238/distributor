package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jabberwocky238/distributor/config"
	"github.com/jabberwocky238/distributor/handler"
	"github.com/jabberwocky238/distributor/k8s"
	"github.com/jabberwocky238/distributor/store"
)

func main() {
	var configPath, listen, adminListen string
	flag.StringVar(&configPath, "c", "config.yaml", "config file path")
	flag.StringVar(&configPath, "config", "config.yaml", "config file path")
	flag.StringVar(&listen, "l", "localhost:8080", "business listen address")
	flag.StringVar(&listen, "listen", "localhost:8080", "business listen address")
	flag.StringVar(&adminListen, "admin", "localhost:8081", "admin listen address")
	flag.Parse()

	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	memStore := store.NewMemoryStore()
	for _, w := range cfg.Workers {
		if _, err := memStore.Set(w); err != nil {
			log.Printf("failed to load worker %s: %v", w.DomainPrefix(), err)
		}
	}

	var k8sClient *k8s.Client
	k8sClient, err = k8s.NewClient()
	if err != nil {
		log.Printf("k8s client init failed (running outside cluster?): %v", err)
	}

	proxyHandler := handler.NewProxyHandler(memStore)
	registerHandler := handler.NewRegisterHandler(memStore, k8sClient)

	// 业务服务器 (8080) - 反向代理 (原生 http)
	go func() {
		log.Printf("starting business server on %s", listen)
		if err := http.ListenAndServe(listen, proxyHandler); err != nil {
			log.Fatalf("business server error: %v", err)
		}
	}()

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
