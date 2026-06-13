package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/luma-cli/lumer-cli/internal/output"
)

func init() {
	rootCmd.AddCommand(
		&cobra.Command{Use: "version", Short: "Show version",
			Run: func(c *cobra.Command, a []string) { printVersion() }},
	)
	// Core media commands
	addTTS()
	addLipSync()
	addASR()
	addEnhance()
	addImage()
	addVideo()
	addSubtitle()

	// Workflow commands
	addBGM()
	addCover()
	addPIP()

	// Resource commands
	addMaterial()
	addAsset()
	addAssets()
	addAvatarPersona()
	addContent()
	addResearch()
	addURLRead()
	addProfile()
	addProject()
	addAuth()
	addVoice()
	addAgent()
	addTools()
	addSkills()
	addUpdate()
	addSocial()
	addDownload()
	addRuntime()
	addResource()
	addDefaults()
	addEnv()
	addScript()
	addTask()
	addDouyin()
	addViral()
	// align must be last — it's registered separately
	addAlign()
}

func addTTS() {
	var filePath, voiceName, outputPath string
	var speechRate float64
	var trimLongSilence bool
	cmd := &cobra.Command{
		Use:   "tts [text]",
		Short: "Text to speech",
		RunE: func(c *cobra.Command, args []string) error {
			text := ""
			if filePath != "" {
				data, err := os.ReadFile(filePath)
				if err != nil {
					return output.ErrSystem("read file %s: %v", filePath, err)
				}
				text = string(data)
			} else if len(args) > 0 {
				text = args[0]
			}
			if text == "" {
				return output.ErrValidation("text or --file is required")
			}
			return runTTS(text, voiceName, speechRate, trimLongSilence, outputPath)
		},
	}
	cmd.Flags().StringVar(&filePath, "file", "", "Read text from file")
	cmd.Flags().StringVar(&voiceName, "voice", "男声3", "Voice name")
	cmd.Flags().Float64Var(&speechRate, "speech-rate", 1.1, "Speech rate")
	cmd.Flags().BoolVar(&trimLongSilence, "trim-long-silence", false, "Trim long silences")
	cmd.Flags().StringVarP(&outputPath, "output", "o", "", "Output wav path")
	rootCmd.AddCommand(cmd)
}

func addLipSync() {
	var avatar, audioPath, audioKey, outputPath, multiShotJSON string
	var randomStart, noSuperRes bool
	var guidanceScale float64
	var numInferenceSteps, superresScale, timeoutSec int
	cmd := &cobra.Command{
		Use:   "lipsync",
		Short: "Digital human lip sync",
		RunE: func(c *cobra.Command, args []string) error {
			if avatar == "" {
				return output.ErrValidation("--avatar is required")
			}
			return runLipSync(avatar, audioPath, audioKey, outputPath, randomStart, guidanceScale, float64(numInferenceSteps), noSuperRes, superresScale, multiShotJSON, timeoutSec)
		},
	}
	cmd.Flags().StringVar(&avatar, "avatar", "", "Digital avatar name")
	cmd.Flags().StringVar(&audioPath, "audio", "", "Audio file path")
	cmd.Flags().StringVar(&audioKey, "audio-key", "", "Cloud audio object key")
	cmd.Flags().StringVarP(&outputPath, "output", "o", "", "Output video path")
	cmd.Flags().BoolVar(&randomStart, "random-start", true, "Start avatar from random position")
	cmd.Flags().Float64Var(&guidanceScale, "guidance-scale", 1.0, "Lip-sync guidance scale")
	cmd.Flags().IntVar(&numInferenceSteps, "num-inference-steps", 15, "Inference steps")
	cmd.Flags().BoolVar(&noSuperRes, "no-superres", false, "Disable super-resolution")
	cmd.Flags().IntVar(&superresScale, "superres-scale", 2, "Super-resolution scale")
	cmd.Flags().StringVar(&multiShotJSON, "multi-shot-json", "", "JSON payload for multi_shot")
	cmd.Flags().IntVar(&timeoutSec, "timeout", 600, "Cloud task timeout in seconds")
	rootCmd.AddCommand(cmd)
}

func addASR() {
	var language, outputPath string
	cmd := &cobra.Command{
		Use:   "asr <video_or_audio>",
		Short: "Speech recognition",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			return runASRFileWithOutput(args[0], language, outputPath, resolveProjectByName(""))
		},
	}
	cmd.Flags().StringVar(&language, "language", "zh", "Language code")
	cmd.Flags().StringVar(&outputPath, "output", "", "Output ASR JSON path")
	rootCmd.AddCommand(cmd)
}

func addEnhance() {
	var outputPath string
	var scale int
	cmd := &cobra.Command{
		Use:   "enhance <video>",
		Short: "Video enhancement",
		Args:  cobra.ExactArgs(1),
		RunE:  func(c *cobra.Command, args []string) error { return runEnhance(args[0], scale, outputPath) },
	}
	cmd.Flags().IntVar(&scale, "scale", 2, "Upscale factor")
	cmd.Flags().StringVarP(&outputPath, "output", "o", "", "Output path")
	rootCmd.AddCommand(cmd)
}

func addImage() { rootCmd.AddCommand(nativeCmd("image", "AI image generation", cmdImage)) }

func addVideo() { rootCmd.AddCommand(nativeCmd("video", "AI video generation", cmdVideo)) }

func addSubtitle() {
	var transcriptPath, outputPath, segmentsOutput, projectName, fontResource, persona string
	var isTextMode, skipEffects, skipHighlight bool
	var maxChars, fontSize, sideMargin, bottomMargin int
	var color, strokeColor, highlightColor string
	cmd := &cobra.Command{
		Use:   "subtitle <video_or_text>",
		Short: "Subtitle generation and rendering",
		RunE: func(c *cobra.Command, args []string) error {
			input := ""
			if len(args) > 0 {
				input = args[0]
			}
			return runSubtitle(input, isTextMode, transcriptPath, outputPath, segmentsOutput, projectName,
				maxChars, fontSize, sideMargin, bottomMargin, fontResource,
				color, strokeColor, highlightColor, skipEffects, skipHighlight, persona)
		},
	}
	cmd.Flags().BoolVar(&isTextMode, "text", false, "Treat input as raw text")
	cmd.Flags().StringVar(&transcriptPath, "transcript", "", "Transcript file path")
	cmd.Flags().StringVarP(&outputPath, "output", "o", "", "Output video path")
	cmd.Flags().StringVar(&segmentsOutput, "segments-output", "", "Segments JSON output path")
	cmd.Flags().StringVar(&projectName, "project", "", "Project name")
	cmd.Flags().IntVar(&maxChars, "max-chars", 15, "Max chars per segment")
	cmd.Flags().IntVar(&fontSize, "font-size", 0, "Font size in px")
	cmd.Flags().IntVar(&sideMargin, "side-margin", 0, "Left/right margin")
	cmd.Flags().IntVar(&bottomMargin, "bottom-margin", 0, "Bottom margin")
	cmd.Flags().StringVar(&fontResource, "font", "", "Font resource ID")
	cmd.Flags().StringVar(&color, "color", "", "Font color")
	cmd.Flags().StringVar(&strokeColor, "stroke", "", "Stroke color")
	cmd.Flags().StringVar(&highlightColor, "highlight-color", "", "Highlight color")
	cmd.Flags().BoolVar(&skipEffects, "no-effects", true, "Skip effects")
	cmd.Flags().BoolVar(&skipHighlight, "no-highlight", false, "Skip highlight")
	cmd.Flags().StringVar(&persona, "persona", "", "Persona hint")
	rootCmd.AddCommand(cmd)
}

func addBGM() {
	var bgmValue, outputPath, voiceVol, bgmVol string
	cmd := &cobra.Command{
		Use:   "bgm mix <video>",
		Short: "Mix background music",
		Args:  cobra.ExactArgs(2),
		RunE: func(c *cobra.Command, args []string) error {
			if args[0] != "mix" {
				return fmt.Errorf("expected: bgm mix <video>")
			}
			return runBGM(args[1], bgmValue, outputPath, voiceVol, bgmVol)
		},
	}
	cmd.Flags().StringVar(&bgmValue, "bgm", "", "BGM file or cloud resource ID")
	cmd.Flags().StringVarP(&outputPath, "output", "o", "step6_bgm.mp4", "Output mp4 path")
	cmd.Flags().StringVar(&voiceVol, "voice-volume", "1.0", "Voice volume")
	cmd.Flags().StringVar(&bgmVol, "bgm-volume", "0.25", "BGM volume")
	rootCmd.AddCommand(cmd)
}

func addCover()    { rootCmd.AddCommand(nativeCmd("cover", "Cover frame and image rendering", cmdCover)) }
func addPIP()      { rootCmd.AddCommand(nativeCmd("pip", "Picture-in-picture", cmdPIP)) }
func addMaterial() { rootCmd.AddCommand(nativeCmd("material", "Material management", cmdMaterial)) }
func addAsset()    { rootCmd.AddCommand(nativeCmd("asset", "Asset upload and listing", cmdAsset)) }
func addAssets()   { rootCmd.AddCommand(nativeCmd("assets", "Asset Registry V2 commands", cmdAssets)) }
func addAvatarPersona() {
	rootCmd.AddCommand(nativeCmd("avatar-persona", "Avatar persona management", cmdAvatarPersona))
}
func addContent()  { rootCmd.AddCommand(nativeCmd("content", "Content planning", cmdContent)) }
func addResearch() { rootCmd.AddCommand(nativeCmd("research", "Content research", cmdResearch)) }
func addURLRead() {
	rootCmd.AddCommand(nativeCmd("url-read", "Read URL content through cloud Jina Reader", cmdURLRead))
}
func addProfile() { rootCmd.AddCommand(nativeCmd("profile", "Global content profiles", cmdProfile)) }
func addProject() { rootCmd.AddCommand(nativeCmd("project", "Project management", cmdProject)) }
func addAuth()    { rootCmd.AddCommand(nativeCmd("auth", "Authentication", cmdAuth)) }
func addVoice()   { rootCmd.AddCommand(nativeCmd("voice", "Voice management", cmdVoice)) }
func addAgent()   { rootCmd.AddCommand(nativeCmd("agent", "Run agent abilities", cmdAgent)) }
func addTools()   { rootCmd.AddCommand(nativeCmd("tools", "Tool discovery", cmdTools)) }
func addSkills()  { rootCmd.AddCommand(nativeCmd("skills", "Skills management", cmdSkills)) }
func addUpdate()  { rootCmd.AddCommand(nativeCmd("update", "Update CLI", cmdUpdate)) }
func addSocial()  { rootCmd.AddCommand(nativeCmd("social", "Social media download", cmdSocial)) }
func addDownload() {
	rootCmd.AddCommand(&cobra.Command{
		Use: "download <url> [output_file]", Short: "Download files",
		RunE: func(c *cobra.Command, args []string) error {
			url := ""
			out := "download.mp4"
			if len(args) > 0 {
				url = args[0]
			}
			if len(args) > 1 {
				out = args[1]
			}
			return runDownload(url, out)
		},
	})
}
func addRuntime()  { rootCmd.AddCommand(nativeCmd("runtime", "Runtime management", cmdRuntime)) }
func addResource() { rootCmd.AddCommand(nativeCmd("resource", "Resource management", cmdResource)) }
func addDefaults() {
	rootCmd.AddCommand(&cobra.Command{
		Use: "defaults show", Short: "Show defaults",
		RunE: func(c *cobra.Command, args []string) error { return cmdDefaults([]string{"show"}) },
	})
}
func addEnv()    { rootCmd.AddCommand(nativeCmd("env", "Backend environment", cmdEnv)) }
func addScript() { rootCmd.AddCommand(nativeCmd("script", "Script helpers", cmdScript)) }
func addTask()   { rootCmd.AddCommand(nativeCmd("task", "Task status", cmdTask)) }
func addDouyin() { rootCmd.AddCommand(nativeCmd("douyin", "Douyin helpers", cmdDouyin)) }
func addViral()  { rootCmd.AddCommand(nativeCmd("viral", "Viral copy helpers", cmdViral)) }
func addAlign()  { rootCmd.AddCommand(nativeCmd("align", "Align subtitles", cmdAlign)) }

func nativeCmd(use, short string, fn commandHandler) *cobra.Command {
	return &cobra.Command{Use: use, Short: short, DisableFlagParsing: true,
		RunE: func(c *cobra.Command, args []string) error { return fn(args) }}
}
