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
			Run: func(c *cobra.Command, a []string) { fmt.Printf("luma-cli version %s\n", version) }},
	)
	// Core media commands
	addTTS()
	addLipSync()
	addASR()
	addEnhance()
	addSubtitle()

	// Workflow commands
	addBGM()
	addCover()
	addPIP()

	// Resource commands
	addMaterial()
	addResearch()
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
	var language string
	cmd := &cobra.Command{
		Use:   "asr <video_or_audio>",
		Short: "Speech recognition",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error { return runASRFile(args[0], language) },
	}
	cmd.Flags().StringVar(&language, "language", "zh", "Language code")
	rootCmd.AddCommand(cmd)
}

func addEnhance() {
	var outputPath string
	var scale int
	cmd := &cobra.Command{
		Use:   "enhance <video>",
		Short: "Video enhancement",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error { return runEnhance(args[0], scale, outputPath) },
	}
	cmd.Flags().IntVar(&scale, "scale", 2, "Upscale factor")
	cmd.Flags().StringVarP(&outputPath, "output", "o", "", "Output path")
	rootCmd.AddCommand(cmd)
}

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
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			return runBGM(args[0], bgmValue, outputPath, voiceVol, bgmVol)
		},
	}
	cmd.Flags().StringVar(&bgmValue, "bgm", "", "BGM file or cloud resource ID")
	cmd.Flags().StringVarP(&outputPath, "output", "o", "step6_bgm.mp4", "Output mp4 path")
	cmd.Flags().StringVar(&voiceVol, "voice-volume", "1.0", "Voice volume")
	cmd.Flags().StringVar(&bgmVol, "bgm-volume", "0.25", "BGM volume")
	rootCmd.AddCommand(cmd)
}

func addCover()    { rootCmd.AddCommand(wrapCmd("cover", "Cover frame and image rendering", cmdCover)) }
func addPIP()      { rootCmd.AddCommand(wrapCmd("pip", "Picture-in-picture", cmdPIP)) }
func addMaterial() { rootCmd.AddCommand(wrapCmd("material", "Material management", cmdMaterial)) }
func addResearch() { rootCmd.AddCommand(wrapCmd("research", "Content research", cmdResearch)) }
func addProject()  { rootCmd.AddCommand(wrapCmd("project", "Project management", cmdProject)) }
func addAuth()     { rootCmd.AddCommand(wrapCmd("auth", "Authentication", cmdAuth)) }
func addVoice()    { rootCmd.AddCommand(wrapCmd("voice", "Voice management", cmdVoice)) }
func addAgent()    { rootCmd.AddCommand(wrapCmd("agent", "Run agent abilities", cmdAgent)) }
func addTools()    { rootCmd.AddCommand(wrapCmd("tools", "Tool discovery", cmdTools)) }
func addSkills()   { rootCmd.AddCommand(wrapCmd("skills", "Skills management", cmdSkills)) }
func addUpdate()   { rootCmd.AddCommand(wrapCmd("update", "Update CLI", cmdUpdate)) }
func addSocial()   { rootCmd.AddCommand(wrapCmd("social", "Social media download", cmdSocial)) }
func addDownload() { rootCmd.AddCommand(wrapCmd("download", "Download files", cmdDownload)) }
func addRuntime()  { rootCmd.AddCommand(wrapCmd("runtime", "Runtime management", cmdRuntime)) }
func addResource() { rootCmd.AddCommand(wrapCmd("resource", "Resource management", cmdResource)) }
func addDefaults() { rootCmd.AddCommand(wrapCmd("defaults", "Show defaults", cmdDefaults)) }
func addScript()   { rootCmd.AddCommand(wrapCmd("script", "Script helpers", cmdScript)) }
func addTask()     { rootCmd.AddCommand(wrapCmd("task", "Task status", cmdTask)) }
func addDouyin()   { rootCmd.AddCommand(wrapCmd("douyin", "Douyin helpers", cmdDouyin)) }
func addViral()    { rootCmd.AddCommand(wrapCmd("viral", "Viral copy helpers", cmdViral)) }
func addAlign()    { rootCmd.AddCommand(wrapCmd("align", "Align subtitles", cmdAlign)) }

func wrapCmd(use, short string, fn commandHandler) *cobra.Command {
	return &cobra.Command{Use: use, Short: short, DisableFlagParsing: true,
		RunE: func(c *cobra.Command, args []string) error { return fn(args) }}
}
