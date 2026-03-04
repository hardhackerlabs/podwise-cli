# podwise-cli

CLI client for [podwise.ai](https://podwise.ai) — turn any podcast episode into AI-powered insights from your terminal.

Podwise transforms hours of podcasts into summaries, outlines, transcripts, Q&A, and mind maps. This CLI lets you access those features without opening a browser, and pipe the results into your existing workflow.

## Usage

```
podwise get <episode-url> [flags]
```

### Flags (planned)

```
--type    string   output type: summary | outline | transcript | qa | mindmap  (default: summary)
--lang    string   output language: en | zh | ja | ko | fr | de | es | pt      (default: episode language)
--output  string   write to a file or directory instead of stdout
--export  string   export to: notion | obsidian | readwise | logseq
--format  string   file format: md | pdf | srt | xmind                         (default: md)
```

### Examples

```bash
# Print AI summary to stdout
podwise get https://podwise.ai/dashboard/episodes/7360326

# Get transcript in Chinese
podwise get <url> --type transcript --lang zh

# Save mind map as xmind file
podwise get <url> --type mindmap --format xmind --output ./notes/

# Export summary directly to Obsidian
podwise get <url> --export obsidian
```

## Project Structure

```
.
├── main.go
├── cmd/
│   ├── root.go          # root command
│   └── get.go           # podwise get
└── internal/
    ├── config/          # API key, output preferences
    ├── feed/            # resolve episode URLs / RSS feeds
    ├── podcast/         # core data types (Episode, Insight)
    └── storage/         # local cache
```

## Development

```bash
# run locally
go build -o podwise .
./podwise --help

# cross-platform build (macOS arm64 + Linux amd64) → outputs to dist/
goreleaser release --snapshot --clean