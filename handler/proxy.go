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
	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = targetURL.Scheme
			req.URL.Host = targetURL.Host
			req.Host = targetURL.Host
		},
	}
	proxy.ServeHTTP(w, r)
}

// extractDomainPrefix 从 Host 提取 workerid.ownerid
// 支持两种格式:
// 1. workerid.ownerid.worker.example.com -> workerid.ownerid
// 2. workerid-ownerid.worker.example.com -> workerid.ownerid
func extractDomainPrefix(host string) string {
	// 找到第一个点的位置
	firstDot := -1
	for i := 0; i < len(host); i++ {
		if host[i] == '.' {
			firstDot = i
			break
		}
	}
	if firstDot == -1 {
		return ""
	}

	firstPart := host[:firstDot]
	rest := host[firstDot+1:]

	// 检查 rest 是否以 "worker." 开头 (杠连接格式)
	if len(rest) >= 7 && rest[:7] == "worker." {
		// 格式: workerid-ownerid.worker.example.com
		// 找到 firstPart 中的杠，转换为点
		for i := 0; i < len(firstPart); i++ {
			if firstPart[i] == '-' {
				return firstPart[:i] + "." + firstPart[i+1:]
			}
		}
		return ""
	}

	// 格式: workerid.ownerid.worker.example.com
	// 找到第二个点
	for i := 0; i < len(rest); i++ {
		if rest[i] == '.' {
			return firstPart + "." + rest[:i]
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
