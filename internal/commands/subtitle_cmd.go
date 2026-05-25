package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/luma-cli/lumer-cli/subtitle"
)

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
	segments, err = runAlignment(segments, opts.isTextMode, opts.input, cfg.CardKey, projDirs)
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
	fontSize, marginV := resolveFontSize(opts.fontSize, opts.bottomMargin, height)
	sideMargin := opts.sideMargin
	if sideMargin <= 0 {
		sideMargin = 60
	}
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
		HighlightScale: 1.25, MarginL: sideMargin, MarginR: sideMargin, MarginV: marginV,
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

	// Rename output to include content hash for traceability.
	if hashed, err := hashSuffixFile(outputPath); err == nil {
		outputPath = hashed
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
	fmt.Println("    --side-margin <n>        - Left/right subtitle margin in px (default: product setting)")
	fmt.Println("    --bottom-margin <n>      - Bottom subtitle margin in px (default: product setting)")
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
