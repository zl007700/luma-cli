package social

import "github.com/luma-cli/lumer-cli/shortcuts/common"

// Shortcuts returns social download atomic capabilities.
func Shortcuts() []common.Shortcut {
	return []common.Shortcut{
		{
			ID:          "social.download",
			Service:     "social",
			Command:     "download",
			Description: "Download a Douyin video from a share link and return the saved file path.",
			Risk:        "write",
			Flags: []common.Flag{
				{Name: "share_link", Description: "Douyin share link or /video/{id} URL.", Required: true},
				{Name: "output", Description: "Output video file path (default: derived from video title)."},
			},
			Outputs:  []string{"video_path", "title"},
			Examples: []string{"luma-cli social download \"https://v.douyin.com/xxx\" --output video.mp4"},
		},
	}
}
