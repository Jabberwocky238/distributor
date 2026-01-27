package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/jabberwocky238/distributor/store"
)

type ProxyHandler struct {
	store *store.MemoryStore
}

func NewProxyHandler(s *store.MemoryStore) *ProxyHandler {
	return &ProxyHandler{store: s}
}

func (h *ProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	host := r.Host
	// 移除端口号
	for i := len(host) - 1; i >= 0; i-- {
		if host[i] == ':' {
			host = host[:i]
			break
		}
	}

	// 从 Host 提取 worker_id.owner_id 前缀
	domainPrefix := extractDomainPrefix(host)
	if domainPrefix == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid host format: " + host})
		return
	}

	worker, ok := h.store.Get(domainPrefix)
	if !ok {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		json.NewEncoder(w).Encode(map[string]string{"error": "no worker found for: " + domainPrefix})
		return
	}

	serviceName := domainToServiceName(worker.DomainPrefix())
	target := fmt.Sprintf("http://%s.worker.svc.cluster.local:%d",
		serviceName, worker.Port)

	targetURL, _ := url.Parse(target)
	proxy := httputil.NewSingleHostReverseProxy(targetURL)
	proxy.ServeHTTP(w, r)
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
