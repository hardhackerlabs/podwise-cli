#!/usr/bin/env bash
set -euo pipefail

if [[ $# -lt 1 || $# -gt 2 ]]; then
  echo "Usage: $0 <episode-url|video-url|local-file-path> [output-dir]" >&2
  exit 1
fi

input="$1"
output_dir="${2:-./podwise-output}"
process_log="${output_dir}/process.log"

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "Missing required command: $1" >&2
    exit 1
  fi
}

require_cmd podwise
mkdir -p "$output_dir"

process_cmd=(podwise process "$input" --interval 30s --timeout 45m)
if [[ -n "${PODWISE_PROCESS_TITLE:-}" ]]; then
  process_cmd+=(--title "$PODWISE_PROCESS_TITLE")
fi
if [[ -n "${PODWISE_PROCESS_HOTWORDS:-}" ]]; then
  process_cmd+=(--hotwords "$PODWISE_PROCESS_HOTWORDS")
fi

echo "Processing: $input"
"${process_cmd[@]}" | tee "$process_log"

episode_url="$input"
if [[ ! "$episode_url" =~ ^https://podwise\.ai/dashboard/episodes/[0-9]+$ ]]; then
  episode_url="$(grep -Eo 'https://podwise\.ai/dashboard/episodes/[0-9]+' "$process_log" | tail -n 1 || true)"
  if [[ -z "$episode_url" ]]; then
    echo "Failed to resolve podwise episode URL from process output." >&2
    exit 1
  fi
fi

echo "Resolved episode URL: $episode_url"

echo "Fetching transcript"
podwise get transcript "$episode_url" --format text >"${output_dir}/transcript.txt"

echo "Fetching summary/chapters/qa/mindmap/highlights/keywords"
podwise get summary "$episode_url" >"${output_dir}/summary.md"
podwise get chapters "$episode_url" >"${output_dir}/chapters.md"
podwise get qa "$episode_url" >"${output_dir}/qa.md"
podwise get mindmap "$episode_url" >"${output_dir}/mindmap.md"
podwise get highlights "$episode_url" >"${output_dir}/highlights.md"
podwise get keywords "$episode_url" >"${output_dir}/keywords.md"

echo
echo "Done. Files written to: $output_dir"
echo "Process log: $process_log"
