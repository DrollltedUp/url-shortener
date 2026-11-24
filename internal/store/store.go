package store

import (
	"context"
	"sync"
)

type DataStore struct {
	mtx  sync.RWMutex
	data map[string]string
}

func NewStore() *DataStore {
	return &DataStore{
		data: make(map[string]string),
	}
}

func (s *DataStore) Save(_ context.Context, code, target string) error {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	if _, ok := s.data[code]; ok {
		return ErrCodeTaken
	}

	s.data[code] = target

	return nil
}

func (s *DataStore) Get(_ context.Context, code string) (string, bool) {

	s.mtx.RLock()
	defer s.mtx.RUnlock()

	str, ok := s.data[code]

	return str, ok
}
