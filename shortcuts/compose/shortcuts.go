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
		},
		{
			ID:          "cover.frame",
			Service:     "cover",
			Command:     "frame",
			Description: "Extract one cover frame from a clean visual video with ffmpeg. In workflows, use the pre-subtitle PIP/lipsync video, not a subtitled or final mixed video.",
			Risk:        "write",
			Flags: []common.Flag{
				{Name: "video", Description: "Clean input video path before burned subtitles/BGM.", Required: true},
				{Name: "time", Description: "Timestamp in seconds.", Default: "1.0"},
				{Name: "output", Description: "Output image path.", Default: "step6_cover_frame.png"},
			},
			Outputs:  []string{"output_path"},
			Examples: []string{"luma-cli cover frame step4_picture_in_picture.mp4 --time 1.0 --output cover_frame.png"},
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
		},
		{
			ID:          "cover.generate",
			Service:     "cover",
			Command:     "generate",
			Description: "Generate cover candidates from a clean visual video or image. In workflows, use the pre-subtitle PIP/lipsync video, not step5_subtitle.mp4 or a final mixed video.",
			Risk:        "network-write",
			Flags: []common.Flag{
				{Name: "source", Description: "Clean input video/image before burned subtitles/BGM.", Required: true},
				{Name: "title", Description: "Cover title.", Required: true},
				{Name: "subtitle", Description: "Cover subtitle."},
				{Name: "count", Description: "Number of cover candidates.", Default: "6"},
				{Name: "output-dir", Description: "Output cover directory.", Default: "covers"},
			},
			Outputs:  []string{"manifest_path", "cover_paths"},
			Examples: []string{"luma-cli cover generate step4_picture_in_picture.mp4 --title \"封面标题\" --subtitle \"封面副标题\" --count 12 --output-dir step7_covers"},
			Skills:   []string{"luma-workflow-viral-remix"},
		},
	}
}
