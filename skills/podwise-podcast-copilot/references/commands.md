# Podwise CLI Command Reference

## 1) Search Episodes

```bash
podwise search "<query>"
podwise search "<query>" --limit 20
podwise search "<query>" --json
```

## 2) Process Source URLs

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

## 3) Control Polling and Timeout

```bash
podwise process <url> --interval 30s --timeout 30m
podwise process <url> --no-wait
```

## 4) Fetch Result Content

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

## 5) Force Cache Refresh

```bash
podwise get summary <episode-url> --refresh
podwise get transcript <episode-url> --refresh
```

## 6) Configuration Management

```bash
podwise config set api_key <your-sk-xxxx>
podwise config set api_base_url https://podwise.ai/api
podwise config show
```

## 7) Common Use Case Mappings

```bash
# Use case: Process a URL and retrieve its summary
podwise process "<input-url>"
podwise get summary "<resolved-episode-url>"

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
