package memcached

import (
	"encoding/json"

	"github.com/bradfitz/gomemcache/memcache"
)

type MemcachedRepository interface {
	Set(string, interface{}) error
	Get(string, interface{}) (bool, error)
	DeleteAll() error
}

type memcachedRepository struct {
	s *memcache.Client
}

func NewMemcachedRepository(server ...string) (MemcachedRepository, error) {
	s := memcache.New(server...)
	if err := s.Ping(); err != nil {
		return nil, err
	}

	return &memcachedRepository{
		s: s,
	}, nil
}

func (m *memcachedRepository) Set(k string, v interface{}) error {
	o, err := json.Marshal(v)
	if err != nil {
		return err
	}

	return m.s.Set(&memcache.Item{
		Key:   k,
		Value: o,
	})
}

func (m *memcachedRepository) Get(k string, v interface{}) (bool, error) {
	i, err := m.s.Get(k)
	if err != nil && err != memcache.ErrCacheMiss {
		return false, err
	}

	if err == memcache.ErrCacheMiss {
		return false, nil
	}

	return true, json.Unmarshal(i.Value, v)
}

func (m *memcachedRepository) DeleteAll() error {
	return m.s.DeleteAll()
}
