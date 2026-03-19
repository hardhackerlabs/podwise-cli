# Podwise CLI Command Reference

## 1) Search Episodes or Podcasts

```bash
# Backward-compatible episode search shorthand
podwise search "<query>"
podwise search "<query>" --limit 10

# Explicit episode search
podwise search episode "<query>"
podwise search episode "<query>" --limit 10
podwise search episode "<query>" --limit 10 --json

# Podcast search
podwise search podcast "<query>"
podwise search podcast "<query>" --limit 10
podwise search podcast "<query>" --json
```

## 2) Discover Popular Episodes

```bash
podwise popular
podwise popular --limit 10
podwise popular --json
```

## 3) List Followed Podcast Updates

```bash
# Episodes from followed podcasts
podwise list followed-episodes
podwise list followed-episodes --date today
podwise list followed-episodes --date yesterday
podwise list followed-episodes --date 2026-03-01
podwise list followed-episodes --latest 7
podwise list followed-episodes --latest 7 --json

# Followed podcasts with recent new episodes
podwise list followed-podcasts
podwise list followed-podcasts --date today
podwise list followed-podcasts --date yesterday
podwise list followed-podcasts --date 2026-03-01
podwise list followed-podcasts --latest 14
podwise list followed-podcasts --latest 14 --json
```

## 4) Ask Questions Across Transcripts

```bash
podwise ask "the future of AI agents"
podwise ask "How does retrieval augmented generation work?" --sources
```

## 5) Process Source URLs

```bash
# Podwise episode URL
podwise process https://podwise.ai/dashboard/episodes/<id>

# Xiaoyuzhou episode URL
podwise process https://www.xiaoyuzhoufm.com/episode/<id>

# YouTube long / short URL
podwise process https://www.youtube.com/watch?v=<id>
podwise process https://youtu.be/<id>

# Local media file (.mp3 .wav .m4a .mp4 .m4v .mov .webm)
podwise process ./interview.mp3
podwise process ./meeting.wav --title "Meeting Recording"
podwise process ./demo.mp4 --title "Product Launch Screen Recording" --hotwords "Podwise,LLM,ASR"
```

## 6) Control Polling and Timeout

```bash
podwise process <url> --interval 30s --timeout 30m
podwise process <url> --no-wait
```

## 7) Fetch Result Content

```bash
podwise get transcript <episode-url>
podwise get transcript <episode-url> --format text
podwise get transcript <episode-url> --format json
podwise get transcript <episode-url> --format srt
podwise get transcript <episode-url> --format vtt
podwise get transcript <episode-url> --seconds
podwise get summary <episode-url>
podwise get qa <episode-url>
podwise get chapters <episode-url>
podwise get mindmap <episode-url>
podwise get highlights <episode-url>
podwise get keywords <episode-url>
```

## 8) Force Cache Refresh

```bash
podwise get summary <episode-url> --refresh
podwise get transcript <episode-url> --refresh
```

## 9) Configuration Management

```bash
podwise config set api_key <your-sk-xxxx>
podwise config set api_base_url https://podwise.ai/api
podwise config show
```

## 10) Common Use Case Mappings

```bash
# Use case: Discover trending episodes
podwise popular --limit 10

# Use case: Check new episodes from followed podcasts today
podwise list followed-episodes --date today

# Use case: Check which followed podcasts updated recently
podwise list followed-podcasts --latest 7

# Use case: Ask a question across podcast transcripts
podwise ask "What are the biggest risks of agentic AI?" --sources

# Use case: Process a URL and retrieve its summary
podwise process "<input-url>"
podwise get summary "<resolved-episode-url>"

# Use case: Find episodes about a topic
podwise search episode "AI agent" --limit 10

# Use case: Find podcasts by show name
podwise search podcast "Lex Fridman" --limit 10

# Use case: Structured review and retrospective
podwise get chapters "<resolved-episode-url>"
podwise get highlights "<resolved-episode-url>"
podwise get keywords "<resolved-episode-url>"

# Use case: Export transcript / subtitles
podwise get transcript "<resolved-episode-url>" --format text
podwise get transcript "<resolved-episode-url>" --format srt
podwise get transcript "<resolved-episode-url>" --format vtt

# Use case: Process a local audio/video file
podwise process "./meeting.m4a" --title "Weekly Standup Recording" --hotwords "product,roadmap"
podwise get summary "<resolved-episode-url>"
podwise get transcript "<resolved-episode-url>" --format text
```
