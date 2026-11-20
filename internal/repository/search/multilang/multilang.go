package multilang

import (
	"github.com/blevesearch/bleve/v2/analysis"
	"github.com/blevesearch/bleve/v2/registry"
	"github.com/etkecc/go-apm/v2"
	"github.com/pemistahl/lingua-go"
)

const (
	// Name used for all components
	Name = "multilang"
	// LangDivider is appended to the input; the detected lang name is stored after it
	LangDivider = byte('_')
)

// Register multilang analyzer
func Register(detector lingua.LanguageDetector, defaultLang string) {
	log := apm.Log()
	defer func() {
		if err := recover(); err != nil {
			log.Error("cannot register multilang analyzer", "error", err)
		}
	}()

	err := registry.RegisterCharFilter(Name, func(_ map[string]any, _ *registry.Cache) (analysis.CharFilter, error) {
		return &CharFilter{detector: detector, fallback: defaultLang}, nil
	})
	if err != nil {
		log.Error("cannot register multilang char filter", "error", err, "charfilter", Name)
	}

	err = registry.RegisterTokenizer(Name, func(_ map[string]any, cache *registry.Cache) (analysis.Tokenizer, error) {
		analyzer, err := cache.AnalyzerNamed(defaultLang)
		if err != nil {
			log.Error("cannot find analyzer by name", "error", err, "tokenizer", Name, "analyzer", defaultLang)
			return nil, err
		}
		return &Tokenizer{cache: cache, fallback: analyzer}, nil
	})
	if err != nil {
		log.Error("cannot register multilang tokenizer", "error", err, "tokenizer", Name)
	}

	err = registry.RegisterAnalyzer(Name, func(_ map[string]any, cache *registry.Cache) (analysis.Analyzer, error) {
		charfilter, err := cache.CharFilterNamed(Name)
		if err != nil {
			log.Error("cannot find multilang char filter", "error", err, "analyzer", Name)
			return nil, err
		}
		tokenizer, err := cache.TokenizerNamed(Name)
		if err != nil {
			log.Error("cannot find multilang tokenizer", "error", err, "analyzer", Name)
			return nil, err
		}
		analyzer := &analysis.DefaultAnalyzer{
			CharFilters: []analysis.CharFilter{charfilter},
			Tokenizer:   tokenizer,
		}

		return analyzer, nil
	})
	if err != nil {
		log.Error("cannot register multilang analyzer", "error", err, "analyzer", Name)
	}
}
