package multilang

import (
	"log"

	"github.com/blevesearch/bleve/v2/analysis"
	"github.com/blevesearch/bleve/v2/registry"
	"github.com/pemistahl/lingua-go"
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
	defer func() {
		if err := recover(); err != nil {
			log.Println("ERROR: cannot register multilang analyzer:", err)
		}
	}()

	registry.RegisterCharFilter(Name, func(config map[string]interface{}, cache *registry.Cache) (analysis.CharFilter, error) {
		return &CharFilter{detector: detector, fallback: defaultLang}, nil
	})
	registry.RegisterTokenizer(Name, func(config map[string]interface{}, cache *registry.Cache) (analysis.Tokenizer, error) {
		analyzer, err := cache.AnalyzerNamed(defaultLang)
		if err != nil {
			log.Println(Name, "cannot find analyzer by name", defaultLang, err)
			return nil, err
		}
		return &Tokenizer{cache: cache, fallback: analyzer}, nil
	})
	registry.RegisterAnalyzer(Name, func(config map[string]interface{}, cache *registry.Cache) (analysis.Analyzer, error) {
		charfilter, err := cache.CharFilterNamed(Name)
		if err != nil {
			log.Println(Name, "cannot find multilang char filter", err)
			return nil, err
		}
		tokenizer, err := cache.TokenizerNamed(Name)
		if err != nil {
			log.Println(Name, "cannot find multilang tokenizer", err)
			return nil, err
		}
		analyzer := &analysis.DefaultAnalyzer{
			CharFilters: []analysis.CharFilter{charfilter},
			Tokenizer:   tokenizer,
		}

		return analyzer, nil
	})
}
