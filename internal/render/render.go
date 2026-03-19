// Package render provides terminal markdown rendering via glamour.
package render

import (
	"regexp"

	"charm.land/glamour/v2"
	"github.com/charmbracelet/x/term"
)

// Markdown renders the given markdown string for the terminal using glamour.
// style must be one of the built-in glamour style names (dark, light, dracula,
// tokyo-night, ascii, notty, pink); an empty or unrecognised value falls back
// to "dark". Word-wrap adapts to the current terminal width (falls back to 100
// columns when stdout is not a TTY). If rendering fails for any reason, the
// raw markdown string is returned unchanged so output is never silently lost.
func Markdown(md, style string) string {
	if style == "" {
		style = "dark"
	}
	r, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle(style),
		glamour.WithWordWrap(termWidth()),
	)
	if err != nil {
		// Unknown style — retry with the safe default.
		r, err = glamour.NewTermRenderer(
			glamour.WithStandardStyle("dark"),
			glamour.WithWordWrap(termWidth()),
		)
		if err != nil {
			return md
		}
	}

	out, err := r.Render(md)
	if err != nil {
		return md
	}
	return out
}

// MarkdownAnswer is like Markdown but designed for AI-generated answer text
// that uses CommonMark multi-paragraph list items (continuation paragraphs
// indented with 3 or 4 spaces). Glamour's ANSI renderer collapses these
// paragraphs into a single line per list item, so we strip the indentation
// first, turning them into top-level paragraphs that glamour renders correctly.
func MarkdownAnswer(md, style string) string {
	return Markdown(denestListContinuations(md), style)
}

// listContinuationRe matches lines that start with exactly 3 or 4 spaces
// followed by a non-whitespace character — the CommonMark continuation-
// paragraph indentation for ordered list items:
//   - `1. `…`9. `   → marker is 3 chars wide → continuation needs 3 spaces
//   - `10. `…`99. ` → marker is 4 chars wide → continuation needs 4 spaces
var listContinuationRe = regexp.MustCompile(`(?m)^ {3,4}(\S)`)

// denestListContinuations removes the 3-space indentation from list item
// continuation paragraphs so they become top-level paragraphs.
func denestListContinuations(md string) string {
	return listContinuationRe.ReplaceAllString(md, "$1")
}

// termWidth returns the width of the terminal attached to stdout,
// falling back to 100 when stdout is not a TTY or the width is unknown.
func termWidth() int {
	w, _, err := term.GetSize(1) // fd 1 = stdout
	if err != nil || w <= 0 {
		return 100
	}
	return w
}
