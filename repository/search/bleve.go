package search

import (
	"log"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/analysis/analyzer/custom"
	regexp_char_filter "github.com/blevesearch/bleve/v2/analysis/char/regexp"
	"github.com/blevesearch/bleve/v2/analysis/tokenizer/letter"
	"github.com/blevesearch/bleve/v2/mapping"
)

type Index struct {
	index bleve.Index
}

var (
	charfilter = map[string]interface{}{
		"regexp":  `(#|!|:)`,
		"replace": ` `,
		"type":    regexp_char_filter.Name,
	}
	analyzer = map[string]interface{}{
		"type": custom.Name,
		"char_filters": []interface{}{
			`matrixchars`,
		},
		"tokenizer": letter.Name,
		"token_filters": []interface{}{
			`to_lower`,
		},
	}
)

func getIndexMapping() mapping.IndexMapping {
	m := bleve.NewIndexMapping()
	m.TypeField = "type"
	m.DefaultType = "room"
	err := m.AddCustomCharFilter("matrixchars", charfilter)
	if err != nil {
		log.Println("index", "cannot create custom char filter", err)
	}

	err = m.AddCustomAnalyzer("matrixuris", analyzer)
	if err != nil {
		log.Println("index", "cannot create custom analyzer", err)
	}

	textFM := bleve.NewTextFieldMapping()
	keywordFM := bleve.NewKeywordFieldMapping()
	numericFM := bleve.NewNumericFieldMapping()
	matrixURIFM := bleve.NewTextFieldMapping()
	matrixURIFM.Analyzer = "matrixuris"

	r := bleve.NewDocumentMapping()
	r.AddFieldMappingsAt("id", matrixURIFM)
	r.AddFieldMappingsAt("type", keywordFM)
	r.AddFieldMappingsAt("alias", matrixURIFM)
	r.AddFieldMappingsAt("name", textFM)
	r.AddFieldMappingsAt("topic", textFM)
	r.AddFieldMappingsAt("avatar", keywordFM)
	r.AddFieldMappingsAt("avatar_url", keywordFM)
	r.AddFieldMappingsAt("server", keywordFM)
	r.AddFieldMappingsAt("members", numericFM)
	r.AddFieldMappingsAt("language", keywordFM)
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
