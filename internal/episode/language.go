package episode

import "strings"

// Language represents a supported translation language.
type Language struct {
	// Name is the human-readable identifier used in CLI flags and help text.
	Name string
	// Code is the value sent to the Podwise API as the translation parameter.
	Code string
}

// SupportedLanguages lists the translation languages available through the API.
var SupportedLanguages = []Language{
	{Name: "Chinese", Code: "zh"},
	{Name: "Traditional-Chinese", Code: "zh-TW"},
	{Name: "English", Code: "en"},
	{Name: "Japanese", Code: "ja"},
	{Name: "Korean", Code: "ko"},
}

// LookupLanguage returns the Language whose Name matches s (case-insensitive).
// The second return value is false when no match is found.
func LookupLanguage(s string) (Language, bool) {
	for _, lang := range SupportedLanguages {
		if strings.EqualFold(lang.Name, s) {
			return lang, true
		}
	}
	return Language{}, false
}

// LanguageNames returns the Name field of every supported language, in order.
// Useful for building flag usage strings and validation error messages.
func LanguageNames() []string {
	names := make([]string, len(SupportedLanguages))
	for i, lang := range SupportedLanguages {
		names[i] = lang.Name
	}
	return names
}
