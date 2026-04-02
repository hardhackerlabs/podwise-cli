package episode

import (
	"fmt"
	"strings"
)

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

// ResolveLangName validates name (case-insensitive) and returns the API-accepted
// language name with hyphens replaced by spaces (e.g. "Traditional-Chinese" →
// "Traditional Chinese"). Returns an empty string unchanged. Returns an error
// listing valid names when the name is not recognised.
func ResolveLangName(name string) (string, error) {
	if name == "" {
		return "", nil
	}
	lang, ok := LookupLanguage(name)
	if !ok {
		return "", fmt.Errorf("unsupported language %q: available languages are %s", name, strings.Join(LanguageNames(), ", "))
	}
	return strings.ReplaceAll(lang.Name, "-", " "), nil
}

// ResolveLangCode validates name (case-insensitive) and returns the BCP-47
// language code used by export APIs (e.g. "Chinese" → "zh", "Traditional-Chinese"
// → "zh-TW"). Returns an empty string unchanged. Returns an error listing valid
// names when the name is not recognised.
func ResolveLangCode(name string) (string, error) {
	if name == "" {
		return "", nil
	}
	lang, ok := LookupLanguage(name)
	if !ok {
		return "", fmt.Errorf("unsupported language %q: available languages are %s", name, strings.Join(LanguageNames(), ", "))
	}
	return lang.Code, nil
}
