package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/luma-cli/lumer-cli/cloud"
	"github.com/luma-cli/lumer-cli/internal/atom"
	"github.com/luma-cli/lumer-cli/project"
	"github.com/luma-cli/lumer-cli/subtitle"
)

// subtitleOptions holds parsed CLI options for the subtitle command.
type subtitleOptions struct {
	input          string // video file path or raw text
	isTextMode     bool
	transcriptPath string
	outputPath     string
	segmentsOutput string
	projectName    string
	maxChars       int
	fontSize       int
	fontResource   string
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
	defaults := loadClientDefaults(cfg)

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
	applySubtitleDefaults(opts, defaults)

	// Run pipeline
	rawText := opts.input
	var err error

	// Step 1: ASR (video mode only)
	if !opts.isTextMode {
		if opts.transcriptPath != "" {
			data, err := os.ReadFile(opts.transcriptPath)
			if err != nil {
				fmt.Printf("Error: read transcript failed: %v\n", err)
				return
			}
			rawText = strings.TrimSpace(string(data))
			fmt.Printf("Step 1/6: Using transcript: %s\n", opts.transcriptPath)
			if rawText == "" {
				fmt.Println("Error: transcript is empty")
				return
			}
		} else {
			rawText, err = runASR(opts.input, cfg.CardKey, projDirs)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				return
			}
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
	segments, err = runAlignment(segments, sentenceGroups, opts.isTextMode, opts.input, cfg.CardKey, projDirs)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

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
	if opts.segmentsOutput != "" {
		segmentsPath, err := absoluteOutputPath(opts.segmentsOutput)
		if err != nil {
			fmt.Printf("Error: bad segments output path: %v\n", err)
			return
		}
		if err := writeJSONFile(segmentsPath, map[string]any{"segments": segments, "sentence_groups": sentenceGroups}); err != nil {
			fmt.Printf("Error writing segments: %v\n", err)
			return
		}
		recordProjectArtifact("segments", segmentsPath, "subtitle.segments")
		fmt.Printf("  Segments written: %s\n", segmentsPath)
	}

	// Step 6: Generate ASS and burn
	fmt.Println("Step 6/6: Generating ASS and burning subtitles...")
	width, height := resolveVideoSize(opts.isTextMode, opts.input)
	fontSize, marginV := resolveFontSize(opts.fontSize, height)
	fontPath := ""
	if opts.fontResource != "" {
		fontPath, err = cacheDefaultResource(opts.fontResource, cfg)
		if err != nil {
			fmt.Printf("  Warning: resolve subtitle font failed: %v\n", err)
		}
	}

	opts2 := subtitle.ASSOptions{
		PlayResX: width, PlayResY: height,
		FontName: fontNameFromPath(fontPath), FontSize: fontSize,
		Color: opts.color, StrokeColor: opts.strokeColor,
		BackColor: "#000000", HighlightColor: opts.highlightColor,
		HighlightScale: 1.25, MarginL: 60, MarginR: 60, MarginV: marginV,
		Outline: float64(fontSize) * 0.07, Shadow: 0, Spacing: 2.0,
	}

	assPath := resolveAssPath(outputPath, proj, projDirs)
	fontDir := filepath.Dir(assPath)
	if fontPath != "" {
		if copiedFont, err := copyFontToDir(fontPath, fontDir); err == nil {
			fontPath = copiedFont
		} else {
			fmt.Printf("  Warning: copy subtitle font failed: %v\n", err)
		}
	}
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
	if err := subtitle.BurnSubtitles(opts.input, assPath, outputPath, fontDir, ""); err != nil {
		fmt.Printf("Error burning subtitles: %v\n", err)
		os.Remove(assPath)
		return
	}

	// Apply effect overlays
	effectSegments := filterEffectSegments(segments)
	if len(effectSegments) > 0 {
		fmt.Printf("  Rendering %d effect overlays...\n", len(effectSegments))
		finalPath, err := subtitle.ApplyEffectOverlays(outputPath, effectSegments, width, height, fontSize, opts.color, opts.strokeColor, opts.highlightColor, fontPath, projDirs.effects)
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

func isBoolFlag(flag string) bool {
	return flag == "--text" || flag == "--effects" || flag == "--no-effects" || flag == "--skip-effects" || flag == "--no-highlight"
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

func runAlignment(segments []subtitle.Segment, sentenceGroups []subtitle.SentenceGroup, isTextMode bool, videoPath, cardKey string, dirs projectDirs) ([]subtitle.Segment, error) {
	fmt.Println("Step 3/6: Aligning segments to audio timestamps...")
	if isTextMode {
		totalDuration := float64(len(segments) * 3)
		segments = subtitle.FallbackEvenAlign(segments, totalDuration)
		fmt.Println("  Text-only mode: using estimated even alignment")
		return segments, nil
	}

	audioPath := filepath.Join(dirs.audio, "subtitle_align.wav")
	if dirs.audio == "" {
		tmp, err := os.CreateTemp("", "luma-subtitle-align-*.wav")
		if err != nil {
			return nil, fmt.Errorf("create alignment audio temp file failed: %w", err)
		}
		audioPath = tmp.Name()
		tmp.Close()
		defer os.Remove(audioPath)
	} else if err := os.MkdirAll(dirs.audio, 0755); err != nil {
		return nil, fmt.Errorf("create project audio dir failed: %w", err)
	}

	if err := subtitle.ExtractAudio(videoPath, audioPath, ""); err != nil {
		return nil, fmt.Errorf("extract audio for alignment failed: %w", err)
	}

	alignTargets := sentenceGroups
	if len(alignTargets) == 0 {
		alignTargets = buildFallbackSentenceGroups(segments)
	}
	texts := make([]string, 0, len(alignTargets))
	for _, group := range alignTargets {
		texts = append(texts, group.Text)
	}
	aligned, err := atom.RunAlignment(atom.AlignmentOptions{
		AudioPath:  audioPath,
		TextList:   texts,
		Language:   "zh",
		CardKey:    cardKey,
		TimeoutSec: 300,
	})
	if err != nil {
		return nil, err
	}
	if len(aligned) < len(alignTargets) {
		return nil, fmt.Errorf("cloud alignment returned too few segments: got %d, want %d", len(aligned), len(alignTargets))
	}
	segments = applySentenceGroupAlignment(segments, alignTargets, aligned)
	if hasUntimedSegments(segments) {
		return nil, fmt.Errorf("cloud alignment left some subtitle segments without timing")
	}
	fmt.Printf("  Cloud alignment applied: %d sentence groups -> %d subtitle segments\n", len(aligned), len(segments))
	return segments, nil
}

func buildFallbackSentenceGroups(segments []subtitle.Segment) []subtitle.SentenceGroup {
	groups := make([]subtitle.SentenceGroup, 0, len(segments))
	for _, seg := range segments {
		groups = append(groups, subtitle.SentenceGroup{StartSegID: seg.SegID, EndSegID: seg.SegID, Text: seg.Text})
	}
	return groups
}

func applySentenceGroupAlignment(segments []subtitle.Segment, groups []subtitle.SentenceGroup, aligned []atom.AlignmentSegment) []subtitle.Segment {
	segmentIndex := make(map[int]int, len(segments))
	for i, seg := range segments {
		segmentIndex[seg.SegID] = i
	}
	for i, group := range groups {
		if i >= len(aligned) {
			break
		}
		distributeGroupTiming(segments, segmentIndex, group, aligned[i].Start, aligned[i].End)
	}
	return segments
}

func distributeGroupTiming(segments []subtitle.Segment, segmentIndex map[int]int, group subtitle.SentenceGroup, start, end float64) {
	if end <= start {
		return
	}
	childIndexes := make([]int, 0, group.EndSegID-group.StartSegID+1)
	totalChars := 0
	for segID := group.StartSegID; segID <= group.EndSegID; segID++ {
		idx, ok := segmentIndex[segID]
		if !ok {
			continue
		}
		childIndexes = append(childIndexes, idx)
		totalChars += len([]rune(segments[idx].Text))
	}
	if len(childIndexes) == 0 {
		return
	}
	cursor := start
	duration := end - start
	for i, idx := range childIndexes {
		if i == len(childIndexes)-1 {
			segments[idx].Start = cursor
			segments[idx].End = end
			break
		}
		chars := len([]rune(segments[idx].Text))
		next := cursor + duration*(float64(chars)/float64(max(totalChars, 1)))
		segments[idx].Start = cursor
		segments[idx].End = next
		cursor = next
	}
}

func hasUntimedSegments(segments []subtitle.Segment) bool {
	for _, seg := range segments {
		if seg.End <= seg.Start {
			return true
		}
	}
	return false
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
			return filepath.Join(dirs.output, "step5_subtitle.mp4")
		}
		return filepath.Join(dirs.output, "step5_subtitle.mp4")
	}
	if opts.isTextMode {
		return "step5_subtitle.mp4"
	}
	return "step5_subtitle.mp4"
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
	fmt.Println("    --transcript <path>      - Use previous-step transcript text instead of ASR")
	fmt.Println("    --output <path>          - Output video path (default: <input>_subtitled.mp4)")
	fmt.Println("    --segments-output <path> - Output segment JSON path")
	fmt.Println("    --project <name>         - Use specified project for output organization")
	fmt.Println("    --max-chars <n>          - Max chars per segment (default: 15)")
	fmt.Println("    --font-size <n>          - Font size in px (default: auto)")
	fmt.Println("    --font <path_or_resource_id> - Subtitle font (default: product setting)")
	fmt.Println("    --color <hex>            - Font color (default: #FDFDFF)")
	fmt.Println("    --stroke <hex>           - Stroke color (default: #1F0101)")
	fmt.Println("    --highlight-color <hex>  - Highlight color (default: #FFD95A)")
	fmt.Println("    --effects                - Enable experimental effect subtitles")
	fmt.Println("    --no-effects             - Skip experimental subtitle effects (default)")
	fmt.Println("    --no-highlight           - Skip LLM keyword highlight")
	fmt.Println("    --persona <text>         - Persona hint for LLM splitting")
	fmt.Println("")
	fmt.Println("  Example:")
	fmt.Println("    luma-cli subtitle input.mp4")
	fmt.Println("    luma-cli subtitle input.mp4 --font-size 48")
	fmt.Println("    luma-cli subtitle --text \"这是文案\"")
	fmt.Println("    luma-cli subtitle input.mp4 --project my-video")
}
