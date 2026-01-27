package handler

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/gin-gonic/gin"
	"github.com/jabberwocky238/distributor/store"
)

type ProxyHandler struct {
	store *store.MemoryStore
}

func NewProxyHandler(s *store.MemoryStore) *ProxyHandler {
	return &ProxyHandler{store: s}
}

func (h *ProxyHandler) Handle(c *gin.Context) {
	host := c.Request.Host
	// 移除端口号
	for i := len(host) - 1; i >= 0; i-- {
		if host[i] == ':' {
			host = host[:i]
			break
		}
	}

	worker, ok := h.store.Get(host)
	if !ok {
		c.JSON(http.StatusBadGateway, gin.H{
			"error": "no worker found for host: " + host,
		})
		return
	}

	serviceName := domainToServiceName(worker.Domain)
	target := fmt.Sprintf("http://%s.worker.svc.cluster.local:%d",
		serviceName, worker.Port)

	targetURL, _ := url.Parse(target)
	proxy := httputil.NewSingleHostReverseProxy(targetURL)
	proxy.ServeHTTP(c.Writer, c.Request)
}

func domainToServiceName(domain string) string {
	result := make([]byte, 0, len(domain))
	for i := 0; i < len(domain); i++ {
		if domain[i] == '.' {
			result = append(result, '-')
		} else {
			result = append(result, domain[i])
		}
	}
	return string(result)
}
