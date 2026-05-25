package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/luma-cli/lumer-cli/internal/atom"
	"github.com/luma-cli/lumer-cli/subtitle"
)

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

func runAlignment(segments []subtitle.Segment, isTextMode bool, videoPath, cardKey string, dirs projectDirs) ([]subtitle.Segment, error) {
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

	texts := make([]string, len(segments))
	for i, seg := range segments {
		texts[i] = seg.Text
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
	if len(aligned) < len(segments) {
		return nil, fmt.Errorf("cloud alignment returned too few segments: got %d, want %d", len(aligned), len(segments))
	}
	for i := range segments {
		if i < len(aligned) {
			segments[i].Start = aligned[i].Start
			segments[i].End = aligned[i].End
		}
	}
	if hasUntimedSegments(segments) {
		return nil, fmt.Errorf("cloud alignment left some subtitle segments without timing")
	}
	fmt.Printf("  Cloud alignment applied: %d segments\n", len(segments))
	return segments, nil
}

func hasUntimedSegments(segments []subtitle.Segment) bool {
	for _, seg := range segments {
		if seg.End <= seg.Start {
			return true
		}
	}
	return false
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
