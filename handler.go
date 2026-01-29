package main

import (
	"net/http"
	"regexp"

	"github.com/gin-gonic/gin"
)

var validIDRegex = regexp.MustCompile(`^[a-z0-9]+$`)

type Worker struct {
	WorkerID string          `json:"worker_id" yaml:"worker_id"`
	OwnerID  string          `json:"owner_id" yaml:"owner_id"`
	Image    string          `json:"image" yaml:"image"`
	Port     int             `json:"port" yaml:"port"`
	History  []HistoryRecord `json:"history" yaml:"-"`
}

type RegisterHandler struct {
	store     *MemoryStore
	k8sClient *K8sClient
}

func NewRegisterHandler(s *MemoryStore, k *K8sClient) *RegisterHandler {
	return &RegisterHandler{store: s, k8sClient: k}
}

type RegisterRequest struct {
	WorkerID string `json:"worker_id" binding:"required"`
	OwnerID  string `json:"owner_id" binding:"required"`
	Image    string `json:"image" binding:"required"`
	Force    bool   `json:"force"`
	Port     int    `json:"port" binding:"required"`
}

func (h *RegisterHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if !validIDRegex.MatchString(req.OwnerID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "owner_id must contain only lowercase letters and numbers"})
		return
	}
	if !validIDRegex.MatchString(req.WorkerID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "worker_id must contain only lowercase letters and numbers"})
		return
	}

	worker := &Worker{
		WorkerID: req.WorkerID,
		OwnerID:  req.OwnerID,
		Image:    req.Image,
		Port:     req.Port,
	}

	// 先检查冲突和镜像变化
	imageChanged, err := h.store.Set(worker)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	// 只有镜像变化时才通知 k8s
	if (imageChanged || req.Force) && h.k8sClient != nil {
		if err := h.k8sClient.Deploy(worker); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "k8s deploy failed: " + err.Error(),
			})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"status":        "ok",
		"image_changed": imageChanged,
	})
}

func (h *RegisterHandler) Delete(c *gin.Context) {
	domain := c.Param("domain")
	if domain == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "domain required"})
		return
	}

	worker, ok := h.store.Get(domain)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "worker not found"})
		return
	}

	if h.k8sClient != nil {
		if err := h.k8sClient.Delete(worker); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "k8s delete failed: " + err.Error(),
			})
			return
		}
	}

	h.store.Delete(domain)
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *RegisterHandler) List(c *gin.Context) {
	workers := h.store.List()
	c.JSON(http.StatusOK, workers)
}
