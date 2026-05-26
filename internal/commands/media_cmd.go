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
)

const defaultVoiceName = "男声3"

func cmdTTS(args []string) error {
	parsed := cmdutil.Parse(args)
	filePath := parsed.String("file", "")
	var text string
	if filePath != "" {
		data, err := os.ReadFile(filePath)
		if err != nil {
			fmt.Printf("Error: read file %s: %v\n", filePath, err)
			return nil
		}
		text = strings.TrimSpace(string(data))
		if text == "" {
			fmt.Printf("Error: file %s is empty\n", filePath)
			return nil
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
		return nil
	}

	voiceName := parsed.String("voice", defaultVoiceName)
	speechRate, err := parsed.Float("speech-rate", 1.1)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return nil
	}
	trimLongSilence, err := parsed.Bool("trim-long-silence", false)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return nil
	}

	cfg := loadConfig()
	if cfg == nil {
		fmt.Println("Error: not logged in. Run: luma-cli auth login <card_key>")
		return nil
	}

	voiceKey, err := atom.ResolveAssetKey("voice", voiceName, cfg.CardKey)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return nil
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
		return nil
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
		return nil
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
			OK:   true,
			Data: map[string]any{"task_id": result.TaskID, "audio_object_key": result.AudioObjectKey, "result_object_key": result.ResultObjectKey, "output_url": result.OutputURL, "output_path": outputPath},
		})
		return nil
	}

	fmt.Printf("  Task ID: %s\n", result.TaskID)

	localFileWritten := false
	if outputPath != "" {
		if info, err := os.Stat(outputPath); err == nil && !info.IsDir() {
			localFileWritten = true
		}
	}

	if localFileWritten {
		fmt.Printf("\nDone! Output saved: %s\n", outputPath)
	} else {
		fmt.Printf("\nDone! Audio object key: %s\n", result.AudioObjectKey)
	}
	return nil
}

func cmdLipSync(args []string) error {
	parsed := cmdutil.Parse(args)
	avatar := parsed.String("avatar", "")
	audioPath := parsed.String("audio", "")
	audioKey := parsed.String("audio-key", "")
	outputPath := parsed.String("output", "")
	randomStart, err := parsed.Bool("random-start", true)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return nil
	}

	if avatar == "" {
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
		return nil
	}

	cfg := loadConfig()
	if cfg == nil {
		fmt.Println("Error: not logged in. Run: luma-cli auth login <card_key>")
		return nil
	}

	// Resolve AvatarKey from friendly name or object key.
	avatarKey, err := atom.ResolveAssetKey("video", avatar, cfg.CardKey)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return nil
	}

	// Establish AudioKey: explicit flag, then project LatestTTSKey, then upload.
	if audioKey == "" {
		proj := resolveProjectByName("")
		if proj != nil && proj.LatestTTSKey != "" {
			audioKey = proj.LatestTTSKey
		} else if audioPath != "" {
			if _, err := os.Stat(audioPath); err != nil {
				fmt.Printf("Error: audio file not found: %s\n", audioPath)
				return nil
			}
			uploadResult, err := cloud.UploadFile(audioPath, cfg.CardKey, "tts_output")
			if err != nil {
				fmt.Printf("Error: upload audio failed: %v\n", err)
				return nil
			}
			audioKey = uploadResult
		} else {
			fmt.Println("Error: --audio is required when no prior TTS was made in this project")
			return nil
		}
	}

	proj := resolveProjectByName("")
	if outputPath == "" && proj != nil {
		outputPath = filepath.Join(proj.SubDir(project.DirOutput), "step3_lipsync.mp4")
	} else if outputPath == "" {
		outputPath = "step3_lipsync.mp4"
	}
	outputPath, err = absoluteOutputPath(outputPath)
	if err != nil {
		fmt.Printf("Error: invalid output path: %v\n", err)
		return nil
	}

	guidanceScale, err := parsed.Float("guidance-scale", 1.0)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return nil
	}
	numInferenceSteps, err := parsed.Int("num-inference-steps", 15)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return nil
	}
	noSuperRes, err := parsed.Bool("no-superres", false)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return nil
	}
	superresScale, err := parsed.Int("superres-scale", 2)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return nil
	}
	timeoutSec, err := parsed.Int("timeout", 600)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return nil
	}
	multiShotJSONPath := parsed.String("multi-shot-json", "")

	opts := atom.LipSyncOptions{
		VideoKey:          avatarKey,
		AudioKey:          audioKey,
		OutputPath:        outputPath,
		RandomStart:       randomStart,
		GuidanceScale:     guidanceScale,
		NumInferenceSteps: numInferenceSteps,
		DisableSuperres:   noSuperRes,
		SuperresScale:     superresScale,
		CardKey:           cfg.CardKey,
		TimeoutSec:        timeoutSec,
	}
	if multiShotJSONPath != "" {
		multiShotPayload, err := loadJSONMap(multiShotJSONPath)
		if err != nil {
			fmt.Printf("Error: read multi-shot json failed: %v\n", err)
			return nil
		}
		opts.MultiShot = multiShotPayload
	}

	fmt.Println("Submitting LipSync task...")
	fmt.Printf("  Avatar: %s\n", avatar)
	fmt.Printf("  AudioKey: %s\n", audioKey)
	fmt.Printf("  Output: %s\n", outputPath)
	result, err := atom.RunLipSync(opts)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return nil
	}
	fmt.Printf("  Task ID: %s\n", result.TaskID)

	// Rename output to include content hash for traceability.
	if hashed, err := hashSuffixFile(outputPath); err == nil {
		outputPath = hashed
		result.OutputPath = hashed
	}

	recordStep(proj, "lipsync", audioKey, outputPath)
	fmt.Printf("\nDone! Saved to: %s\n", outputPath)
	return nil
}

func loadJSONMap(filePath string) (map[string]any, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}
