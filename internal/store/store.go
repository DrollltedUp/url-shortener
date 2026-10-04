package store

import (
	"errors"
	"sync"
)

type Store struct {
	mtx  sync.RWMutex
	data map[string]string
}

var ErrCodeTaken = errors.New("code already taken")

func NewStore() *Store {
	return &Store{
		data: make(map[string]string),
	}
}

func (s *Store) Save(code, target string) error {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	if _, ok := s.data[code]; ok {
		return ErrCodeTaken
	}

	s.data[code] = target

	return nil
}

func (s *Store) Get(code string) (string, bool) {

	s.mtx.RLock()
	defer s.mtx.RUnlock()

	str, ok := s.data[code]

	return str, ok
}
