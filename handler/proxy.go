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

	// 从 Host 提取 worker_id.owner_id 前缀
	// 格式: {worker_id}.{owner_id}.worker.example.com
	domainPrefix := extractDomainPrefix(host)
	if domainPrefix == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid host format: " + host,
		})
		return
	}

	worker, ok := h.store.Get(domainPrefix)
	if !ok {
		c.JSON(http.StatusBadGateway, gin.H{
			"error": "no worker found for: " + domainPrefix,
		})
		return
	}

	serviceName := domainToServiceName(worker.DomainPrefix())
	target := fmt.Sprintf("http://%s.worker.svc.cluster.local:%d",
		serviceName, worker.Port)

	targetURL, _ := url.Parse(target)
	proxy := httputil.NewSingleHostReverseProxy(targetURL)
	proxy.ServeHTTP(c.Writer, c.Request)
}

// extractDomainPrefix 从 Host 提取 worker_id.owner_id
// 输入: nginx.distributor.worker.example.com
// 输出: nginx.distributor
func extractDomainPrefix(host string) string {
	// 找到前两个点的位置，提取 worker_id.owner_id
	dotCount := 0
	for i := 0; i < len(host); i++ {
		if host[i] == '.' {
			dotCount++
			if dotCount == 2 {
				return host[:i]
			}
		}
	}
	return ""
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
