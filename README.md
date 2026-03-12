# podwise-cli

CLI client for [podwise.ai](https://podwise.ai) — turn any podcast episode into AI-powered insights, designed for use in AI agents and skills workflows.

Podwise transforms hours of podcasts into transcripts, summaries, outlines, Q&A, and mind maps. This CLI is purpose-built as a **tool for AI agents** — letting LLMs, skills runtimes, and automation pipelines fetch structured podcast insights without a browser or human in the loop.

> Looking for ready-to-use agent skills? Jump to [Agent Skills](#agent-skills) →

## Installation

### Automatic (Recommended)

Run the following command to install the latest version of `podwise`:

```bash
curl -sL https://raw.githubusercontent.com/hardhackerlabs/podwise-cli/main/install.sh | sh
```

### Manual (Binary)

1. Download the latest binary for your OS and architecture from [GitHub Releases](https://github.com/hardhackerlabs/podwise-cli/releases).
2. Unpack the archive (e.g., `tar -xzf podwise_linux_amd64.tar.gz`).
3. Move the `podwise` binary to a directory in your PATH, for example:
   ```bash
   mv podwise /usr/local/bin/
   ```
4. Make sure it's executable: `chmod +x /usr/local/bin/podwise`.

### From Source

If you have Go installed, you can build and install the binary directly from the source:

```bash
git clone https://github.com/hardhackerlabs/podwise-cli.git
cd podwise-cli
go build -o podwise .
# Move the binary to a directory in your PATH, e.g.,
sudo mv podwise /usr/local/bin/
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

#### Search Episodes

```bash
podwise search "Hard Fork"
```

#### Process an Episode

```bash
# Podwise episode URL (Recommended)
podwise process https://podwise.ai/dashboard/episodes/7360326

# 小宇宙 episode URL 
podwise process https://www.xiaoyuzhoufm.com/episode/abc123

# Youtube video URL
podwise process https://www.youtube.com/watch?v=d0-Gn_Bxf8s
podwise process https://youtu.be/d0-Gn_Bxf8s`,
```

#### Get Episode Details

```bash
# Get summary
podwise get summary https://podwise.ai/dashboard/episodes/7360326

# Get transcript
podwise get transcript <episode-url>
```

For more details on all available commands and flags, run:
```bash
podwise --help
```

## Agent Skills

> **Prerequisites:** Before installing skills, make sure you have completed the [Installation](#installation) and [Configuration](#configuration) steps above — the `podwise` CLI must be installed and your `api_key` must be set.

Podwise provides official agent skills out of the box. Run the following command to install the latest skills into your current directory:

```bash
curl -sL https://raw.githubusercontent.com/hardhackerlabs/podwise-cli/main/install-skills.sh | sh
```

You can also build your own skills on top of the `podwise` CLI to create custom workflows that fit your needs.