package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/luma-cli/lumer-cli/internal/atom"
	"github.com/luma-cli/lumer-cli/project"
	"github.com/luma-cli/lumer-cli/subtitle"
)

// subtitleOptions holds parsed CLI options for the subtitle command.
type subtitleOptions struct {
	input          string // video file path or raw text
	isTextMode     bool
	outputPath     string
	projectName    string
	maxChars       int
	fontSize       int
	color          string
	strokeColor    string
	highlightColor string
	skipEffects    bool
	skipHighlight  bool
	persona        string
}

func cmdSubtitle(args []string) {
	if len(args) < 1 {
		printSubtitleUsage()
		return
	}

	opts := parseSubtitleArgs(args)
	if opts == nil {
		return
	}

	// Load config
	cfg := loadConfig()
	if cfg == nil {
		fmt.Println("Error: not logged in. Run: luma-cli auth login <card_key>")
		return
	}

	// Resolve project
	proj := resolveProjectByName(opts.projectName)
	if proj != nil {
		fmt.Printf("Project: %s\n", proj.Name)
		if !opts.isTextMode && proj.Source == "" {
			proj.Source = opts.input
			proj.Save()
		}
	}

	// Determine output paths
	projDirs := getProjectDirs(proj)
	outputPath := resolveOutputPath(opts, proj, projDirs)

	// Run pipeline
	rawText := opts.input
	var err error

	// Step 1: ASR (video mode only)
	if !opts.isTextMode {
		rawText, err = runASR(opts.input, cfg.CardKey, projDirs)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
	}

	// Step 2: LLM split
	fmt.Printf("Step 2/6: LLM splitting text into segments (max %d chars/seg)...\n", opts.maxChars)
	llmClient := subtitle.NewLLMClient(cfg.CardKey, "", "")
	segments, sentenceGroups, err := subtitle.SplitByLLM(rawText, opts.maxChars, llmClient, opts.persona)
	if err != nil {
		fmt.Printf("  Warning: LLM split failed, using fallback: %v\n", err)
	}
	fmt.Printf("  Generated %d segments, %d sentence groups\n", len(segments), len(sentenceGroups))

	// Step 3: Align
	segments = runAlignment(segments, opts.isTextMode, opts.input)

	// Step 4: Highlight
	if !opts.skipHighlight {
		fmt.Println("Step 4/6: LLM assigning highlight keywords...")
		segments, err = llmClient.HighlightByLLM(segments)
		if err != nil {
			fmt.Printf("  Warning: highlight failed: %v\n", err)
		}
		printSegmentCount(segments, "highlighted", func(s subtitle.Segment) bool { return s.HighlightWord != "" })
	}

	// Step 5: Effects
	if !opts.skipEffects {
		fmt.Println("Step 5/6: LLM assigning subtitle effects...")
		maxEffects := max(2, (len(segments)+9)/10)
		segments, err = llmClient.AssignEffectsByLLM(segments, maxEffects)
		if err != nil {
			fmt.Printf("  Warning: effects failed: %v\n", err)
		}
		printSegmentCount(segments, "with effects", func(s subtitle.Segment) bool {
			return s.EffectType != "" && s.EffectType != "none"
		})
	}

	// Step 6: Generate ASS and burn
	fmt.Println("Step 6/6: Generating ASS and burning subtitles...")
	width, height := resolveVideoSize(opts.isTextMode, opts.input)
	fontSize, marginV := resolveFontSize(opts.fontSize, height)

	opts2 := subtitle.ASSOptions{
		PlayResX: width, PlayResY: height,
		FontName: "Microsoft YaHei", FontSize: fontSize,
		Color: opts.color, StrokeColor: opts.strokeColor,
		BackColor: "#000000", HighlightColor: opts.highlightColor,
		HighlightScale: 1.25, MarginL: 60, MarginR: 60, MarginV: marginV,
		Outline: float64(fontSize) * 0.07, Shadow: 0, Spacing: 2.0,
	}

	assPath := resolveAssPath(outputPath, proj, projDirs)
	if err := subtitle.WriteASS(segments, assPath, opts2); err != nil {
		fmt.Printf("Error writing ASS: %v\n", err)
		return
	}
	fmt.Printf("  ASS written: %s\n", assPath)

	if opts.isTextMode {
		fmt.Println("  Text-only mode: ASS generated, no video to burn into")
		fmt.Printf("Output ASS: %s\n", assPath)
		recordStep(proj, "subtitle (text-only)", "", assPath)
		return
	}

	// Burn subtitles
	fontDir := filepath.Dir(assPath)
	if err := subtitle.BurnSubtitles(opts.input, assPath, outputPath, fontDir, ""); err != nil {
		fmt.Printf("Error burning subtitles: %v\n", err)
		os.Remove(assPath)
		return
	}

	// Apply effect overlays
	effectSegments := filterEffectSegments(segments)
	if len(effectSegments) > 0 {
		fmt.Printf("  Rendering %d effect overlays...\n", len(effectSegments))
		finalPath, err := subtitle.ApplyEffectOverlays(outputPath, effectSegments, width, height, fontSize, opts.color, opts.strokeColor, opts.highlightColor, projDirs.effects)
		if err != nil {
			fmt.Printf("  Warning: effect overlay failed: %v (continuing without effects)\n", err)
		} else if finalPath != outputPath {
			outputPath = finalPath
		}
	}

	fmt.Printf("\nDone! Output: %s\n", outputPath)
	recordStep(proj, "subtitle", opts.input, outputPath)

	// Summary
	fmt.Printf("\n--- Subtitle Summary ---\n")
	fmt.Printf("Segments: %d\n", len(segments))
	printSegmentCount(segments, "highlighted", func(s subtitle.Segment) bool { return s.HighlightWord != "" })
	printSegmentCount(segments, "with effects", func(s subtitle.Segment) bool {
		return s.EffectType != "" && s.EffectType != "none"
	})
	fmt.Printf("Font size: %dpx\n", fontSize)
	fmt.Printf("Output: %s\n", outputPath)
	if proj != nil {
		fmt.Printf("Project: %s\n", proj.Name)
	}
}

// --- Arg parsing ---

func parseSubtitleArgs(args []string) *subtitleOptions {
	opts := &subtitleOptions{
		maxChars:       15,
		color:          "#FDFDFF",
		strokeColor:    "#1F0101",
		highlightColor: "#FFD95A",
	}

	// First pass: find positional arg and --text flag
	positionalArgs := make([]string, 0)
	for i := 0; i < len(args); i++ {
		if args[i] == "--text" {
			opts.isTextMode = true
			continue
		}
		if strings.HasPrefix(args[i], "--") {
			// Skip flag value too
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
		case "--no-effects":
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
	return flag == "--text" || flag == "--no-effects" || flag == "--no-highlight"
}

// --- Pipeline steps ---

func runASR(videoPath, cardKey string, dirs projectDirs) (string, error) {
	fmt.Println("Step 1/6: Extracting audio and running ASR...")

	fmt.Println("  Uploading video for ASR...")
	fmt.Println("  Submitting ASR task...")
	result, err := atom.RunASR(atom.ASROptions{
		FilePath:   videoPath,
		Language:   "zh",
		CardKey:    cardKey,
		TimeoutSec: 180,
	})
	if err != nil {
		return "", err
	}
	fmt.Printf("  Uploaded: %s\n", result.ObjectKey)
	fmt.Printf("  Task ID: %s\n", result.TaskID)
	fmt.Printf("  ASR text length: %d chars\n", len(result.Text))
	return result.Text, nil
}

func runAlignment(segments []subtitle.Segment, isTextMode bool, videoPath string) []subtitle.Segment {
	fmt.Println("Step 3/6: Aligning segments to audio timestamps...")
	if isTextMode {
		totalDuration := float64(len(segments) * 3)
		segments = subtitle.FallbackEvenAlign(segments, totalDuration)
		fmt.Println("  Text-only mode: using estimated even alignment")
		return segments
	}

	dur, err := subtitle.GetVideoDuration(videoPath, "")
	if err != nil || dur <= 0 {
		dur = float64(len(segments) * 3)
	}
	segments = subtitle.FallbackEvenAlign(segments, dur)
	fmt.Printf("  Using even alignment (duration: %.1fs)\n", dur)
	return segments
}

// --- Helpers ---

type projectDirs struct {
	audio     string
	subtitles string
	effects   string
	output    string
}

func getProjectDirs(proj *project.Project) projectDirs {
	if proj == nil {
		return projectDirs{}
	}
	return projectDirs{
		audio:     proj.SubDir(project.DirAudio),
		subtitles: proj.SubDir(project.DirSubtitles),
		effects:   proj.SubDir(project.DirEffects),
		output:    proj.SubDir(project.DirOutput),
	}
}

func resolveProjectByName(name string) *project.Project {
	if name != "" {
		p, err := project.FindByName(name)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return nil
		}
		return p
	}
	p, _ := project.GetActiveProject()
	return p
}

func resolveOutputPath(opts *subtitleOptions, proj *project.Project, dirs projectDirs) string {
	if opts.outputPath != "" {
		return opts.outputPath
	}
	if proj != nil {
		if opts.isTextMode {
			return filepath.Join(dirs.output, "subtitle_output.mp4")
		}
		ext := filepath.Ext(opts.input)
		return filepath.Join(dirs.output, strings.TrimSuffix(filepath.Base(opts.input), ext)+"_subtitled"+ext)
	}
	if opts.isTextMode {
		return "subtitle_output.mp4"
	}
	ext := filepath.Ext(opts.input)
	return strings.TrimSuffix(opts.input, ext) + "_subtitled" + ext
}

func resolveVideoSize(isTextMode bool, videoPath string) (int, int) {
	width, height := 1080, 1920
	if !isTextMode {
		w, h, err := subtitle.GetVideoSize(videoPath)
		if err == nil && w > 0 && h > 0 {
			width, height = w, h
		}
	}
	return width, height
}

func resolveFontSize(requestedFontSize, height int) (int, int) {
	autoFontSize, _, _, marginV := subtitle.AutoSizeParams(0, height)
	if requestedFontSize > 0 {
		return requestedFontSize, marginV
	}
	return autoFontSize, marginV
}

func resolveAssPath(outputPath string, proj *project.Project, dirs projectDirs) string {
	if proj == nil {
		return outputPath + ".ass"
	}
	assName := filepath.Base(outputPath)
	ext := filepath.Ext(assName)
	return filepath.Join(dirs.subtitles, strings.TrimSuffix(assName, ext)+".ass")
}

func filterEffectSegments(segments []subtitle.Segment) []subtitle.Segment {
	var result []subtitle.Segment
	for _, seg := range segments {
		if seg.EffectType != "" && seg.EffectType != "none" {
			result = append(result, seg)
		}
	}
	return result
}

func printSegmentCount(segments []subtitle.Segment, label string, match func(subtitle.Segment) bool) {
	count := 0
	for _, seg := range segments {
		if match(seg) {
			count++
		}
	}
	pct := float64(count) / float64(len(segments)) * 100
	fmt.Printf("  %d segments %s (%.0f%%)\n", count, label, pct)
}

func printSubtitleUsage() {
	fmt.Println("usage: luma-cli subtitle <video_or_text> [options]")
	fmt.Println("")
	fmt.Println("  Full pipeline: ASR -> LLM split -> align -> highlight -> effects -> burn")
	fmt.Println("")
	fmt.Println("  Arguments:")
	fmt.Println("    <video_or_text>  - Video file OR plain text (use --text flag)")
	fmt.Println("")
	fmt.Println("  Options:")
	fmt.Println("    --text                   - Treat argument as raw text instead of video file")
	fmt.Println("    --output <path>          - Output video path (default: <input>_subtitled.mp4)")
	fmt.Println("    --project <name>         - Use specified project for output organization")
	fmt.Println("    --max-chars <n>          - Max chars per segment (default: 15)")
	fmt.Println("    --font-size <n>          - Font size in px (default: auto)")
	fmt.Println("    --color <hex>            - Font color (default: #FDFDFF)")
	fmt.Println("    --stroke <hex>           - Stroke color (default: #1F0101)")
	fmt.Println("    --highlight-color <hex>  - Highlight color (default: #FFD95A)")
	fmt.Println("    --no-effects             - Skip LLM subtitle effects")
	fmt.Println("    --no-highlight           - Skip LLM keyword highlight")
	fmt.Println("    --persona <text>         - Persona hint for LLM splitting")
	fmt.Println("")
	fmt.Println("  Example:")
	fmt.Println("    luma-cli subtitle input.mp4")
	fmt.Println("    luma-cli subtitle input.mp4 --font-size 48")
	fmt.Println("    luma-cli subtitle --text \"这是文案\"")
	fmt.Println("    luma-cli subtitle input.mp4 --project my-video")
}
