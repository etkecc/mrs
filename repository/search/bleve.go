package search

import (
	"github.com/blevesearch/bleve/v2"
)

type Index struct {
	index bleve.Index
}

// OpenIndex opens existing index file
func OpenIndex(path string) (*Index, error) {
	index, err := bleve.Open(path)
	if err != nil {
		return nil, err
	}

	return &Index{index}, err
}

// NewIndex creates a new empty index file
func NewIndex(path string) (*Index, error) {
	index, err := bleve.New(path, bleve.NewIndexMapping())
	if err != nil {
		return nil, err
	}

	return &Index{index}, err
}

// Close index
func (i *Index) Close() error {
	return i.index.Close()
}
