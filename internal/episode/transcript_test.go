package episode

import (
	"testing"
)

// ─── endsWithSentenceTerminator ───────────────────────────────────────────────

func TestEndsWithSentenceTerminator(t *testing.T) {
	cases := []struct {
		input string
		want  bool
	}{
		// ASCII terminators
		{"Hello world.", true},
		{"Really?", true},
		{"Stop!", true},
		{"Pause…", true},
		{"Done;", true},
		// CJK full-width terminators
		{"你好。", true},
		{"真的吗？", true},
		{"太好了！", true},
		{"继续；", true},
		// Non-terminators
		{"Hello world", false},
		{"中间逗号,", false},
		{"trailing space.  ", true}, // trimmed
		{"", false},
		{"   ", false},
	}

	for _, tc := range cases {
		got := endsWithSentenceTerminator(tc.input)
		if got != tc.want {
			t.Errorf("endsWithSentenceTerminator(%q) = %v, want %v", tc.input, got, tc.want)
		}
	}
}

// ─── MergeSegments ────────────────────────────────────────────────────────────

// seg is a convenience constructor for test segments.
func seg(speaker, content string, startMs, endMs float64) Segment {
	return Segment{
		Speaker: speaker,
		Content: content,
		Start:   startMs,
		End:     endMs,
	}
}

func TestMergeSegments_Empty(t *testing.T) {
	if got := MergeSegments(nil, 60_000); got != nil {
		t.Errorf("expected nil, got %v", got)
	}
	if got := MergeSegments([]Segment{}, 60_000); got != nil {
		t.Errorf("expected nil for empty slice, got %v", got)
	}
}

func TestMergeSegments_Single(t *testing.T) {
	in := []Segment{seg("Alice", "Hello.", 0, 1000)}
	out := MergeSegments(in, 60_000)
	if len(out) != 1 {
		t.Fatalf("expected 1 segment, got %d", len(out))
	}
	if out[0].Content != "Hello." {
		t.Errorf("unexpected content: %q", out[0].Content)
	}
}

// Same speaker, no sentence terminator, within max duration → all merged into one.
func TestMergeSegments_SameSpeakerNoPunctuation(t *testing.T) {
	in := []Segment{
		seg("Alice", "one", 0, 1000),
		seg("Alice", "two", 1000, 2000),
		seg("Alice", "three", 2000, 3000),
	}
	out := MergeSegments(in, 60_000)
	if len(out) != 1 {
		t.Fatalf("expected 1 merged segment, got %d", len(out))
	}
	if out[0].Content != "one two three" {
		t.Errorf("unexpected content: %q", out[0].Content)
	}
	if out[0].Start != 0 {
		t.Errorf("start should be 0, got %v", out[0].Start)
	}
	if out[0].End != 3000 {
		t.Errorf("end should be 3000, got %v", out[0].End)
	}
}

// Punctuation alone does NOT trigger a flush; same-speaker segments within
// duration are merged regardless of whether they end with a terminator.
func TestMergeSegments_PunctuationNoImmediateFlush(t *testing.T) {
	in := []Segment{
		seg("Alice", "first sentence.", 0, 1000),
		seg("Alice", "second sentence.", 1000, 2000),
		seg("Alice", "third", 2000, 3000),
	}
	out := MergeSegments(in, 60_000)
	if len(out) != 1 {
		t.Fatalf("expected 1 merged segment, got %d: %+v", len(out), out)
	}
	if out[0].Content != "first sentence. second sentence. third" {
		t.Errorf("unexpected content: %q", out[0].Content)
	}
}

// All same-speaker segments within duration are merged, even when some end
// with sentence-terminating punctuation.
func TestMergeSegments_AccumulateAcrossPunctuation(t *testing.T) {
	in := []Segment{
		seg("Alice", "part one", 0, 1000),
		seg("Alice", "part two", 1000, 2000),
		seg("Alice", "part three.", 2000, 3000),
		seg("Alice", "new sentence", 3000, 4000),
	}
	out := MergeSegments(in, 60_000)
	if len(out) != 1 {
		t.Fatalf("expected 1 merged segment, got %d: %+v", len(out), out)
	}
	if out[0].Content != "part one part two part three. new sentence" {
		t.Errorf("unexpected content: %q", out[0].Content)
	}
}

// Speaker change must always start a new segment.
func TestMergeSegments_SpeakerChange(t *testing.T) {
	in := []Segment{
		seg("Alice", "hello", 0, 1000),
		seg("Bob", "world", 1000, 2000),
		seg("Bob", "again", 2000, 3000),
	}
	out := MergeSegments(in, 60_000)
	if len(out) != 2 {
		t.Fatalf("expected 2 segments, got %d: %+v", len(out), out)
	}
	if out[0].Speaker != "Alice" || out[0].Content != "hello" {
		t.Errorf("[0] unexpected: %+v", out[0])
	}
	if out[1].Speaker != "Bob" || out[1].Content != "world again" {
		t.Errorf("[1] unexpected: %+v", out[1])
	}
}

// Duration limit must split even when speaker is the same and no punctuation.
func TestMergeSegments_DurationExceeded(t *testing.T) {
	// maxDuration = 5s; third segment pushes end to 15 000 ms
	in := []Segment{
		seg("Alice", "one", 0, 2000),
		seg("Alice", "two", 2000, 4000),
		seg("Alice", "three", 10_000, 15_000), // segmentEnd(15000) - start(0) = 15000 > 5000
	}
	out := MergeSegments(in, 5_000)
	if len(out) != 2 {
		t.Fatalf("expected 2 segments, got %d: %+v", len(out), out)
	}
	if out[0].Content != "one two" {
		t.Errorf("[0] unexpected content: %q", out[0].Content)
	}
	if out[1].Content != "three" {
		t.Errorf("[1] unexpected content: %q", out[1].Content)
	}
}

// Empty-content segments are ignored entirely and don't break merging.
func TestMergeSegments_SkipsEmptyContent(t *testing.T) {
	in := []Segment{
		seg("Alice", "hello", 0, 1000),
		seg("Alice", "", 1000, 1500), // empty → skip
		seg("Alice", "world", 1500, 2000),
	}
	out := MergeSegments(in, 60_000)
	if len(out) != 1 {
		t.Fatalf("expected 1 segment, got %d: %+v", len(out), out)
	}
	if out[0].Content != "hello world" {
		t.Errorf("unexpected content: %q", out[0].Content)
	}
}

// No speaker (empty string) is treated as a valid, consistent speaker key.
func TestMergeSegments_NoSpeaker(t *testing.T) {
	in := []Segment{
		seg("", "alpha", 0, 1000),
		seg("", "beta", 1000, 2000),
	}
	out := MergeSegments(in, 60_000)
	if len(out) != 1 {
		t.Fatalf("expected 1 segment, got %d", len(out))
	}
	if out[0].Content != "alpha beta" {
		t.Errorf("unexpected content: %q", out[0].Content)
	}
	if out[0].Speaker != "" {
		t.Errorf("speaker should be empty, got %q", out[0].Speaker)
	}
}

// CJK punctuation does NOT trigger an immediate flush; segments are merged
// when speaker and duration allow.
func TestMergeSegments_CJKPunctuation(t *testing.T) {
	in := []Segment{
		seg("Alice", "这是第一句。", 0, 1000),
		seg("Alice", "这是第二句", 1000, 2000),
	}
	out := MergeSegments(in, 60_000)
	if len(out) != 1 {
		t.Fatalf("expected 1 merged segment, got %d: %+v", len(out), out)
	}
	if out[0].Content != "这是第一句。 这是第二句" {
		t.Errorf("unexpected content: %q", out[0].Content)
	}
}

// When duration is exceeded, the cut falls back to the last punctuation mark
// already in the accumulation; the leftover segments start a new group.
func TestMergeSegments_DurationCutAtLastPunctuation(t *testing.T) {
	// maxDuration = 10 s
	// acc so far: A(0-3s), B(3-6s, "."), C(6-9s)
	// next seg D(9-14s): segmentEnd(D)=14, 14-0=14 > 10 → exceeded
	// last terminator in acc is B at idx=1
	// → flush [A,B], carry [C], then append D → acc=[C,D]
	in := []Segment{
		seg("Alice", "sentence A", 0, 3000),
		seg("Alice", "sentence B.", 3000, 6000),
		seg("Alice", "sentence C", 6000, 9000),
		seg("Alice", "sentence D", 9000, 14_000),
		seg("Alice", "sentence E", 14_000, 16_000),
	}
	out := MergeSegments(in, 10_000)
	if len(out) != 2 {
		t.Fatalf("expected 2 segments, got %d: %+v", len(out), out)
	}
	if out[0].Content != "sentence A sentence B." {
		t.Errorf("[0] unexpected content: %q", out[0].Content)
	}
	if out[1].Content != "sentence C sentence D sentence E" {
		t.Errorf("[1] unexpected content: %q", out[1].Content)
	}
}

// When duration is exceeded and there is no punctuation in the accumulation,
// the entire acc is flushed as-is before starting the new segment.
func TestMergeSegments_DurationCutNoPunctuation(t *testing.T) {
	in := []Segment{
		seg("Alice", "alpha", 0, 4000),
		seg("Alice", "beta", 4000, 8000),
		seg("Alice", "gamma", 8000, 15_000), // segmentEnd=15000, 15000-0=15000 > 10000
	}
	out := MergeSegments(in, 10_000)
	if len(out) != 2 {
		t.Fatalf("expected 2 segments, got %d: %+v", len(out), out)
	}
	if out[0].Content != "alpha beta" {
		t.Errorf("[0] unexpected content: %q", out[0].Content)
	}
	if out[1].Content != "gamma" {
		t.Errorf("[1] unexpected content: %q", out[1].Content)
	}
}

// Merged segment Start/End/Speaker are taken from the correct segments.
func TestMergeSegments_TimestampInheritance(t *testing.T) {
	in := []Segment{
		seg("Bob", "a", 1000, 2000),
		seg("Bob", "b", 2000, 3000),
		seg("Bob", "c", 3000, 4500),
	}
	out := MergeSegments(in, 60_000)
	if len(out) != 1 {
		t.Fatalf("expected 1 segment, got %d", len(out))
	}
	if out[0].Start != 1000 {
		t.Errorf("start should be 1000, got %v", out[0].Start)
	}
	if out[0].End != 4500 {
		t.Errorf("end should be 4500, got %v", out[0].End)
	}
	if out[0].Speaker != "Bob" {
		t.Errorf("speaker should be Bob, got %q", out[0].Speaker)
	}
}
