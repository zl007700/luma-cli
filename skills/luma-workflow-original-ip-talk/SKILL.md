---
name: luma-workflow-original-ip-talk
description: "Orchestrate a complete original Luma IP spoken video as pure glue: call the profile-to-script workflow, generate TTS and a digital human, align narration, render a production PPT, overlay the avatar, add subtitles, and create a cover."
---

# Luma Original IP Talk Workflow

This skill is glue only. Delegate content reasoning to `luma-content-ip-writing`, digital-human work to
`luma-digital-human`, PPT design/rendering to `luma-ppt-video`, and subtitles to `luma-subtitle`.

## Inputs

- `<profile_id>`
- target duration
- selected voice and role, or permission to choose existing assets

## Flow

### Step 1: Create Reviewed Script And Materials

Call `luma-content-ip-writing`.

Required outputs:

- accepted `07_script_<topic_id>.json`
- accepted `09_script_review_<topic_id>.json`
- `05_material_assets_<topic_id>.json`
- `materials/<topic_id>/final_assets/deliverables_manifest.json`
- selected topic ID and title

Stop if any output is missing.

### Step 2: Create Transcript

Write the exact `full_script` from the accepted script to:

- `transcript.txt`

Do not paraphrase or edit after review.

### Step 3: Select Voice And Role

Run:

```bash
luma-cli asset list voice
luma-cli asset list roles
```

Choose returned asset names. Do not invent or generate a role.

### Step 4: Generate TTS

Run:

```bash
luma-cli --json tts \
  --file transcript.txt \
  --voice <voice_name> \
  --output 09_tts.wav
```

Save the returned `audio_object_key`.

### Step 5: Generate Digital Human

Call `luma-digital-human`, or run:

```bash
luma-cli lipsync \
  --avatar <role_name> \
  --audio-key <audio_object_key> \
  --output 10_digital_human.mp4
```

Required output:

- `10_digital_human.mp4`

### Step 6: Align Narration

Run:

```bash
luma-cli asr 09_tts.wav \
  --language zh \
  --output 11_align.json
```

Required output:

- `11_align.json` with timed sentences or segments

### Step 7: Build Production PPT

Call `luma-ppt-video`.

Inputs:

- `11_align.json`
- accepted script JSON
- material assets JSON
- deliverables manifest
- `09_tts.wav`
- `10_digital_human.mp4`

Required outputs:

- `12_ppt/index.html`
- `12_ppt/config.json`
- `13_ppt.mp4`
- `14_ppt_with_avatar.mp4`

Do not use the demo PPT generator for final output.

### Step 8: Add Subtitles

Call `luma-subtitle`, or run:

```bash
luma-cli subtitle \
  14_ppt_with_avatar.mp4 \
  --transcript transcript.txt \
  --output 15_subtitle.mp4
```

If no role was used, subtitle `13_ppt.mp4`.

### Step 9: Create Cover

Run:

```bash
luma-cli cover frame \
  15_subtitle.mp4 \
  --output 16_cover.jpg
```

### Step 10: Verify

Confirm these final files exist and are non-empty:

- `15_subtitle.mp4`
- `16_cover.jpg`

Report their absolute paths and the selected topic title.
