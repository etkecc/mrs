package search

import (
	"os"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/analysis/analyzer/custom"
	regexp_char_filter "github.com/blevesearch/bleve/v2/analysis/char/regexp"
	"github.com/blevesearch/bleve/v2/analysis/lang/en"
	"github.com/blevesearch/bleve/v2/analysis/tokenizer/letter"
	"github.com/blevesearch/bleve/v2/analysis/tokenizer/unicode"
	"github.com/blevesearch/bleve/v2/mapping"
	"github.com/pemistahl/lingua-go"

	"gitlab.com/etke.cc/mrs/api/repository/search/multilang"
	"gitlab.com/etke.cc/mrs/api/utils"
)

const backupSuffix = ".bak"

type Index struct {
	index bleve.Index
	path  string
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

func getIndexMapping() mapping.IndexMapping {
	m := bleve.NewIndexMapping()
	m.TypeField = "type"
	m.DefaultType = "room"
	err := m.AddCustomCharFilter("matrix_chars", charfilter)
	if err != nil {
		utils.Logger.Error().Err(err).Msg("cannot create custom char filter")
	}

	err = m.AddCustomAnalyzer("matrix_id", analyzerID)
	if err != nil {
		utils.Logger.Error().Err(err).Msg("cannot create matrix_id analyzer")
	}

	err = m.AddCustomAnalyzer("matrix_alias", analyzerAlias)
	if err != nil {
		utils.Logger.Error().Err(err).Msg("cannot create matrix_alias analyzer")
	}

	textFM := bleve.NewTextFieldMapping()
	textFM.Analyzer = multilang.Name

	// noindexFM is used for values that just need to be stored, but not analyzed or searched
	noindexFM := bleve.NewKeywordFieldMapping()
	noindexFM.Store = true
	noindexFM.Index = false
	noindexFM.IncludeInAll = false
	noindexFM.IncludeTermVectors = false

	numericFM := bleve.NewNumericFieldMapping()

	matrixIDFM := bleve.NewTextFieldMapping()
	matrixIDFM.Analyzer = "matrix_id"

	matrixAliasFM := bleve.NewTextFieldMapping()
	matrixAliasFM.Analyzer = "matrix_alias"

	r := bleve.NewDocumentMapping()
	r.AddFieldMappingsAt("id", matrixIDFM)
	r.AddFieldMappingsAt("type", noindexFM)
	r.AddFieldMappingsAt("alias", matrixAliasFM)
	r.AddFieldMappingsAt("name", textFM)
	r.AddFieldMappingsAt("topic", textFM)
	r.AddFieldMappingsAt("avatar", noindexFM)
	r.AddFieldMappingsAt("avatar_url", noindexFM)
	r.AddFieldMappingsAt("server", bleve.NewKeywordFieldMapping())
	r.AddFieldMappingsAt("members", numericFM)
	r.AddFieldMappingsAt("language", bleve.NewKeywordFieldMapping())
	m.AddDocumentMapping("room", r)

	return m
}

// NewIndex creates or opens an index
func NewIndex(path string, detector lingua.LanguageDetector, defaultLang string) (*Index, error) {
	multilang.Register(detector, defaultLang)
	i := &Index{
		path: path,
	}
	err := i.load()

	return i, err
}

// load index from path
func (i *Index) load() error {
	index, err := bleve.Open(i.path)
	if err != nil {
		index, err = bleve.New(i.path, getIndexMapping())
		if err != nil {
			return err
		}
	}
	i.index = index
	return nil
}

// Swap index
func (i *Index) Swap() error {
	if err := i.index.Close(); err != nil {
		return err
	}

	if err := os.RemoveAll(i.path + backupSuffix); err != nil {
		utils.Logger.Warn().Err(err).Msg("cannot remove index backup")
	}

	if err := os.Rename(i.path, i.path+backupSuffix); err != nil {
		utils.Logger.Error().Err(err).Msg("cannot move index")
	}
	return i.load()
}

// Len returns size of the index (number of docs)
func (i *Index) Len() int {
	vUint, _ := i.index.DocCount() //nolint:errcheck // that's ok
	return int(vUint)
}

// Close index
func (i *Index) Close() error {
	return i.index.Close()
}
