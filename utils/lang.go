package utils

import "github.com/pemistahl/lingua-go"

// DetectLanguage and return it's ISO 639-1 code and confidence
func DetectLanguage(detector lingua.LanguageDetector, text string) (string, float64) {
	cvs := detector.ComputeLanguageConfidenceValues(text)
	if len(cvs) == 0 {
		return "-", 0
	}

	var lang lingua.Language
	var confidence float64
	for _, cv := range cvs {
		if cv.Value() > confidence {
			lang = cv.Language()
			confidence = cv.Value()
		}
	}

	return lang.IsoCode639_1().String(), confidence
}
