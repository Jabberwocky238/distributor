package handler

import (
	"net/http"
	"regexp"

	"github.com/gin-gonic/gin"
	"github.com/jabberwocky238/distributor/k8s"
	"github.com/jabberwocky238/distributor/store"
)

var validIDRegex = regexp.MustCompile(`^[a-z0-9_]+$`)

type RegisterHandler struct {
	store     *store.MemoryStore
	k8sClient *k8s.Client
}

func NewRegisterHandler(s *store.MemoryStore, k *k8s.Client) *RegisterHandler {
	return &RegisterHandler{store: s, k8sClient: k}
}

type RegisterRequest struct {
	WorkerID string `json:"worker_id" binding:"required"`
	OwnerID  string `json:"owner_id" binding:"required"`
	Image    string `json:"image" binding:"required"`
	Port     int    `json:"port" binding:"required"`
}

func (h *RegisterHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if !validIDRegex.MatchString(req.OwnerID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "owner_id must contain only lowercase letters, numbers and underscores"})
		return
	}
	if !validIDRegex.MatchString(req.WorkerID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "worker_id must contain only lowercase letters, numbers and underscores"})
		return
	}

	worker := &store.Worker{
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
	if imageChanged && h.k8sClient != nil {
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
