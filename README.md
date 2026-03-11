# podwise-cli

CLI client for [podwise.ai](https://podwise.ai) — turn any podcast episode into AI-powered insights, designed for use in AI agents and skills workflows.

Podwise transforms hours of podcasts into summaries, outlines, transcripts, Q&A, and mind maps. This CLI is purpose-built as a **tool for AI agents** — letting LLMs, skills runtimes, and automation pipelines fetch structured podcast insights without a browser or human in the loop.

## Installation

Run the following command to install the latest version of `podwise`:

```bash
curl -sL https://raw.githubusercontent.com/hardhackerlabs/podwise-cli/main/install.sh | sh
```

## Configuration

First, create your [podwise.ai](https://podwise.ai/dashboard/settings/developer) API key:

```bash
# Set your API key
podwise config set api_key your-sk-xxxx

# Verify connection
podwise config show
```

The configuration is stored at `~/.config/podwise/config.toml`.

## Usage

You can search for podcast episodes or process specific episodes to get summaries and transcripts.

### Search Episodes

```bash
podwise search "Hard Fork"
```

### Process an Episode

```bash
# Podwise episode URL (Recommended)
podwise process https://podwise.ai/dashboard/episodes/7360326

# 小宇宙 episode URL 
podwise process https://www.xiaoyuzhoufm.com/episode/abc123

# Youtube video URL
podwise process https://www.youtube.com/watch?v=d0-Gn_Bxf8s
podwise process https://youtu.be/d0-Gn_Bxf8s`,
```

### Get Episode Details

```bash
# Get summary
podwise get summary http://podwise.ai/dashboard/episodes/7360326

# Get transcript
podwise get transcript <episode-url>
```

For more details on all available commands and flags, run:
```bash
podwise --help
```
