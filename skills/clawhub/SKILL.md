---
name: podwise-podcast-copilot
description: "Podcast copilot workflows with podwise CLI: search podcasts or episodes by keyword, monitor followed shows for new releases, find current popular episodes, ask questions or extract insights by searching across podcast transcript content, process Podwise episode URLs, YouTube videos, Xiaoyuzhou episode links, and local audio or video files, then retrieve transcript, summary, chapters, Q&A, mind map, highlights, and keywords. Use when the user asks to find podcasts or episodes by name/topic/keyword, discover hot shows, check subscription updates, answer a question from podcast transcripts, synthesize transcript-grounded insights across podcasts, summarize an episode, extract subtitles or a transcript, turn YouTube into notes, summarize a Xiaoyuzhou episode, transcribe a local recording, or extract key points from a local video or audio file."
---

# Podwise Podcast Copilot

Use this skill to turn raw podcast, video, and audio inputs into structured outputs that are easy to read, export, or reuse.

## Goals

1. Verify that `podwise` is installed and that the API key is configured.
2. Distinguish keyword discovery from transcript-grounded answering, then choose the correct path: `search`, `list`, `popular`, `ask`, `process`, or `get`.
3. Ask for user confirmation before any `process` command because it consumes quota/credits, unless the user explicitly asked to proceed without pausing.
4. Fetch AI outputs only after `process` completes successfully and reaches `done`.
5. Return output in the shape that matches the workflow: discovery lists, transcript-grounded answers, or episode artifacts.

## Step 1: Check the Environment

Run:

```bash
podwise --help
podwise config show
```

Treat `podwise --help` and `podwise <command> --help` as the source of truth for exact flags, subcommands, and examples.

When a workflow needs command details beyond this skill, inspect help progressively:

```bash
podwise search --help
podwise search episode --help
podwise search podcast --help
podwise ask --help
podwise list --help
podwise process --help
podwise get --help
podwise get transcript --help
```

## Step 2: Choose the Workflow

### Discover Content, Podcasts, Episodes

- Use `search` when the user wants matching podcasts or episodes returned as results to browse.
- Use `ask` when the user wants an answer, synthesis, or insights derived from transcript content across podcasts.
- If the user wants to find episodes by keyword, or title, run `podwise search episode <title>`.
- If the user wants to find shows or feeds by podcast name or keyword, run `podwise search podcast <name>`.
- If the user asks what podcasts say about a topic, asks for takeaways, or wants transcript-grounded analysis, run `podwise ask <message>`.
- If the user wants to discover what is currently trending across Podwise, run `podwise popular`.
- If the user wants to check updates from podcasts they already follow, run `podwise list episodes` or `podwise list podcasts`.

### Process an Episode

- Before running any `process` command, ask whether the user wants to spend quota/credits on processing.
- Skip that confirmation only when the user already made the intent explicit, for example by saying to process it now or not to pause for confirmation.
- If the user provides a YouTube or Xiaoyuzhou URL and confirms processing, run `podwise process <url>`; Podwise will import it automatically.
- If the user provides a local audio or video file path and confirms processing, run `podwise process <file>`; Podwise will upload it and create an episode automatically.
- If the user provides a Podwise episode URL and processing may not be complete yet, ask for confirmation first, then run `podwise process <episode-url>`.

### Retrieve AI Results

- If the user wants a specific artifact for an already processed episode, run `podwise get <type> <episode-url>` directly.

## Step 3: Run the Commands

### Search for Episodes or Podcasts

```bash
podwise search episode "AI agent" --limit 10 --json
podwise search podcast "Lex Fridman" --limit 10 --json
```

Use `--json` when the output will be parsed by another tool or step.

Use `search` when the user wants a list of matching podcasts or episodes, not a synthesized answer.

Use episode search when the user is looking for a specific episode, or episode title keywords.

Use podcast search when the user is looking for a show by name, wants candidate podcasts to subscribe to, or is trying to identify the podcast behind an episode.

Multiple words are treated as a single phrase; pass them together as one quoted query or as separate arguments.

### Discover Popular Episodes

```bash
podwise popular --json
```

Use `popular` when the user wants hot, trending, or currently popular episodes rather than keyword matches.

### Check Followed Podcast Updates

```bash
podwise list episodes --json
podwise list episodes --json --date today
podwise list episodes --json --date yesterday
podwise list episodes --json --date 2026-03-01 
podwise list episodes --json --latest 7
podwise list podcasts --json
podwise list podcasts --json --date today
podwise list podcasts --json --latest 14
```

Use `list episodes` to see new episodes published by shows the user follows.

Use `list podcasts` to see which followed podcasts have had recent updates.

### Ask Questions Across Podcast Transcripts

```bash
podwise ask "the future of AI agents"
```

Use `ask` when the user wants a synthesized answer, takeaways, comparison, or insights grounded in podcast transcripts rather than a list of matching episodes.

Do not use `ask` when the user only wants to locate podcasts or episodes by keyword; use `search` for that.

`ask` may take up to about `60s` because Podwise searches transcripts and generates an answer.

### Process an Episode, Video, or Local File

```bash
podwise process https://podwise.ai/dashboard/episodes/7360326
podwise process https://www.youtube.com/watch?v=d0-Gn_Bxf8s
podwise process https://youtu.be/d0-Gn_Bxf8s
podwise process https://www.xiaoyuzhoufm.com/episode/abc123
podwise process ./interview.mp3
podwise process ./meeting.wav --title "Product Review Meeting"
podwise process ./demo.mp4 --title "Launch Demo Recording" --hotwords "Podwise,LLM,ASR"
```

`process` always waits for processing to finish before it exits.

Because `process` consumes quota/credits, do not run it silently. Confirm with the user first unless they explicitly asked to proceed without pausing.

Supported local file extensions: `.mp3 .wav .m4a .mp4 .m4v .mov .webm`.

### Retrieve AI Outputs

```bash
podwise get transcript <episode-url>
podwise get summary <episode-url>
podwise get qa <episode-url>
podwise get chapters <episode-url>
podwise get mindmap <episode-url>
podwise get highlights <episode-url>
podwise get keywords <episode-url>
```

`podwise get` accepts only a Podwise episode URL. Do not pass a YouTube URL, a Xiaoyuzhou URL, or a local file path to `get`.

For local files, follow the same pattern: run `process <file>` first, then use the resulting Podwise episode URL with `get`.

## User Request to Command Mapping

- "What do podcasts say about this topic?" -> `ask`
- "Summarize the main insights from podcast transcripts on this topic" -> `ask`
- "Find podcast episodes about this topic" -> `search episode`
- "Find a few episodes on this topic, then I will choose what to read" -> `search episode`
- "Find the podcast/show named Lex Fridman" -> `search podcast`
- "Show me what's trending on Podwise right now" -> `popular`
- "Show today's new episodes from podcasts I follow" -> `list episodes`
- "Show which followed podcasts published recently" -> `list podcasts`
- "Process this YouTube video and return the transcript and summary" -> confirm first, then `process` + `get transcript` + `get summary`
- "Export this episode as subtitles" -> `get transcript --format srt` or `get transcript --format vtt`
- "Give me an episode recap with summary, chapters, highlights, and keywords" -> `get summary` + `get chapters` + `get highlights` + `get keywords`
- "Transcribe this local audio/video file and summarize the key points" -> confirm first, then `process <file>` + `get transcript` + `get summary`

## Common Failure Cases

- If `podwise` is missing or not configured correctly, stop immediately and tell the user to fix the CLI setup first.
- If a local file does not exist or the extension is unsupported, stop and ask for a valid path or supported media format.
- If the workflow requires `process` and the user has not agreed to spend quota/credits, pause and ask for confirmation before continuing.
- If `ask` returns an error or plan-limit issue, report it directly; do not fabricate an answer.

## Output Contract

For `process` and `get` workflows, include:

1. The resolved Podwise episode URL.
2. The current processing status.
3. The requested artifacts such as summary or transcript.
4. Any unavailable artifact explicitly marked as unavailable.

For `ask` workflows, include:

1. The answer text.
2. Source excerpts and episode links when `--sources` is used.
3. A note if the answer is uncited because sources were not requested.

For `search`, `list`, and `popular` workflows, return the result list in a scannable order and preserve the CLI's intent: discovery, updates, or trending content.

For `search` workflows specifically, return matching podcasts or episodes rather than a transcript-synthesized answer.