//  Copyright (c) 2017 Couchbase, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// 		http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package es

import (
	"github.com/blevesearch/bleve/v2/analysis"
	"github.com/blevesearch/bleve/v2/registry"

	"github.com/blevesearch/bleve/v2/analysis/token/lowercase"
	"github.com/blevesearch/bleve/v2/analysis/tokenizer/unicode"
)

const AnalyzerName = "es"

func AnalyzerConstructor(config map[string]interface{},
	cache *registry.Cache) (analysis.Analyzer, error) {
	unicodeTokenizer, err := cache.TokenizerNamed(unicode.Name)
	if err != nil {
		return nil, err
	}
	toLowerFilter, err := cache.TokenFilterNamed(lowercase.Name)
	if err != nil {
		return nil, err
	}
	normalizeEsFilter, err := cache.TokenFilterNamed(NormalizeName)
	if err != nil {
		return nil, err
	}
	stopEsFilter, err := cache.TokenFilterNamed(StopName)
	if err != nil {
		return nil, err
	}
	lightStemmerEsFilter, err := cache.TokenFilterNamed(LightStemmerName)
	if err != nil {
		return nil, err
	}
	rv := analysis.DefaultAnalyzer{
		Tokenizer: unicodeTokenizer,
		TokenFilters: []analysis.TokenFilter{
			toLowerFilter,
			stopEsFilter,
			normalizeEsFilter,
			lightStemmerEsFilter,
		},
	}
	return &rv, nil
}

func init() {
	err := registry.RegisterAnalyzer(AnalyzerName, AnalyzerConstructor)
	if err != nil {
		panic(err)
	}
}
