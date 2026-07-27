package utils

// langCodes is the source of truth for both the supported language codes
// and the order they should be displayed in — map iteration order in Go is
// randomized, so it can't be relied on for a stable UI order.
var langCodes = []string{"en", "hi", "gu", "ta"}

var langNames = map[string]string{
	"en": "English",
	"hi": "Hindi",
	"gu": "Gujarati",
	"ta": "Tamil",
}

func LangName(code string) string {
	if name, ok := langNames[code]; ok {
		return name
	}
	return code
}

func LangCodes() []string {
	return langCodes
}
