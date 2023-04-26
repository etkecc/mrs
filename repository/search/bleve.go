package search

import (
	"log"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/analysis/analyzer/custom"
	regexp_char_filter "github.com/blevesearch/bleve/v2/analysis/char/regexp"
	"github.com/blevesearch/bleve/v2/analysis/lang/en"
	"github.com/blevesearch/bleve/v2/analysis/tokenizer/letter"
	"github.com/blevesearch/bleve/v2/analysis/tokenizer/unicode"
	"github.com/blevesearch/bleve/v2/mapping"
	"github.com/pemistahl/lingua-go"

	"gitlab.com/etke.cc/mrs/api/repository/search/multilang"
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
	analyzerID = map[string]interface{}{
		"type": custom.Name,
		"char_filters": []interface{}{
			`matrix_chars`,
		},
		"tokenizer": letter.Name,
		"token_filters": []interface{}{
			`to_lower`,
		},
	}
	analyzerAlias = map[string]interface{}{
		"type": custom.Name,
		"char_filters": []interface{}{
			`matrix_chars`,
		},
		"tokenizer": unicode.Name,
		"token_filters": []interface{}{
			`to_lower`,
			en.StopName,
		},
	}
)

func getIndexMapping(detector lingua.LanguageDetector, defaultLang string) mapping.IndexMapping {
	multilang.Register(detector, defaultLang)

	m := bleve.NewIndexMapping()
	m.TypeField = "type"
	m.DefaultType = "room"
	err := m.AddCustomCharFilter("matrix_chars", charfilter)
	if err != nil {
		log.Println("index", "cannot create custom char filter", err)
	}

	err = m.AddCustomAnalyzer("matrix_id", analyzerID)
	if err != nil {
		log.Println("index", "cannot create matrix_id analyzer", err)
	}

	err = m.AddCustomAnalyzer("matrix_alias", analyzerAlias)
	if err != nil {
		log.Println("index", "cannot create matrix_alias analyzer", err)
	}

	textFM := bleve.NewTextFieldMapping()
	textFM.Analyzer = multilang.Name
	keywordFM := bleve.NewKeywordFieldMapping()
	numericFM := bleve.NewNumericFieldMapping()
	matrixIDFM := bleve.NewTextFieldMapping()
	matrixIDFM.Analyzer = "matrix_id"
	matrixAliasFM := bleve.NewTextFieldMapping()
	matrixAliasFM.Analyzer = "matrix_alias"

	r := bleve.NewDocumentMapping()
	r.AddFieldMappingsAt("id", matrixIDFM)
	r.AddFieldMappingsAt("type", keywordFM)
	r.AddFieldMappingsAt("alias", matrixAliasFM)
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
func NewIndex(path string, detector lingua.LanguageDetector, defaultLang string) (*Index, error) {
	index, err := bleve.New(path, getIndexMapping(detector, defaultLang))
	if err != nil {
		return nil, err
	}

	return &Index{index}, err
}

// Close index
func (i *Index) Close() error {
	return i.index.Close()
}
