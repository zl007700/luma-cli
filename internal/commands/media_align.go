package commands

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/luma-cli/lumer-cli/internal/atom"
	"github.com/luma-cli/lumer-cli/internal/cmdutil"
	"github.com/luma-cli/lumer-cli/subtitle"
)

func cmdAlign(args []string) {
	parsed := cmdutil.Parse(args)
	audioPath := parsed.String("audio", "")
	segmentsPath := parsed.String("segments", "")
	outputPath := parsed.String("output", "")
	language := parsed.String("language", "zh")

	if audioPath == "" || segmentsPath == "" {
		fmt.Println("usage: luma-cli align --audio <tts.wav> --segments <segments.json> [--output <path>] [--language zh|en]")
		fmt.Println("")
		fmt.Println("  Aligns subtitle segments to audio timestamps via cloud alignment API.")
		fmt.Println("  Expects a segments JSON file (from subtitle.split or subtitle --text --segments-output).")
		fmt.Println("  Uses sentence_groups for alignment; falls back to segments.")
		fmt.Println("  Distributes sentence-group timing back to individual segments.")
		fmt.Println("")
		fmt.Println("  Options:")
		fmt.Println("    --audio <file>       Local audio file (e.g. TTS output WAV)")
		fmt.Println("    --segments <file>    Segments JSON file with sentence_groups or segments array")
		fmt.Println("    --output <path>      Output aligned JSON path (default: align_result.json)")
		fmt.Println("    --language <code>    Recognition language (default: zh)")
		return
	}

	if _, err := os.Stat(audioPath); err != nil {
		fmt.Printf("Error: audio file not found: %s\n", audioPath)
		return
	}
	data, err := os.ReadFile(segmentsPath)
	if err != nil {
		fmt.Printf("Error: read segments file failed: %v\n", err)
		return
	}

	// Parse full segment structure preserving SegID, StartSegID, EndSegID, etc.
	var fullPayload struct {
		Segments       []subtitle.Segment       `json:"segments"`
		SentenceGroups []subtitle.SentenceGroup `json:"sentence_groups"`
	}
	if err := json.Unmarshal(data, &fullPayload); err != nil {
		// try result.Segments wrapper
		var wrapped struct {
			Result struct {
				Segments       []subtitle.Segment       `json:"segments"`
				SentenceGroups []subtitle.SentenceGroup `json:"sentence_groups"`
			} `json:"result"`
		}
		if err2 := json.Unmarshal(data, &wrapped); err2 != nil {
			fmt.Printf("Error: parse segments JSON failed: %v\n", err)
			return
		}
		fullPayload = wrapped.Result
	}

	cfg := loadConfig()
	if cfg == nil {
		fmt.Println("Error: not logged in. Run: luma-cli auth login <card_key>")
		return
	}

	// Build text list from segments (direct 1:1 alignment)
	var texts []string
	if len(fullPayload.Segments) > 0 {
		for _, seg := range fullPayload.Segments {
			texts = append(texts, seg.Text)
		}
	} else if len(fullPayload.SentenceGroups) > 0 {
		for _, g := range fullPayload.SentenceGroups {
			texts = append(texts, g.Text)
		}
	}
	if len(texts) == 0 {
		fmt.Println("Error: no segments or sentence_groups found in input")
		return
	}

	fmt.Printf("Aligning %d text items to audio...\n", len(texts))
	result, err := atom.RunAlignment(atom.AlignmentOptions{
		AudioPath:  audioPath,
		TextList:   texts,
		Language:   language,
		CardKey:    cfg.CardKey,
		TimeoutSec: 300,
	})
	if err != nil {
		fmt.Printf("Error: alignment failed: %v\n", err)
		return
	}
	fmt.Printf("  Aligned: %d segments\n", len(result))

	// Map alignment results directly to segments by index (1:1)
	if len(fullPayload.Segments) > 0 {
		for i := range fullPayload.Segments {
			if i < len(result) {
				fullPayload.Segments[i].Start = result[i].Start
				fullPayload.Segments[i].End = result[i].End
			}
		}
		if hasUntimedSegments(fullPayload.Segments) {
			fmt.Println("Error: cloud alignment left some subtitle segments without timing")
			return
		}
	}

	if outputPath == "" {
		outputPath = "align_result.json"
	}
	outputPath, _ = absoluteOutputPath(outputPath)

	outData, _ := json.MarshalIndent(map[string]any{
		"segments":        fullPayload.Segments,
		"sentence_groups": fullPayload.SentenceGroups,
		"count":           len(fullPayload.Segments),
	}, "", "  ")
	if err := os.WriteFile(outputPath, outData, 0644); err != nil {
		fmt.Printf("Error: write output failed: %v\n", err)
		return
	}
	fmt.Printf("Saved to: %s (%d segments, %d sentence groups)\n", outputPath, len(fullPayload.Segments), len(fullPayload.SentenceGroups))

	proj := resolveProjectByName("")
	if proj != nil {
		recordStep(proj, "align", segmentsPath, outputPath)
	}
}
