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
	var configPath, listen string
	flag.StringVar(&configPath, "c", "config.yaml", "config file path")
	flag.StringVar(&configPath, "config", "config.yaml", "config file path")
	flag.StringVar(&listen, "l", "localhost:8080", "listen address")
	flag.StringVar(&listen, "listen", "localhost:8080", "listen address")
	flag.Parse()

	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	memStore := store.NewMemoryStore()
	for _, w := range cfg.Workers {
		if _, err := memStore.Set(w); err != nil {
			log.Printf("failed to load worker %s: %v", w.Domain, err)
		}
	}

	var k8sClient *k8s.Client
	k8sClient, err = k8s.NewClient()
	if err != nil {
		log.Printf("k8s client init failed (running outside cluster?): %v", err)
	}

	proxyHandler := handler.NewProxyHandler(memStore)
	registerHandler := handler.NewRegisterHandler(memStore, k8sClient)

	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	api := r.Group("/api")
	{
		api.POST("/register", registerHandler.Register)
		api.DELETE("/register/:domain", registerHandler.Delete)
		api.GET("/workers", registerHandler.List)
	}

	r.NoRoute(proxyHandler.Handle)

	log.Printf("starting server on %s", listen)
	if err := r.Run(listen); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
