package multilang

import (
	"github.com/blevesearch/bleve/v2/analysis"
	"github.com/blevesearch/bleve/v2/registry"
	"github.com/pemistahl/lingua-go"
	"github.com/rs/zerolog"

	"github.com/etkecc/mrs/internal/utils"
)

const (
	// Name used for all components
	Name = "multilang"
	// LangDivider is a special symbol added to the end of the input
	// after that symbol detected lang name is stored
	LangDivider = byte('_')
)

// Register multilang analyzer
func Register(detector lingua.LanguageDetector, defaultLang string) {
	log := zerolog.Ctx(utils.NewContext())
	defer func() {
		if err := recover(); err != nil {
			log.Error().Any("error", err).Msg("cannot register multilang analyzer")
		}
	}()

	registry.RegisterCharFilter(Name, func(_ map[string]any, _ *registry.Cache) (analysis.CharFilter, error) {
		return &CharFilter{detector: detector, fallback: defaultLang}, nil
	})
	registry.RegisterTokenizer(Name, func(_ map[string]any, cache *registry.Cache) (analysis.Tokenizer, error) {
		analyzer, err := cache.AnalyzerNamed(defaultLang)
		if err != nil {
			log.Error().Err(err).Str("tokenizer", Name).Str("analyzer", defaultLang).Msg("cannot find analyzer by name")
			return nil, err
		}
		return &Tokenizer{cache: cache, fallback: analyzer}, nil
	})
	registry.RegisterAnalyzer(Name, func(_ map[string]any, cache *registry.Cache) (analysis.Analyzer, error) {
		charfilter, err := cache.CharFilterNamed(Name)
		if err != nil {
			log.Error().Err(err).Str("analyzer", Name).Msg("cannot find multilang char filter")
			return nil, err
		}
		tokenizer, err := cache.TokenizerNamed(Name)
		if err != nil {
			log.Error().Err(err).Str("analyzer", Name).Msg("cannot find multilang tokenizer")
			return nil, err
		}
		analyzer := &analysis.DefaultAnalyzer{
			CharFilters: []analysis.CharFilter{charfilter},
			Tokenizer:   tokenizer,
		}

		return analyzer, nil
	})
}
