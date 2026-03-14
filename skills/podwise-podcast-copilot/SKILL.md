---
name: podwise-podcast-copilot
description: "End-to-end podcast and media processing with podwise CLI: search episodes, process Podwise episode URLs, YouTube videos, Xiaoyuzhou episode links, and local audio or video files, wait for processing to finish, then retrieve transcript, summary, chapters, Q&A, mind map, highlights, and keywords. Use when the user asks to process a podcast, summarize an episode, extract subtitles or a transcript, turn YouTube into notes, summarize a Xiaoyuzhou episode, transcribe a local recording, or extract key points from a local video or audio file."
---

# Podwise Podcast Copilot

Use this skill to turn raw podcast, video, and audio inputs into structured outputs that are easy to read, export, or reuse.

## Goals

1. Verify that `podwise` is installed and that the API key is configured.
2. Choose the correct path: `search`, `process`, or `get`.
3. Fetch AI outputs only after `process` completes successfully and reaches `done`.
4. Always return the Podwise episode URL and the current processing status with the results.

## Step 1: Check the Environment

Run:

```bash
podwise --help
podwise config show
```

If `podwise` is not installed yet, load [references/installation.md](references/installation.md) for installation and initial configuration steps.

## Step 2: Choose the Workflow

- If the user provides only an episode title or keywords, run `podwise search "<query>" --limit 10` unless the user explicitly asks for a different number of results.
- If the user provides a YouTube or Xiaoyuzhou URL, run `podwise process <url>`; Podwise will import it automatically.
- If the user provides a local audio or video file path, run `podwise process <file>`; Podwise will upload it and create an episode automatically.
- If the user provides a Podwise episode URL and processing may not be complete yet, run `podwise process <episode-url>`.
- If the user wants a specific artifact for an already processed episode, run `podwise get <type> <episode-url>` directly.

## Step 3: Run the Commands

### Search for Episodes

```bash
podwise search "Hard Fork" --limit 10
podwise search "AI agent" --json
podwise search "AI agent" --limit 10 --json
```

Default to `--limit 10` for search results unless the user explicitly requests a different limit.

Use `--json` when the output will be parsed by another tool or step.

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

`process` automatically polls the processing status until it finishes and exits.

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

- "Process this YouTube video and give me the transcript and summary" -> `process` + `get transcript` + `get summary`
- "Find a few podcast episodes about this topic" -> `search "<topic>" --limit 10`
- If the user explicitly asks for a different number of search results, use that number instead of `10`.
- "Export subtitles" -> `get transcript --format srt` or `get transcript --format vtt`
- "Give me a structured recap" -> `get summary` + `get chapters` + `get highlights` + `get keywords`
- "Transcribe this local recording/video and extract the key points" -> `process <file>` + `get transcript` + `get summary`

## One-Step Script

Use [scripts/podwise_pipeline.sh](scripts/podwise_pipeline.sh) to automate the full flow:

1. Process the input.
2. Export transcript, summary, chapters, Q&A, mind map, highlights, and keywords.

Run:

```bash
bash scripts/podwise_pipeline.sh "<episode-url|video-url|local-file-path>" "<output-dir>"
```

Default output directory: `./podwise-output`.

Optional environment variables, mainly useful for local file uploads:

```bash
PODWISE_PROCESS_TITLE="Weekly Meeting Recording" PODWISE_PROCESS_HOTWORDS="Podwise,Agent,ASR" \
bash scripts/podwise_pipeline.sh "./meeting.m4a" "./podwise-output"
```

## Common Failure Cases

- If `podwise` is missing or not configured correctly, stop immediately and tell the user to fix the CLI setup first.
- If a local file does not exist or the extension is unsupported, stop and ask for a valid path or supported media format.

Load [references/installation.md](references/installation.md) when the user needs help installing the CLI or setting the API key.

## Output Contract

Always include:

1. The resolved Podwise episode URL.
2. The current processing status.
3. The requested artifacts such as summary or transcript.
4. Any unavailable artifact explicitly marked as unavailable.

Load [references/commands.md](references/commands.md) when exact command examples are needed.
Load [references/installation.md](references/installation.md) when setup or installation help is needed.
