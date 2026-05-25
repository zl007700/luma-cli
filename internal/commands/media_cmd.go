package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/luma-cli/lumer-cli/cloud"
	"github.com/luma-cli/lumer-cli/internal/atom"
	"github.com/luma-cli/lumer-cli/internal/cmdutil"
	"github.com/luma-cli/lumer-cli/internal/output"
	"github.com/luma-cli/lumer-cli/project"
	"github.com/luma-cli/lumer-cli/subtitle"
)

const defaultVoiceName = "男声3"

func cmdASR(args []string) {
	parsed := cmdutil.Parse(args)
	if len(parsed.Positionals) < 1 {
		fmt.Println("usage: luma-cli asr <video_or_audio> [--language zh|en]")
		return
	}

	filePath := parsed.Pos(0)
	if _, err := os.Stat(filePath); err != nil {
		fmt.Printf("Error: file not found: %s\n", filePath)
		return
	}

	cfg := loadConfig()
	if cfg == nil {
		fmt.Println("Error: not logged in. Run: luma-cli auth login <card_key>")
		return
	}

	proj := resolveProjectByName("")
	language := parsed.String("language", "zh")

	fmt.Println("Uploading file...")
	fmt.Println("Submitting ASR task...")
	result, err := atom.RunASR(atom.ASROptions{
		FilePath: filePath,
		Language: language,
		CardKey:  cfg.CardKey,
	})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Printf("  Uploaded: %s\n", result.ObjectKey)
	fmt.Printf("  Task ID: %s\n", result.TaskID)

	fmt.Printf("\n--- ASR Result ---\n")
	fmt.Printf("Text (%d chars):\n%s\n", len(result.Text), result.Text)
	if len(result.Segments) > 0 {
		fmt.Printf("Segments: %d\n", len(result.Segments))
	}

	if proj != nil {
		asrPath := filepath.Join(proj.SubDir(project.DirAudio), "asr_result.json")
		data, _ := json.MarshalIndent(map[string]any{"text": result.Text, "segments": result.Segments, "language": language}, "", "  ")
		os.WriteFile(asrPath, data, 0644)
		fmt.Printf("  Saved to: %s\n", asrPath)
		recordStep(proj, "asr", filePath, asrPath)
	}
}

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

func cmdTTS(args []string) {
	parsed := cmdutil.Parse(args)
	filePath := parsed.String("file", "")
	var text string
	if filePath != "" {
		data, err := os.ReadFile(filePath)
		if err != nil {
			fmt.Printf("Error: read file %s: %v\n", filePath, err)
			return
		}
		text = strings.TrimSpace(string(data))
		if text == "" {
			fmt.Printf("Error: file %s is empty\n", filePath)
			return
		}
	} else {
		text = parsed.Pos(0)
	}
	if text == "" {
		fmt.Println("usage: luma-cli tts <text> [--file <path>] [--voice <name>] [--speech-rate <rate>] [--trim-long-silence] [--output <path>]")
		fmt.Println("")
		fmt.Println("  Provide text via positional argument or --file:")
		fmt.Println("    luma-cli tts \"你好世界\"")
		fmt.Println("    luma-cli tts --file transcript.txt")
		fmt.Println("")
		fmt.Println("  Options:")
		fmt.Printf("    --voice <name>       Voice name. Default: %s\n", defaultVoiceName)
		fmt.Println("    --speech-rate <rate> Speech rate multiplier (default: 1.1)")
		fmt.Println("    --trim-long-silence  Ask the backend to trim long pauses")
		fmt.Println("    --output <path>      Output wav path (default: ./tts_output.wav)")
		fmt.Println("")
		fmt.Println("  List voices: luma-cli asset list voice")
		fmt.Println("")
		fmt.Println("  JSON mode (luma-cli --json tts ...) outputs:")
		fmt.Println("    task_id, audio_object_key, result_object_key, output_url, output_path")
		return
	}
	voiceName := parsed.String("voice", defaultVoiceName)
	speechRate, err := parsed.Float("speech-rate", 1.1)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	trimLongSilence, err := parsed.Bool("trim-long-silence", false)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	cfg := loadConfig()
	if cfg == nil {
		fmt.Println("Error: not logged in. Run: luma-cli auth login <card_key>")
		return
	}

	voiceKey, err := atom.ResolveAssetKey("voice", voiceName, cfg.CardKey)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	proj := resolveProjectByName("")
	outputPath := parsed.String("output", "")
	if outputPath == "" && proj != nil {
		outputPath = filepath.Join(proj.SubDir(project.DirAudio), "step2_tts.wav")
	} else if outputPath == "" {
		outputPath = "step2_tts.wav"
	}
	outputPath, err = absoluteOutputPath(outputPath)
	if err != nil {
		fmt.Printf("Error: invalid output path: %v\n", err)
		return
	}

	if !runtimeOpts.JSON {
		fmt.Println("Submitting TTS task...")
		fmt.Printf("  Voice: %s\n", voiceName)
		fmt.Printf("  Text: %s\n", text)
		fmt.Printf("  Output: %s\n", outputPath)
	}
	result, err := atom.RunTTS(atom.TTSOptions{
		Text:            text,
		VoiceKey:        voiceKey,
		SpeechRate:      speechRate,
		TrimLongSilence: trimLongSilence,
		CardKey:         cfg.CardKey,
		OutputPath:      outputPath,
	})
	if err != nil {
		if runtimeOpts.JSON {
			_ = output.WriteJSON(os.Stdout, output.Envelope{OK: false, Error: err.Error()})
		} else {
			fmt.Printf("Error: %v\n", err)
		}
		return
	}

	if proj != nil && result.AudioObjectKey != "" {
		proj.LatestTTSKey = result.AudioObjectKey
		proj.Save()
	}

	recordStep(proj, "tts", text, outputPath)

	// Rename output to include content hash for traceability.
	if hashed, err := hashSuffixFile(outputPath); err == nil {
		outputPath = hashed
		result.OutputPath = hashed
	}

	// Write JSON sidecar so agents can reliably read audio_object_key without parsing stdout.
	sidecarPath := strings.TrimSuffix(outputPath, filepath.Ext(outputPath)) + ".json"
	sidecarData, _ := json.MarshalIndent(map[string]any{
		"task_id":           result.TaskID,
		"audio_object_key":  result.AudioObjectKey,
		"result_object_key": result.ResultObjectKey,
		"output_url":        result.OutputURL,
		"output_path":       outputPath,
	}, "", "  ")
	os.WriteFile(sidecarPath, sidecarData, 0644)

	if runtimeOpts.JSON {
		_ = output.WriteJSON(os.Stdout, output.Envelope{
			OK: true,
			Data: map[string]any{
				"task_id":           result.TaskID,
				"audio_object_key":  result.AudioObjectKey,
				"result_object_key": result.ResultObjectKey,
				"output_url":        result.OutputURL,
				"output_path":       outputPath,
			},
		})
		return
	}

	fmt.Printf("  Task ID: %s\n", result.TaskID)

	localFileWritten := false
	if outputPath != "" {
		if info, err := os.Stat(outputPath); err == nil && !info.IsDir() {
			localFileWritten = true
		}
	}

	if localFileWritten {
		fmt.Printf("\nDone! Saved to: %s\n", outputPath)
	} else {
		fmt.Println("\nDone! Cloud task completed, but no local file was downloaded.")
		fmt.Println("The backend did not return a downloadable URL for this task.")
	}
}

func cmdLipSync(args []string) {
	fmt.Println("usage: luma-cli lipsync --avatar <name> --audio <file> [--audio-key <key>] [--output <path>]")
	fmt.Println("")
	fmt.Println("  Options:")
	fmt.Println("    --avatar <name>             Digital avatar name")
	fmt.Println("    --audio <file>              Audio file path (uploads to cloud)")
	fmt.Println("    --audio-key <key>           Cloud audio object key (skip upload, use existing)")
	fmt.Println("    --output <path>             Output video path")
	fmt.Println("    --random-start              Start the avatar video from a random position")
	fmt.Println("    --guidance-scale <number>   Lip-sync guidance scale (default: 1.0)")
	fmt.Println("    --num-inference-steps <n>   Inference steps (default: 15)")
	fmt.Println("    --no-superres               Disable super-resolution")
	fmt.Println("    --superres-scale <n>        Super-resolution scale (default: 2)")
	fmt.Println("    --multi-shot-json <file>    JSON payload for backend multi_shot")
	fmt.Println("    --timeout <seconds>         Wait timeout for cloud task (default: 600)")
	fmt.Println("")
	fmt.Println("  List avatars: luma-cli asset list roles")

	parsed := cmdutil.Parse(args)
	avatarName := parsed.String("avatar", "")
	audioPath := parsed.String("audio", "")
	audioKey := parsed.String("audio-key", "")
	outputPath := parsed.String("output", "")
	randomStart, err := parsed.Bool("random-start", false)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	guidanceScale, err := parsed.Float("guidance-scale", 1.0)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	numInferenceSteps, err := parsed.Int("num-inference-steps", 15)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	superresScale, err := parsed.Int("superres-scale", 2)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	timeoutSec, err := parsed.Int("timeout", 600)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	multiShot, err := loadJSONMap(parsed.String("multi-shot-json", ""))
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	if avatarName == "" {
		fmt.Println("Error: --avatar is required. Use 'luma-cli asset list roles' to see available avatars.")
		return
	}

	cfg := loadConfig()
	if cfg == nil {
		fmt.Println("Error: not logged in. Run: luma-cli auth login <card_key>")
		return
	}

	proj := resolveProjectByName("")

	videoKey, err := atom.ResolveAssetKey("roles", avatarName, cfg.CardKey)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	if audioKey == "" && audioPath == "" {
		if proj != nil && proj.LatestTTSKey != "" {
			audioKey = proj.LatestTTSKey
		} else {
			fmt.Println("Error: --audio or --audio-key is required")
			return
		}
	}

	if audioKey == "" {
		if _, err := os.Stat(audioPath); err != nil {
			fmt.Printf("Error: audio file not found: %s\n", audioPath)
			return
		}
		fmt.Println("Uploading audio...")
		audioKey, err = cloud.UploadFile(audioPath, cfg.CardKey, "tts_output")
		if err != nil {
			fmt.Printf("Error: audio upload failed: %v\n", err)
			return
		}
		audioKey = atom.NormalizeResourceKey(audioKey, cfg.CardKey)
		fmt.Printf("  Uploaded: %s\n", audioKey)
	} else {
		audioKey = atom.NormalizeResourceKey(audioKey, cfg.CardKey)
	}

	if outputPath == "" {
		if proj != nil {
			outputPath = filepath.Join(proj.SubDir(project.DirOutput), "step3_lipsync.mp4")
		} else {
			outputPath = "step3_lipsync.mp4"
		}
	}
	outputPath, err = absoluteOutputPath(outputPath)
	if err != nil {
		fmt.Printf("Error: invalid output path: %v\n", err)
		return
	}

	fmt.Println("Submitting LipSync task...")
	fmt.Printf("  Avatar: %s\n", avatarName)
	fmt.Printf("  AudioKey: %s\n", audioKey)
	fmt.Printf("  Output: %s\n", outputPath)
	result, err := atom.RunLipSync(atom.LipSyncOptions{
		VideoKey:          videoKey,
		AudioKey:          audioKey,
		GuidanceScale:     guidanceScale,
		NumInferenceSteps: numInferenceSteps,
		DisableSuperres:   parsed.Has("no-superres"),
		SuperresScale:     superresScale,
		RandomStart:       randomStart,
		MultiShot:         multiShot,
		CardKey:           cfg.CardKey,
		OutputPath:        outputPath,
		TimeoutSec:        timeoutSec,
	})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Printf("  Task ID: %s\n", result.TaskID)

	// Rename output to include content hash for traceability.
	if hashed, err := hashSuffixFile(outputPath); err == nil {
		outputPath = hashed
	}

	fmt.Printf("\nDone! Saved to: %s\n", outputPath)
	recordStep(proj, "lipsync", avatarName, outputPath)
}

func cmdEnhance(args []string) {
	parsed := cmdutil.Parse(args)
	if len(parsed.Positionals) < 1 {
		fmt.Println("usage: luma-cli enhance <video> [--scale 2|4] [--output <path>]")
		return
	}

	videoPath := parsed.Pos(0)
	if _, err := os.Stat(videoPath); err != nil {
		fmt.Printf("Error: file not found: %s\n", videoPath)
		return
	}

	scale, err := parsed.Int("scale", 2)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	cfg := loadConfig()
	if cfg == nil {
		fmt.Println("Error: not logged in. Run: luma-cli auth login <card_key>")
		return
	}

	proj := resolveProjectByName("")
	outputPath := parsed.String("output", "")
	if outputPath == "" {
		if proj != nil {
			outputPath = filepath.Join(proj.SubDir(project.DirOutput), strings.TrimSuffix(filepath.Base(videoPath), filepath.Ext(videoPath))+"_enhanced"+filepath.Ext(videoPath))
		} else {
			ext := filepath.Ext(videoPath)
			outputPath = strings.TrimSuffix(videoPath, ext) + "_enhanced" + ext
		}
	}
	outputPath, err = absoluteOutputPath(outputPath)
	if err != nil {
		fmt.Printf("Error: invalid output path: %v\n", err)
		return
	}

	fmt.Println("Uploading video...")
	fmt.Println("Submitting Enhance task...")
	fmt.Printf("  Output: %s\n", outputPath)
	result, err := atom.RunEnhance(atom.EnhanceOptions{
		VideoPath:  videoPath,
		Scale:      scale,
		CardKey:    cfg.CardKey,
		OutputPath: outputPath,
	})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Printf("  Uploaded: %s\n", result.ObjectKey)
	fmt.Printf("  Task ID: %s\n", result.TaskID)

	fmt.Printf("\nDone! Saved to: %s\n", outputPath)
	recordStep(proj, "enhance", videoPath, outputPath)
}

func loadJSONMap(filePath string) (map[string]any, error) {
	if filePath == "" {
		return nil, nil
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read --multi-shot-json: %w", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("parse --multi-shot-json: %w", err)
	}
	return payload, nil
}
