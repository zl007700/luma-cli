package compose

import "github.com/luma-cli/lumer-cli/shortcuts/common"

// Shortcuts returns local cross-platform ffmpeg composition abilities.
func Shortcuts() []common.Shortcut {
	return []common.Shortcut{
		{
			ID:          "bgm.mix",
			Service:     "bgm",
			Command:     "mix",
			Description: "Mix background music into a video with ffmpeg.",
			Risk:        "write",
			Flags: []common.Flag{
				{Name: "video", Description: "Input video path.", Required: true},
				{Name: "bgm", Description: "BGM local path or cached resource id.", Required: true},
				{Name: "voice-volume", Description: "Original audio volume.", Default: "1.0"},
				{Name: "bgm-volume", Description: "BGM volume.", Default: "0.25"},
				{Name: "output", Description: "Output video path.", Default: "step6_bgm.mp4"},
			},
			Outputs:  []string{"output_path", "bgm_path"},
			Examples: []string{"luma-cli bgm mix subtitle_video.mp4 --bgm bgm_warm --output bgm_video.mp4"},
			Skills:   []string{"luma-video-workflow"},
		},
		{
			ID:          "cover.frame",
			Service:     "cover",
			Command:     "frame",
			Description: "Extract one cover frame from a video with ffmpeg.",
			Risk:        "write",
			Flags: []common.Flag{
				{Name: "video", Description: "Input video path.", Required: true},
				{Name: "time", Description: "Timestamp in seconds.", Default: "1.0"},
				{Name: "output", Description: "Output image path.", Default: "step6_cover_frame.png"},
			},
			Outputs:  []string{"output_path"},
			Examples: []string{"luma-cli cover frame bgm_video.mp4 --time 1.0 --output cover_frame.png"},
			Skills:   []string{"luma-video-workflow"},
		},
		{
			ID:          "cover.render",
			Service:     "cover",
			Command:     "render",
			Description: "Render title and subtitle text onto a cover image.",
			Risk:        "write",
			Flags: []common.Flag{
				{Name: "image", Description: "Input image path.", Required: true},
				{Name: "title", Description: "Cover title.", Required: true},
				{Name: "subtitle", Description: "Cover subtitle."},
				{Name: "font", Description: "Font path or cached resource id."},
				{Name: "output", Description: "Output cover image path.", Default: "step6_cover.jpg"},
			},
			Outputs:  []string{"output_path", "meta_path"},
			Examples: []string{"luma-cli cover render cover_frame.png --title 厨房收纳 --subtitle 零散物品都归位"},
			Skills:   []string{"luma-video-workflow"},
		},
	}
}
