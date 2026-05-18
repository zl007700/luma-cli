package media

import "github.com/luma-cli/lumer-cli/shortcuts/common"

// Shortcuts returns media generation and processing atomic capabilities.
func Shortcuts() []common.Shortcut {
	return []common.Shortcut{
		{
			ID:          "asr.transcribe",
			Service:     "asr",
			Description: "Transcribe a local video or audio file through cloud ASR.",
			Risk:        "write",
			Flags: []common.Flag{
				{Name: "file", Description: "Local video or audio file path.", Required: true},
				{Name: "language", Description: "Recognition language.", Default: "zh"},
			},
			Outputs:  []string{"task_id", "object_key", "text", "segments"},
			Examples: []string{"luma-cli asr input.mp4 --language zh"},
			Skills:   []string{"luma-subtitle", "luma-video-workflow"},
		},
		{
			ID:          "tts.synthesize",
			Service:     "tts",
			Description: "Synthesize speech from text using a named voice asset.",
			Risk:        "write",
			Flags: []common.Flag{
				{Name: "text", Description: "Text to synthesize.", Required: true},
				{Name: "voice", Description: "Voice friendly name or object key.", Default: "男声3"},
				{Name: "speech-rate", Description: "Speech rate multiplier.", Default: "1.1"},
			},
			Outputs:  []string{"task_id", "audio_object_key", "output_path", "output_url"},
			Examples: []string{"luma-cli tts \"你好，欢迎来到直播间\" --voice 男声3"},
			Skills:   []string{"luma-digital-human", "luma-video-workflow"},
		},
		{
			ID:          "lipsync.create",
			Service:     "lipsync",
			Description: "Create a lip-sync digital human video from an avatar and audio.",
			Risk:        "write",
			Flags: []common.Flag{
				{Name: "avatar", Description: "Avatar friendly name or object key.", Required: true},
				{Name: "audio", Description: "Local audio file path. Defaults to latest project TTS audio."},
				{Name: "output", Description: "Output video path."},
			},
			Outputs:  []string{"task_id", "output_path", "output_url"},
			Examples: []string{"luma-cli lipsync --avatar 数字人男 --audio tts_output.wav"},
			Skills:   []string{"luma-digital-human", "luma-video-workflow"},
		},
		{
			ID:          "video.enhance",
			Service:     "enhance",
			Description: "Enhance or upscale a local video.",
			Risk:        "write",
			Flags: []common.Flag{
				{Name: "video", Description: "Local video file path.", Required: true},
				{Name: "scale", Description: "Upscale factor.", Default: "2"},
				{Name: "output", Description: "Output video path."},
			},
			Outputs:  []string{"task_id", "object_key", "output_path", "output_url"},
			Examples: []string{"luma-cli enhance input.mp4 --scale 2"},
			Skills:   []string{"luma-video-workflow"},
		},
		{
			ID:          "subtitle.render",
			Service:     "subtitle",
			Description: "Generate styled subtitles and optionally burn them into a video.",
			Risk:        "write",
			Flags: []common.Flag{
				{Name: "video_or_text", Description: "Local video file path, or raw text with --text.", Required: true},
				{Name: "text", Description: "Treat positional input as raw text."},
				{Name: "output", Description: "Output video path."},
				{Name: "project", Description: "Project name for organized outputs."},
				{Name: "max-chars", Description: "Maximum characters per subtitle segment.", Default: "15"},
			},
			Outputs:  []string{"output_path", "ass_path", "segments"},
			Examples: []string{"luma-cli subtitle input.mp4 --project demo"},
			Skills:   []string{"luma-subtitle"},
		},
	}
}
