---
name: luma-workflow-original-ip-talk
description: "Orchestrate a complete original Luma spoken video from a profile by running the original-script pipeline, then generating TTS, digital human video, PPT-style visuals, subtitles, and a cover."
metadata:
  category: "workflow"
  entrypoint: true
  requires:
    bins: ["luma-cli"]
  relatedSkills: ["luma-shared", "luma-original-script", "luma-digital-human", "luma-ppt-video", "luma-subtitle"]
---

# Luma Original IP Talk Workflow

This skill is glue only. Content reasoning belongs to `luma-original-script`; media execution belongs
to `luma-digital-human`, `luma-ppt-video`, and `luma-subtitle`.

## Inputs

- `profile_id`
- optional `topic_hint`
- selected voice and role, or permission to choose existing assets

## Flow

### 1. Produce The Script

Run the original script pipeline:

```bash
luma-cli --json content original-script run \
  --profile <profile_id> \
  --topic-hint "<optional direction>" \
  --output runs/<run_id>
```

Required files:

- `runs/<run_id>/final.md`
- `runs/<run_id>/final_review.json`
- `runs/<run_id>/run_state.json`

Continue when `run_state.status` is `done`. If `promotion.status` is `blocked`, tell the user the
script was produced but not auto-promoted, then ask before spending media credits unless the user
already requested a best-effort video.

### 2. Create Transcript

Copy the exact contents of `final.md` to:

- `transcript.txt`

Do not paraphrase after review unless the user explicitly asks for manual editing.

### 3. Select Voice And Role

Inspect available assets:

```bash
luma-cli asset list voice
luma-cli asset list roles
```

Use returned asset names. Do not invent or generate a role.

### 4. Generate TTS

```bash
luma-cli --json tts \
  --file transcript.txt \
  --voice <voice_name> \
  --output 09_tts.wav
```

Save the returned `audio_object_key`.

### 5. Generate Digital Human

```bash
luma-cli lipsync \
  --avatar <role_name> \
  --audio-key <audio_object_key> \
  --output 10_digital_human.mp4
```

Required output:

- `10_digital_human.mp4`

### 6. Align Narration

```bash
luma-cli asr 09_tts.wav \
  --language zh \
  --output 11_align.json
```

Required output:

- `11_align.json`

### 7. Build PPT-Style Visuals

Call `luma-ppt-video` with:

- `11_align.json`
- `transcript.txt`
- optional `10_digital_human.mp4`

Expected outputs:

- `12_ppt/index.html`
- `12_ppt/config.json`
- `13_ppt.mp4`
- optional `14_ppt_with_avatar.mp4`

### 8. Add Subtitles

```bash
luma-cli subtitle \
  14_ppt_with_avatar.mp4 \
  --transcript transcript.txt \
  --output 15_subtitle.mp4
```

If no avatar overlay was produced, subtitle `13_ppt.mp4`.

### 9. Create Cover

```bash
luma-cli cover frame \
  15_subtitle.mp4 \
  --output 16_cover.jpg
```

## Verify

Confirm final files exist and are non-empty:

- `15_subtitle.mp4`
- `16_cover.jpg`

Report absolute paths, the original script run directory, and whether `promotion.status` was
`promoted` or `blocked`.
