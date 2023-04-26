package multilang

import (
	"strings"

	"github.com/pemistahl/lingua-go"

	"gitlab.com/etke.cc/mrs/api/utils"
)

// CharFilter detects input language and appends it to the input bytes
type CharFilter struct {
	detector lingua.LanguageDetector
	fallback string
}

// Filter detects input language and appends it to the end of the input bytes
func (c *CharFilter) Filter(input []byte) []byte {
	detectedLang := c.fallback
	if len(input) > 0 {
		lang, _ := utils.DetectLanguage(c.detector, string(input))
		if lang != utils.UnknownLang {
			detectedLang = strings.ToLower(lang)
		}
	}
	input = append(input, LangDivider)
	input = append(input, []byte(detectedLang)...)
	return input
}
