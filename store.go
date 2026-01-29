package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

type HistoryRecord struct {
	Image     string    `json:"image"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (w *Worker) Name() string {
	return fmt.Sprintf("%s-%s", w.WorkerID, w.OwnerID)
}

func (w *Worker) PodName() string {
	return fmt.Sprintf("worker-%s", w.Name())
}

type MemoryStore struct {
	mu       sync.RWMutex
	workers  map[string]*Worker // name -> worker
	filePath string             // 持久化文件路径
}

func NewMemoryStore(filePath string) (*MemoryStore, error) {
	if filePath == "" {
		filePath = "/data/workers.json"
	}

	s := &MemoryStore{
		workers:  make(map[string]*Worker),
		filePath: filePath,
	}

	// 尝试从文件加载
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *MemoryStore) Set(w *Worker) (imageChanged bool, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 检查是否是更新
	if existing, ok := s.workers[w.Name()]; ok {
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

	s.workers[w.Name()] = w
	s.save() // 保存到文件
	return true, nil
}

func (s *MemoryStore) Get(name string) (*Worker, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	w, ok := s.workers[name]
	return w, ok
}

func (s *MemoryStore) Delete(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if w, ok := s.workers[name]; ok {
		fmt.Printf("deleting worker: %+v\n", w)
	}
	delete(s.workers, name)
	s.save() // 保存到文件
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

// save 保存到文件 (调用者必须持有锁)
func (s *MemoryStore) save() error {
	data, err := json.MarshalIndent(s.workers, "", "  ")
	if err != nil {
		fmt.Printf("failed to marshal workers: %v\n", err)
		return err
	}

	if err := os.WriteFile(s.filePath, data, 0644); err != nil {
		fmt.Printf("failed to write workers file: %v\n", err)
	}
	return nil
}

// load 从文件加载 (调用者必须持有锁)
func (s *MemoryStore) load() error {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		if !os.IsNotExist(err) {
			fmt.Printf("not found to read workers file: %v\n", err)
			// create file
			err = s.save()
			return err
		} else {
			return err
		}
	}

	if err := json.Unmarshal(data, &s.workers); err != nil {
		fmt.Printf("failed to unmarshal workers: %v\n", err)
	}
	return nil
}
