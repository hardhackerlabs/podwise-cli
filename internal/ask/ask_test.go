package ask

import (
	"strings"
	"testing"
)

// stubResult returns a Result whose Sources have EpSeq 1001, 1002, 1003.
func stubResult() *Result {
	return &Result{
		Answer: "",
		Sources: []Source{
			{EpSeq: 1001},
			{EpSeq: 1002},
			{EpSeq: 1003},
		},
	}
}

func TestLinkCitations(t *testing.T) {
	r := stubResult()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "standard format",
			input: "See [citation:1] for details.",
			want:  "See [citation:1](https://podwise.ai/dashboard/episodes/1001) for details.",
		},
		{
			name:  "space before number",
			input: "[citation: 2]",
			want:  "[citation:2](https://podwise.ai/dashboard/episodes/1002)",
		},
		{
			name:  "space around colon",
			input: "[citation : 3]",
			want:  "[citation:3](https://podwise.ai/dashboard/episodes/1003)",
		},
		{
			name:  "spaces around all tokens",
			input: "[ citation : 1 ]",
			want:  "[citation:1](https://podwise.ai/dashboard/episodes/1001)",
		},
		{
			name:  "multiple citations in sequence",
			input: "[citation:1][citation:2]",
			want:  "[citation:1](https://podwise.ai/dashboard/episodes/1001)[citation:2](https://podwise.ai/dashboard/episodes/1002)",
		},
		{
			name:  "multiple citations with text between",
			input: "Both [citation:1] and [citation:3] agree.",
			want:  "Both [citation:1](https://podwise.ai/dashboard/episodes/1001) and [citation:3](https://podwise.ai/dashboard/episodes/1003) agree.",
		},
		{
			name:  "out-of-range index left unchanged",
			input: "[citation:99]",
			want:  "[citation:99]",
		},
		{
			name:  "index zero left unchanged",
			input: "[citation:0]",
			want:  "[citation:0]",
		},
		{
			name:  "no citations",
			input: "Plain answer with no citations.",
			want:  "Plain answer with no citations.",
		},
		{
			name:  "mixed valid and out-of-range",
			input: "[citation:2] and [citation:5]",
			want:  "[citation:2](https://podwise.ai/dashboard/episodes/1002) and [citation:5]",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := r.linkCitations(tc.input)
			if got != tc.want {
				t.Errorf("\ninput: %s\n  got: %s\n want: %s", tc.input, got, tc.want)
			}
		})
	}
}

func TestLinkCitations_NormalisesWhitespace(t *testing.T) {
	r := stubResult()
	// All whitespace variants should produce the same canonical [citation:N](...) output.
	variants := []string{
		"[citation:1]",
		"[ citation:1]",
		"[citation:1 ]",
		"[ citation : 1 ]",
		"[citation : 1]",
		"[ citation: 1 ]",
	}
	want := "[citation:1](https://podwise.ai/dashboard/episodes/1001)"
	for _, v := range variants {
		got := r.linkCitations(v)
		if !strings.Contains(got, want) {
			t.Errorf("variant %q: got %q, want it to contain %q", v, got, want)
		}
	}
}
