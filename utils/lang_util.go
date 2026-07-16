package utils

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
