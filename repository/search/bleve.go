package search

import (
	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/mapping"
)

type Index struct {
	index bleve.Index
}

func getIndexMapping() mapping.IndexMapping {
	textFM := bleve.NewTextFieldMapping()
	keywordFM := bleve.NewKeywordFieldMapping()
	numericFM := bleve.NewNumericFieldMapping()

	r := bleve.NewDocumentMapping()
	r.AddFieldMappingsAt("id", keywordFM)
	r.AddFieldMappingsAt("type", keywordFM)
	r.AddFieldMappingsAt("alias", keywordFM)
	r.AddFieldMappingsAt("name", textFM)
	r.AddFieldMappingsAt("topic", textFM)
	r.AddFieldMappingsAt("avatar", keywordFM)
	r.AddFieldMappingsAt("server", keywordFM)
	r.AddFieldMappingsAt("members", numericFM)

	m := bleve.NewIndexMapping()
	m.TypeField = "type"
	m.AddDocumentMapping("room", r)
	return m
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
	index, err := bleve.New(path, getIndexMapping())
	if err != nil {
		return nil, err
	}

	return &Index{index}, err
}

// Close index
func (i *Index) Close() error {
	return i.index.Close()
}
