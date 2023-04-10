package data

import (
	"log"
	"os"
	"sync"

	"gopkg.in/yaml.v3"
)

// Store structure
type Store[K comparable, V any] struct {
	file *os.File
	data map[K]V
	mu   sync.Mutex
}

// NewStore creates new store
func NewStore[K comparable, V any](filepath string, keepInMemory bool) (*Store[K, V], error) {
	file, err := os.OpenFile(filepath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0o644)
	if err != nil {
		return nil, err
	}

	var data map[K]V
	if keepInMemory {
		data = make(map[K]V, 0)
	}

	store := &Store[K, V]{
		file: file,
		data: data,
	}
	store.warmupInMemory()

	return store, nil
}

// readFile reads file's content
func (s *Store[K, V]) readFile() map[K]V {
	datab, err := os.ReadFile(s.file.Name())
	if err != nil {
		log.Println(s.file.Name(), "cannot read", err)
		return nil
	}
	var data map[K]V
	err = yaml.Unmarshal(datab, &data)
	if err != nil {
		log.Println(s.file.Name(), "cannot unmarshal", err)
		return nil
	}

	return data
}

func (s *Store[K, V]) writeFile(data map[K]V) error {
	datab, err := yaml.Marshal(data)
	if err != nil {
		log.Println(s.file.Name(), "cannot marshal", err)
		return err
	}
	err = s.file.Truncate(0)
	if err != nil {
		log.Println(s.file.Name(), "cannot truncate", err)
		return err
	}
	_, err = s.file.Seek(0, 0)
	if err != nil {
		log.Println(s.file.Name(), "cannot seek", err)
		return err
	}

	_, err = s.file.Write(datab)
	if err != nil {
		log.Println(s.file.Name(), "cannot write", err)
		return err
	}

	return nil
}

// warmupInMemory loads content from disk into in-memory cache
func (s *Store[K, V]) warmupInMemory() {
	if s.data == nil {
		return
	}

	data := s.readFile()
	if data == nil {
		return
	}

	s.mu.Lock()
	for k, v := range data {
		s.data[k] = v
	}
	s.mu.Unlock()
}

// Add new item to the store
func (s *Store[K, V]) Add(k K, v V) {
	// if there is in-memory store - use it instead of write into actual file
	if s.data != nil {
		s.data[k] = v
		return
	}

	data := s.readFile()
	data[k] = v
	s.writeFile(data) //nolint:errcheck // handled inside
}

// Get value from the store
func (s *Store[K, V]) Get(k K) V {
	if s.data != nil {
		return s.data[k]
	}
	return s.readFile()[k]
}

// Close closed the store
func (s *Store[K, V]) Close() error {
	defer s.file.Close()
	// no in-memory store = nothing to do
	if s.data == nil {
		return nil
	}

	// dump in-memory store to the file
	data := s.readFile()
	s.mu.Lock()
	if data == nil {
		data = s.data
	} else {
		for k, v := range s.data {
			data[k] = v
		}
	}
	s.mu.Unlock()

	return s.writeFile(data)
}
