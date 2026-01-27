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
	WorkerID string          `json:"worker_id" yaml:"worker_id"`
	OwnerID  string          `json:"owner_id" yaml:"owner_id"`
	Image    string          `json:"image" yaml:"image"`
	Port     int             `json:"port" yaml:"port"`
	History  []HistoryRecord `json:"history" yaml:"-"`
}

func (w *Worker) DomainPrefix() string {
	return fmt.Sprintf("%s.%s", w.WorkerID, w.OwnerID)
}

type MemoryStore struct {
	mu      sync.RWMutex
	workers map[string]*Worker // domain prefix -> worker
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		workers: make(map[string]*Worker),
	}
}

func (s *MemoryStore) Set(w *Worker) (imageChanged bool, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 检查是否是更新
	if existing, ok := s.workers[w.DomainPrefix()]; ok {
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

	s.workers[w.DomainPrefix()] = w
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
		fmt.Printf("deleting worker: %+v\n", w)
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
