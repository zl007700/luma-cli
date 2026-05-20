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
	"github.com/luma-cli/lumer-cli/project"
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

func cmdTTS(args []string) {
	parsed := cmdutil.Parse(args)
	if len(parsed.Positionals) < 1 {
		fmt.Println("usage: luma-cli tts <text> [--voice <name>] [--speech-rate <rate>] [--trim-long-silence] [--output <path>]")
		fmt.Println("")
		fmt.Println("  Options:")
		fmt.Printf("    --voice <name>       Voice name. Default: %s\n", defaultVoiceName)
		fmt.Println("    --speech-rate <rate> Speech rate multiplier (default: 1.1)")
		fmt.Println("    --trim-long-silence  Ask the backend to trim long pauses")
		fmt.Println("    --output <path>      Output wav path (default: ./tts_output.wav)")
		fmt.Println("")
		fmt.Println("  List voices: luma-cli asset list voice")
		return
	}

	text := parsed.Pos(0)
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

	fmt.Println("Submitting TTS task...")
	fmt.Printf("  Voice: %s\n", voiceName)
	fmt.Printf("  Text: %s\n", text)
	fmt.Printf("  Output: %s\n", outputPath)
	result, err := atom.RunTTS(atom.TTSOptions{
		Text:            text,
		VoiceKey:        voiceKey,
		SpeechRate:      speechRate,
		TrimLongSilence: trimLongSilence,
		CardKey:         cfg.CardKey,
		OutputPath:      outputPath,
	})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Printf("  Task ID: %s\n", result.TaskID)

	localFileWritten := false
	if outputPath != "" {
		if info, err := os.Stat(outputPath); err == nil && !info.IsDir() {
			localFileWritten = true
		}
	}

	if proj != nil && result.AudioObjectKey != "" {
		proj.LatestTTSKey = result.AudioObjectKey
		proj.Save()
	}

	if localFileWritten {
		fmt.Printf("\nDone! Saved to: %s\n", outputPath)
	} else {
		fmt.Println("\nDone! Cloud task completed, but no local file was downloaded.")
		fmt.Println("The backend did not return a downloadable URL for this task.")
	}
	recordStep(proj, "tts", text, outputPath)
}

func cmdLipSync(args []string) {
	fmt.Println("usage: luma-cli lipsync --avatar <name> [--audio <file>] [--output <path>]")
	fmt.Println("")
	fmt.Println("  Options:")
	fmt.Println("    --avatar <name>             Digital avatar name")
	fmt.Println("    --audio <file>              Audio file path. Default: use latest TTS output from project")
	fmt.Println("    --output <path>             Output video path")
	fmt.Println("    --random-start              Start the avatar video from a random position")
	fmt.Println("    --guidance-scale <number>   Lip-sync guidance scale (default: 1.0)")
	fmt.Println("    --num-inference-steps <n>   Inference steps (default: 15)")
	fmt.Println("    --no-superres               Disable super-resolution")
	fmt.Println("    --superres-scale <n>        Super-resolution scale (default: 2)")
	fmt.Println("    --multi-shot-json <file>    JSON payload for backend multi_shot")
	fmt.Println("")
	fmt.Println("  List avatars: luma-cli asset list roles")

	parsed := cmdutil.Parse(args)
	avatarName := parsed.String("avatar", "")
	audioPath := parsed.String("audio", "")
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

	var audioKey string
	if audioPath != "" {
		if _, err := os.Stat(audioPath); err != nil {
			fmt.Printf("Error: audio file not found: %s\n", audioPath)
			return
		}
		fmt.Println("Uploading audio...")
		audioKey, err = cloud.UploadFile(audioPath, cfg.CardKey, "lipsync_input")
		if err != nil {
			fmt.Printf("Error: audio upload failed: %v\n", err)
			return
		}
		audioKey = atom.NormalizeResourceKey(audioKey, cfg.CardKey)
		fmt.Printf("  Uploaded: %s\n", audioKey)
	} else if proj != nil && proj.LatestTTSKey != "" {
		audioKey = proj.LatestTTSKey
		fmt.Printf("  Using latest TTS audio: %s\n", audioKey)
	} else {
		fmt.Println("Error: no audio source. Use --audio <file> or run 'luma-cli tts' first in a project.")
		return
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
	})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Printf("  Task ID: %s\n", result.TaskID)

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
