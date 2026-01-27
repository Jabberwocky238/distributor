package store

import (
	"fmt"
	"sync"
	"time"
)

type HistoryRecord struct {
	Image     string    `json:"image"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Worker struct {
	Domain   string          `json:"domain" yaml:"domain"`
	WorkerID string          `json:"worker_id" yaml:"worker_id"`
	OwnerID  string          `json:"owner_id" yaml:"owner_id"`
	Image    string          `json:"image" yaml:"image"`
	Port     int             `json:"port" yaml:"port"`
	History  []HistoryRecord `json:"history" yaml:"-"`
}

type MemoryStore struct {
	mu      sync.RWMutex
	workers map[string]*Worker // domain -> worker
	index   map[string]string  // owner_id:worker_id -> domain
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		workers: make(map[string]*Worker),
		index:   make(map[string]string),
	}
}

func (s *MemoryStore) Set(w *Worker) (imageChanged bool, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := w.OwnerID + ":" + w.WorkerID

	// 检查 owner_id:worker_id 是否已存在于其他 domain
	if existingDomain, ok := s.index[key]; ok && existingDomain != w.Domain {
		return false, fmt.Errorf("worker %s/%s already exists at domain %s", w.OwnerID, w.WorkerID, existingDomain)
	}

	// 检查是否是更新
	if existing, ok := s.workers[w.Domain]; ok {
		if existing.Image == w.Image {
			return false, nil // 镜像未变化
		}
		w.History = existing.History
	}

	// 添加历史记录
	w.History = append(w.History, HistoryRecord{
		Image:     w.Image,
		UpdatedAt: time.Now(),
	})

	s.workers[w.Domain] = w
	s.index[key] = w.Domain
	return true, nil
}

func (s *MemoryStore) Get(domain string) (*Worker, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	w, ok := s.workers[domain]
	return w, ok
}

func (s *MemoryStore) Delete(domain string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if w, ok := s.workers[domain]; ok {
		key := w.OwnerID + ":" + w.WorkerID
		delete(s.index, key)
	}
	delete(s.workers, domain)
}

func (s *MemoryStore) List() []*Worker {
	s.mu.RLock()
	defer s.mu.RUnlock()
	list := make([]*Worker, 0, len(s.workers))
	for _, w := range s.workers {
		list = append(list, w)
	}
	return list
}
