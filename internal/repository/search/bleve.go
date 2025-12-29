package search

import (
	"context"
	"os"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/analysis/analyzer/custom"
	regexp_char_filter "github.com/blevesearch/bleve/v2/analysis/char/regexp"
	"github.com/blevesearch/bleve/v2/analysis/lang/en"
	"github.com/blevesearch/bleve/v2/analysis/tokenizer/letter"
	"github.com/blevesearch/bleve/v2/analysis/tokenizer/unicode"
	"github.com/blevesearch/bleve/v2/analysis/tokenizer/whitespace"
	"github.com/blevesearch/bleve/v2/mapping"
	"github.com/etkecc/go-apm"
	"github.com/pemistahl/lingua-go"

	"github.com/etkecc/mrs/internal/repository/search/multilang"
)

const backupSuffix = ".bak"

type Index struct {
	index bleve.Index
	path  string
}

var (
	charfilter = map[string]any{
		"regexp":  `(#|!|:)`,
		"replace": ` `,
		"type":    regexp_char_filter.Name,
	}
	commaSeparatedCharFilter = map[string]any{
		"regexp":  `[,;]`,
		"replace": ` `,
		"type":    regexp_char_filter.Name,
	}
	analyzerServer = map[string]any{
		"type": custom.Name,
		"char_filters": []any{
			`comma_separated`,
		},
		"tokenizer": whitespace.Name,
		"token_filters": []any{
			`to_lower`,
		},
	}
	analyzerID = map[string]any{
		"type": custom.Name,
		"char_filters": []any{
			`matrix_chars`,
		},
		"tokenizer": letter.Name,
		"token_filters": []any{
			`to_lower`,
		},
	}
	analyzerAlias = map[string]any{
		"type": custom.Name,
		"char_filters": []any{
			`matrix_chars`,
		},
		"tokenizer": unicode.Name,
		"token_filters": []any{
			`to_lower`,
			en.StopName,
		},
	}
)

func getIndexMapping(ctx context.Context) mapping.IndexMapping {
	log := apm.Log(ctx)
	m := bleve.NewIndexMapping()
	m.TypeField = "type"
	m.DefaultType = "room"
	err := m.AddCustomCharFilter("matrix_chars", charfilter)
	if err != nil {
		log.Error().Err(err).Msg("cannot create custom char filter")
	}
	err = m.AddCustomCharFilter("comma_separated", commaSeparatedCharFilter)
	if err != nil {
		log.Error().Err(err).Msg("cannot create comma_separated char filter")
	}

	err = m.AddCustomAnalyzer("server", analyzerServer)
	if err != nil {
		log.Error().Err(err).Msg("cannot create server analyzer")
	}

	err = m.AddCustomAnalyzer("matrix_id", analyzerID)
	if err != nil {
		log.Error().Err(err).Msg("cannot create matrix_id analyzer")
	}

	err = m.AddCustomAnalyzer("matrix_alias", analyzerAlias)
	if err != nil {
		log.Error().Err(err).Msg("cannot create matrix_alias analyzer")
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

	serverFM := bleve.NewKeywordFieldMapping()
	serverFM.Analyzer = "server"

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
	r.AddFieldMappingsAt("server", serverFM)
	r.AddFieldMappingsAt("members", numericFM)
	r.AddFieldMappingsAt("language", bleve.NewKeywordFieldMapping())
	r.AddFieldMappingsAt("room_type", bleve.NewKeywordFieldMapping()) // e.g., "m.space" for spaces, empty for rooms
	r.AddFieldMappingsAt("join_rule", textFM)                         // e.g., "public"
	r.AddFieldMappingsAt("guest_can_join", noindexFM)
	r.AddFieldMappingsAt("world_readable", noindexFM)
	m.AddDocumentMapping("room", r)

	return m
}

// NewIndex creates or opens an index
func NewIndex(path string, detector lingua.LanguageDetector, defaultLang string) (*Index, error) {
	multilang.Register(detector, defaultLang)
	i := &Index{
		path: path,
	}
	err := i.load(apm.NewContext())

	return i, err
}

// load index from path
func (i *Index) load(ctx context.Context) error {
	var index bleve.Index
	var err error
	index, err = i.loadFS(ctx)
	if err != nil {
		return err
	}
	i.index = index
	return nil
}

func (i *Index) loadFS(ctx context.Context) (bleve.Index, error) {
	index, err := bleve.Open(i.path)
	if err != nil {
		index, err = bleve.New(i.path, getIndexMapping(ctx))
		if err != nil {
			return nil, err
		}
	}
	return index, nil
}

// Swap index
func (i *Index) Swap(ctx context.Context) error {
	defer func() {
		// bleve's scorch has data race that may cause panic
		if r := recover(); r != nil {
			log := apm.Log(ctx)
			log.Error().Interface("recover", r).Msg("panic in index swap")
		}
	}()

	if err := i.index.Close(); err != nil {
		return err
	}

	log := apm.Log(ctx)
	if err := os.RemoveAll(i.path + backupSuffix); err != nil {
		log.Warn().Err(err).Msg("cannot remove index backup")
	}

	if err := os.Rename(i.path, i.path+backupSuffix); err != nil {
		log.Error().Err(err).Msg("cannot move index")
	}
	return i.load(ctx)
}

// Len returns size of the index (number of docs)
func (i *Index) Len() int {
	vUint, _ := i.index.DocCount() //nolint:errcheck // that's ok
	return int(vUint)              //nolint:gosec // that's ok
}

// Close index
func (i *Index) Close() error {
	return i.index.Close()
}
