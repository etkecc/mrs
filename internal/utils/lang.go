package utils

import "github.com/pemistahl/lingua-go"

const UnknownLang = "-"

// DetectLanguage and return it's ISO 639-1 code and confidence
func DetectLanguage(detector lingua.LanguageDetector, text string) (langCode string, confidence float64) {
	cvs := detector.ComputeLanguageConfidenceValues(text)
	if len(cvs) == 0 {
		return UnknownLang, 0
	}

	var lang lingua.Language
	for _, cv := range cvs {
		if cv.Value() > confidence {
			lang = cv.Language()
			confidence = cv.Value()
		}
	}

	if confidence < 0.8 {
		return UnknownLang, 0
	}

	return lang.IsoCode639_1().String(), confidence
}
