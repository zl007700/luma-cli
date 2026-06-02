package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/luma-cli/lumer-cli/cloud"
)

// subtitleOptions holds parsed CLI options for the subtitle command.
type subtitleOptions struct {
	input           string // video file path or raw text
	isTextMode      bool
	transcriptPath  string
	outputPath      string
	segmentsOutput  string
	projectName     string
	maxChars        int
	fontSize        int
	fontResource    string
	color           string
	strokeColor     string
	highlightColor  string
	sideMargin      int
	bottomMargin    int
	fontSizeSet     bool
	sideMarginSet   bool
	bottomMarginSet bool
	skipEffects     bool
	skipHighlight   bool
	persona         string
}

func parseSubtitleArgs(args []string) *subtitleOptions {
	opts := &subtitleOptions{
		maxChars:       15,
		color:          "#FDFDFF",
		strokeColor:    "#1F0101",
		highlightColor: "#FFD95A",
		skipEffects:    true,
	}

	// First pass: find positional arg and --text flag
	positionalArgs := make([]string, 0)
	for i := 0; i < len(args); i++ {
		if args[i] == "--text" {
			opts.isTextMode = true
			continue
		}
		if strings.HasPrefix(args[i], "--") {
			if !isBoolFlag(args[i]) && i+1 < len(args) && !strings.HasPrefix(args[i+1], "--") {
				i++
			}
			continue
		}
		positionalArgs = append(positionalArgs, args[i])
	}

	if len(positionalArgs) == 0 {
		printSubtitleUsage()
		return nil
	}

	opts.input = positionalArgs[0]

	if !opts.isTextMode {
		if _, err := os.Stat(opts.input); err != nil {
			fmt.Printf("Error: video file not found: %s\n", opts.input)
			return nil
		}
	}

	// Second pass: parse options
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--output":
			if i+1 < len(args) {
				opts.outputPath = args[i+1]
				i++
			}
		case "--transcript":
			if i+1 < len(args) {
				opts.transcriptPath = args[i+1]
				i++
			}
		case "--segments-output":
			if i+1 < len(args) {
				opts.segmentsOutput = args[i+1]
				i++
			}
		case "--project":
			if i+1 < len(args) {
				opts.projectName = args[i+1]
				i++
			}
		case "--max-chars":
			if i+1 < len(args) {
				fmt.Sscanf(args[i+1], "%d", &opts.maxChars)
				i++
			}
		case "--font-size":
			if i+1 < len(args) {
				fmt.Sscanf(args[i+1], "%d", &opts.fontSize)
				opts.fontSizeSet = true
				i++
			}
		case "--side-margin":
			if i+1 < len(args) {
				fmt.Sscanf(args[i+1], "%d", &opts.sideMargin)
				opts.sideMarginSet = true
				i++
			}
		case "--bottom-margin":
			if i+1 < len(args) {
				fmt.Sscanf(args[i+1], "%d", &opts.bottomMargin)
				opts.bottomMarginSet = true
				i++
			}
		case "--font":
			if i+1 < len(args) {
				opts.fontResource = args[i+1]
				i++
			}
		case "--color":
			if i+1 < len(args) {
				opts.color = args[i+1]
				i++
			}
		case "--stroke":
			if i+1 < len(args) {
				opts.strokeColor = args[i+1]
				i++
			}
		case "--highlight-color":
			if i+1 < len(args) {
				opts.highlightColor = args[i+1]
				i++
			}
		case "--effects":
			opts.skipEffects = false
		case "--no-effects", "--skip-effects":
			opts.skipEffects = true
		case "--no-highlight":
			opts.skipHighlight = true
		case "--persona":
			if i+1 < len(args) {
				opts.persona = args[i+1]
				i++
			}
		}
	}

	return opts
}

func isBoolFlag(flag string) bool {
	return flag == "--text" || flag == "--effects" || flag == "--no-effects" || flag == "--skip-effects" || flag == "--no-highlight"
}

func applySubtitleDefaults(opts *subtitleOptions, defaults *cloud.ClientDefaults) {
	if opts == nil || defaults == nil {
		return
	}
	if opts.fontResource == "" {
		opts.fontResource = defaults.Subtitle.Font
	}
	if opts.fontSize <= 0 && defaults.Subtitle.FontSize > 0 {
		opts.fontSize = defaults.Subtitle.FontSize
	}
	if opts.sideMargin <= 0 && defaults.Subtitle.SideMargin > 0 {
		opts.sideMargin = defaults.Subtitle.SideMargin
	}
	if opts.bottomMargin <= 0 && defaults.Subtitle.BottomMargin > 0 {
		opts.bottomMargin = defaults.Subtitle.BottomMargin
	}
	if defaults.Subtitle.MaxChars > 0 && opts.maxChars == 15 {
		opts.maxChars = defaults.Subtitle.MaxChars
	}
	if opts.color == "#FDFDFF" && defaults.Subtitle.Color != "" {
		opts.color = defaults.Subtitle.Color
	}
	if opts.strokeColor == "#1F0101" && defaults.Subtitle.StrokeColor != "" {
		opts.strokeColor = defaults.Subtitle.StrokeColor
	}
	if opts.highlightColor == "#FFD95A" && defaults.Subtitle.HighlightColor != "" {
		opts.highlightColor = defaults.Subtitle.HighlightColor
	}
	if !defaults.Subtitle.EffectsEnabled {
		opts.skipEffects = true
	}
	if !defaults.Subtitle.HighlightEnabled {
		opts.skipHighlight = true
	}
}
