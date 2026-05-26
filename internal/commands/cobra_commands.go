package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(&cobra.Command{
		Use: "version", Short: "Show version",
		Run: func(c *cobra.Command, a []string) { fmt.Printf("luma-cli version %s\n", version) },
	})


	all := []struct{ use, short string; fn commandHandler }{
		{"bgm mix", "Mix background music", cmdBGM},
		{"cover", "Cover frame and image rendering", cmdCover},
		{"material", "Material management", cmdMaterial},
		{"project", "Project management", cmdProject},
		{"research", "Content research", cmdResearch},
		{"subtitle", "Subtitle generation", cmdSubtitle},
		{"tts", "Text to speech", cmdTTS},
		{"lipsync", "Digital human lip sync", cmdLipSync},
		{"asr", "Speech recognition", cmdASR},
		{"align", "Align subtitles to audio", cmdAlign},
		{"enhance", "Video enhancement", cmdEnhance},
		{"auth", "Authentication", cmdAuth},
		{"voice", "Voice management", cmdVoice},
		{"agent", "Run agent abilities", cmdAgent},
		{"tools", "Tool discovery", cmdTools},
		{"pip", "Picture-in-picture", cmdPIP},
		{"skills", "Skills management", cmdSkills},
		{"update", "Update CLI", cmdUpdate},
		{"social", "Social media download", cmdSocial},
		{"download", "Download files", cmdDownload},
		{"runtime", "Runtime management", cmdRuntime},
		{"resource", "Resource management", cmdResource},
		{"defaults", "Show defaults", cmdDefaults},
		{"script", "Script helpers", cmdScript},
		{"task", "Task status", cmdTask},
		{"douyin", "Douyin helpers", cmdDouyin},
		{"viral", "Viral copy helpers", cmdViral},
	}
	for _, c := range all {
		rootCmd.AddCommand(wrapCmd(c.use, c.short, c.fn))
	}
}

func wrapCmd(use, short string, fn commandHandler) *cobra.Command {
	return &cobra.Command{
		Use:                use,
		Short:              short,
		RunE:               func(c *cobra.Command, args []string) error { return fn(args) },
		DisableFlagParsing: true,
	}
}
