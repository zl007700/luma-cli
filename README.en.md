# luma-cli

[中文](./README.md) | [English](./README.en.md)

`luma-cli` is a video operations and content-production skill package for AI agents. It gives agents callable capabilities for ASR, TTS, digital-human lip sync, subtitles, enhancement, asset management, and project organization.

Install:

```bash
npm install -g @lumageo/luma-cli
```

Login:

```bash
luma-cli auth login <CARD_KEY>
luma-cli auth status
```

Discover agent-callable tools:

```bash
luma-cli tools list
luma-cli --json tools describe tts.synthesize
```

Example workflow:

```bash
luma-cli tts "hello" --voice 男声3
luma-cli lipsync --avatar 数字人男 --audio tts_output.wav --output output.mp4
luma-cli subtitle output.mp4 --output output_subtitled.mp4
luma-cli enhance output_subtitled.mp4 --scale 2
```

For security issues, see [SECURITY.md](./SECURITY.md).
