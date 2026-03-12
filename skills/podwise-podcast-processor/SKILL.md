---
name: podwise-podcast-processor
description: 使用 podwise 对播客、YouTube、小宇宙和本地音视频文件进行转录总结处理的全流程：搜索节目、处理链接或本地文件、轮询处理状态、获取 transcript/summary/chapters/Q&A/mind map/highlights/keywords，并导出结构化结果。用户提出“处理播客”“整理节目内容”“提取字幕”“生成摘要”“YouTube 转文字”“小宇宙总结”“本地音频转文字”“本地视频提取要点”“podcast transcript/summary”等请求时使用。
license: MIT
metadata:
  author: Podwise
  version: "1.0"
---

# Podwise 播客节目处理助手

使用这个 skill，把原始节目音视频转成可直接消费的结构化内容。

## 工作目标

1. 先确认 `podwise` 可用且 API Key 有效。
2. 根据输入选择 `search`、`process` 或 `get` 路径。
3. 只有`process`退出, 且状态到 `done` 才拉取 AI 结果。
4. 输出时始终带上 episode URL 和状态说明。

## 第一步：环境检查

运行：

```bash
podwise --help
podwise config show
```

## 第二步：选择流程

- 用户只给关键词或节目名：运行 `podwise search`。
- 用户给 YouTube / 小宇宙链接：运行 `podwise process <url>`（会自动导入）。
- 用户给本地音视频文件路径：运行 `podwise process <file>`（会自动上传并生成 episode）。
- 用户给 Podwise episode URL：如果不确定已处理完成，先 `podwise process <episode-url>`。
- 用户只要已知节目的某类结果：直接 `podwise get <type> <episode-url>`。

## 第三步：执行命令

### 搜索节目

```bash
podwise search "Hard Fork"
podwise search "AI agent" --json
```

需要给下游程序解析时，使用 `--json`。

### 处理节目或视频

```bash
podwise process https://podwise.ai/dashboard/episodes/7360326
podwise process https://www.youtube.com/watch?v=d0-Gn_Bxf8s
podwise process https://youtu.be/d0-Gn_Bxf8s
podwise process https://www.xiaoyuzhoufm.com/episode/abc123
podwise process ./interview.mp3
podwise process ./meeting.wav --title "产品复盘会"
podwise process ./demo.mp4 --title "发布会录屏" --hotwords "Podwise,LLM,ASR"
```

`process` 过程会持续自动轮询处理进度和状态，直到结束才退出返回。
本地文件可用扩展名：`.mp3 .wav .m4a .mp4 .m4v .mov .webm`。

### 获取 AI 结果

```bash
podwise get transcript <episode-url>
podwise get summary <episode-url>
podwise get qa <episode-url>
podwise get chapters <episode-url>
podwise get mindmap <episode-url>
podwise get highlights <episode-url>
podwise get keywords <episode-url>
```

注意：`get` 获取结果只能接受 podwise episode url 作为参数，不能是 Youtube 和小宇宙链接。`get` 结果将直接打印到标准输出。
本地文件输入同理：先 `process <file>`，再使用处理产出的 podwise episode URL 执行 `get`。

## 中文请求到命令的映射

- “帮我处理这个 YouTube 并输出字幕和摘要”：`process` + `get transcript` + `get summary`。
- “按主题找几期播客”：`search "<主题>" --limit <n>`。
- “导出字幕文件”：`get transcript --format srt` 或 `--format vtt`。
- “给我结构化复盘”：`get summary` + `get chapters` + `get highlights` + `get keywords`。
- “把本地录音/视频转文字并提炼重点”：`process <file>` + `get transcript` + `get summary`。

## 一键脚本（推荐）

使用 [scripts/podwise_pipeline.sh](scripts/podwise_pipeline.sh) 自动完成：

1. 处理输入（Podwise URL / YouTube URL / 小宇宙 URL / 本地音视频文件）。
2. 导出 transcript/summary/chapters/qa/mindmap/highlights/keywords。

运行：

```bash
bash scripts/podwise_pipeline.sh "<episode-url|video-url|local-file-path>" "<output-dir>"
```

默认输出目录：`./podwise-output`。
可选环境变量（仅本地文件常用）：

```bash
PODWISE_PROCESS_TITLE="会议纪要录音" PODWISE_PROCESS_HOTWORDS="Podwise,Agent,ASR" \
bash scripts/podwise_pipeline.sh "./meeting.m4a" "./podwise-output"
```

## 常见错误处理

- 找不到 `podwise` cli 工具或未正确配置，提示用户并直接终止流程执行。安装文档： https://github.com/hardhackerlabs/podwise-cli
- 本地文件不存在或扩展名不支持（仅支持 `.mp3 .wav .m4a .mp4 .m4v .mov .webm`），直接终止并提示修正输入路径/格式。

## 输出格式约定

返回结果时至少包含：

1. 解析后的 episode URL。
2. 当前处理状态。
3. 用户请求的内容块（summary/transcript 等）。
4. 缺失内容明确标记为 unavailable。

需要完整命令清单时，读取 [references/commands.md](references/commands.md)。
