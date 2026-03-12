# Podwise CLI 命令速查

## 1) 搜索节目

```bash
podwise search "<query>"
podwise search "<query>" --limit 20
podwise search "<query>" --json
```

## 2) 处理来源链接

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
podwise process ./meeting.wav --title "会议录音"
podwise process ./demo.mp4 --title "发布会录屏" --hotwords "Podwise,LLM,ASR"
```

## 3) 控制轮询与超时

```bash
podwise process <url> --interval 30s --timeout 30m
podwise process <url> --no-wait
```

## 4) 拉取结果内容

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

## 5) 强制刷新缓存

```bash
podwise get summary <episode-url> --refresh
podwise get transcript <episode-url> --refresh
```

## 6) 配置管理

```bash
podwise config set api_key <your-sk-xxxx>
podwise config set api_base_url https://podwise.ai/api
podwise config show
```

## 7) 常用中文需求映射

```bash
# 需求：处理链接并给摘要
podwise process "<input-url>"
podwise get summary "<resolved-episode-url>"

# 需求：结构化复盘
podwise get chapters "<resolved-episode-url>"
podwise get highlights "<resolved-episode-url>"
podwise get keywords "<resolved-episode-url>"

# 需求：导出字幕
podwise get transcript "<resolved-episode-url>" --format text
podwise get transcript "<resolved-episode-url>" --format srt
podwise get transcript "<resolved-episode-url>" --format vtt

# 需求：处理本地音视频文件
podwise process "./meeting.m4a" --title "周会录音" --hotwords "产品,路线图"
podwise get summary "<resolved-episode-url>"
podwise get transcript "<resolved-episode-url>" --format text
```
