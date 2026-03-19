package render

import (
	"strings"
	"testing"
)

func TestDenestListContinuations(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		// ── items 1–9: 3-space continuation ─────────────────────────────────
		{
			name: "single item 1-9 with 3-space continuation",
			input: "1. **Speaker A** — Podcast Name\n" +
				"\n" +
				"   **00:03:41**\n" +
				"\n" +
				"   \"Quote text here.\"\n",
			want: "1. **Speaker A** — Podcast Name\n" +
				"\n" +
				"**00:03:41**\n" +
				"\n" +
				"\"Quote text here.\"\n",
		},
		{
			name: "multiple items 1-9",
			input: "1. **Speaker A** — Podcast A\n" +
				"\n" +
				"   **00:01:00**\n" +
				"\n" +
				"   \"Quote A\" [citation:1](https://example.com/1)\n" +
				"\n" +
				"2. **Speaker B** — Podcast B\n" +
				"\n" +
				"   **00:02:00**\n" +
				"\n" +
				"   \"Quote B\" [citation:2](https://example.com/2)\n",
			want: "1. **Speaker A** — Podcast A\n" +
				"\n" +
				"**00:01:00**\n" +
				"\n" +
				"\"Quote A\" [citation:1](https://example.com/1)\n" +
				"\n" +
				"2. **Speaker B** — Podcast B\n" +
				"\n" +
				"**00:02:00**\n" +
				"\n" +
				"\"Quote B\" [citation:2](https://example.com/2)\n",
		},

		// ── items 10–99: 4-space continuation ───────────────────────────────
		{
			name: "single item 10-99 with 4-space continuation",
			input: "10. **Speaker X** — Podcast X\n" +
				"\n" +
				"    **00:34:42**\n" +
				"\n" +
				"    \"Quote X\" [citation:12](https://example.com/12)\n",
			want: "10. **Speaker X** — Podcast X\n" +
				"\n" +
				"**00:34:42**\n" +
				"\n" +
				"\"Quote X\" [citation:12](https://example.com/12)\n",
		},
		{
			name: "mixed items 1-9 and 10-99",
			input: "9. **Speaker Nine** — Podcast Nine\n" +
				"\n" +
				"   **00:09:00**\n" +
				"\n" +
				"   \"Quote 9\"\n" +
				"\n" +
				"10. **Speaker Ten** — Podcast Ten\n" +
				"\n" +
				"    **00:10:00**\n" +
				"\n" +
				"    \"Quote 10\"\n",
			want: "9. **Speaker Nine** — Podcast Nine\n" +
				"\n" +
				"**00:09:00**\n" +
				"\n" +
				"\"Quote 9\"\n" +
				"\n" +
				"10. **Speaker Ten** — Podcast Ten\n" +
				"\n" +
				"**00:10:00**\n" +
				"\n" +
				"\"Quote 10\"\n",
		},

		// ── boundary: should NOT strip ───────────────────────────────────────
		{
			name:  "1-2 leading spaces are left alone",
			input: " one space\n  two spaces\n",
			want:  " one space\n  two spaces\n",
		},
		{
			name: "5-space indentation is left alone (deeper nesting)",
			input: "1. item\n" +
				"\n" +
				"     five spaces\n",
			want: "1. item\n" +
				"\n" +
				"     five spaces\n",
		},
		{
			name:  "list item first lines are not touched",
			input: "1. **Speaker** — Podcast\n2. **Speaker B** — Podcast B\n",
			want:  "1. **Speaker** — Podcast\n2. **Speaker B** — Podcast B\n",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "plain text without lists",
			input: "Just a normal paragraph.\n\nAnother paragraph.\n",
			want:  "Just a normal paragraph.\n\nAnother paragraph.\n",
		},
		{
			name: "heading and paragraph are untouched",
			input: "# Q: some question\n\n" +
				"Found 3 insights:\n",
			want: "# Q: some question\n\n" +
				"Found 3 insights:\n",
		},

		// ── boundary: 4 leading spaces followed by whitespace is left alone ──
		// (a blank continuation line "    " should not collapse to empty then
		//  re-strip; the regex requires \S after the spaces)
		{
			name:  "line of only spaces is left alone",
			input: "    \n",
			want:  "    \n",
		},

		// ── real-world snapshot ──────────────────────────────────────────────
		{
			name: "real answer excerpt items 1 and 10",
			input: "Found 20 insights:\n\n" +
				"1. **Andrej Karpathy** — Making AI accessible\n" +
				"\n" +
				"   **00:03:41**\n" +
				"\n" +
				"   \"Everyone is trying to build...\" [citation:1](https://podwise.ai/dashboard/episodes/1)\n" +
				"\n" +
				"10. **Matt Bornstein** — What Is an AI Agent?\n" +
				"\n" +
				"    **00:34:42**\n" +
				"\n" +
				"    \"I will actually bet on multimodality...\" [citation:12](https://podwise.ai/dashboard/episodes/2)\n",
			want: "Found 20 insights:\n\n" +
				"1. **Andrej Karpathy** — Making AI accessible\n" +
				"\n" +
				"**00:03:41**\n" +
				"\n" +
				"\"Everyone is trying to build...\" [citation:1](https://podwise.ai/dashboard/episodes/1)\n" +
				"\n" +
				"10. **Matt Bornstein** — What Is an AI Agent?\n" +
				"\n" +
				"**00:34:42**\n" +
				"\n" +
				"\"I will actually bet on multimodality...\" [citation:12](https://podwise.ai/dashboard/episodes/2)\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := denestListContinuations(tc.input)
			if got != tc.want {
				t.Errorf("denestListContinuations() mismatch:\n--- want ---\n%s\n--- got  ---\n%s",
					visualise(tc.want), visualise(got))
			}
		})
	}
}

// visualise replaces spaces with · and newlines with ↵\n so diffs are readable.
func visualise(s string) string {
	s = strings.ReplaceAll(s, " ", "·")
	s = strings.ReplaceAll(s, "\n", "↵\n")
	return s
}
