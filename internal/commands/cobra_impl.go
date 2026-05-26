package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/luma-cli/lumer-cli/cloud"
	"github.com/luma-cli/lumer-cli/internal/atom"
	"github.com/luma-cli/lumer-cli/internal/output"
	"github.com/luma-cli/lumer-cli/project"
)

// --- TTS ---

func runTTS(text, voiceName string, speechRate float64, trimLongSilence bool, outputPath string) error {
	cfg, err := requireConfig()
	if err != nil {
		return err
	}
	voiceKey, err := atom.ResolveAssetKey("voice", voiceName, cfg.CardKey)
	if err != nil {
		return output.ErrSystem("%v", err)
	}
	proj := resolveProjectByName("")
	if outputPath == "" && proj != nil {
		outputPath = filepath.Join(proj.SubDir(project.DirAudio), "step2_tts.wav")
	} else if outputPath == "" {
		outputPath = "step2_tts.wav"
	}
	outputPath = resolveProjectOutput(proj, outputPath)
	outputPath, err = absoluteOutputPath(outputPath)
	if err != nil {
		return output.ErrValidation("invalid output path: %v", err)
	}
	if !runtimeOpts.JSON {
		fmt.Printf("Submitting TTS task...\n  Voice: %s\n  Output: %s\n", voiceName, outputPath)
	}
	result, err := atom.RunTTS(atom.TTSOptions{
		Text: text, VoiceKey: voiceKey, SpeechRate: speechRate,
		TrimLongSilence: trimLongSilence, CardKey: cfg.CardKey, OutputPath: outputPath,
	})
	if err != nil {
		if runtimeOpts.JSON {
			_ = output.WriteJSON(os.Stdout, output.Envelope{OK: false, Error: err.Error()})
			return output.ErrBare(4)
		}
		return output.ErrNetwork("TTS: %v", err)
	}
	if proj != nil && result.AudioObjectKey != "" {
		proj.LatestTTSKey = result.AudioObjectKey
		proj.Save()
	}
	recordStep(proj, "tts", text, outputPath)
	if hashed, err := hashSuffixFile(outputPath); err == nil {
		outputPath = hashed
		result.OutputPath = hashed
	}
	sidecarPath := strings.TrimSuffix(outputPath, filepath.Ext(outputPath)) + ".json"
	sidecarData, _ := json.MarshalIndent(map[string]any{
		"task_id": result.TaskID, "audio_object_key": result.AudioObjectKey,
		"result_object_key": result.ResultObjectKey, "output_url": result.OutputURL, "output_path": outputPath,
	}, "", "  ")
	os.WriteFile(sidecarPath, sidecarData, 0644)
	if runtimeOpts.JSON {
		_ = output.WriteJSON(os.Stdout, output.Envelope{OK: true, Data: map[string]any{
			"task_id": result.TaskID, "audio_object_key": result.AudioObjectKey,
			"result_object_key": result.ResultObjectKey, "output_url": result.OutputURL, "output_path": outputPath,
		}})
		return nil
	}
	fmt.Printf("  Task ID: %s\n", result.TaskID)
	if info, err := os.Stat(outputPath); err == nil && !info.IsDir() {
		fmt.Printf("\nDone! Output saved: %s\n", outputPath)
	} else {
		fmt.Printf("\nDone! Audio object key: %s\n", result.AudioObjectKey)
	}
	return nil
}

// --- LipSync ---

func runLipSync(avatar, audioPath, audioKey, outputPath string, randomStart bool, guidanceScale float64, numInferenceSteps float64, noSuperRes bool, superresScale int, multiShotJSON string, timeoutSec int) error {
	cfg, err := requireConfig()
	if err != nil {
		return err
	}
	avatarKey, err := atom.ResolveAssetKey("roles", avatar, cfg.CardKey)
	if err != nil {
		return output.ErrSystem("%v", err)
	}
	if audioKey == "" {
		proj := resolveProjectByName("")
		if proj != nil && proj.LatestTTSKey != "" {
			audioKey = proj.LatestTTSKey
		} else if audioPath != "" {
			if _, err := os.Stat(audioPath); err != nil {
				return output.ErrValidation("audio file not found: %s", audioPath)
			}
			result, err := cloud.UploadFile(audioPath, cfg.CardKey, "tts_output")
			if err != nil {
				return output.ErrNetwork("upload audio failed: %v", err)
			}
			audioKey = result
		} else {
			return output.ErrValidation("--audio is required when no prior TTS was made in this project")
		}
	}
	proj := resolveProjectByName("")
	if outputPath == "" && proj != nil {
		outputPath = filepath.Join(proj.SubDir(project.DirOutput), "step3_lipsync.mp4")
	} else if outputPath == "" {
		outputPath = "step3_lipsync.mp4"
	}
	outputPath = resolveProjectOutput(proj, outputPath)
	outputPath, err = absoluteOutputPath(outputPath)
	if err != nil {
		return output.ErrValidation("invalid output path: %v", err)
	}
	opts := atom.LipSyncOptions{
		VideoKey: avatarKey, AudioKey: audioKey, OutputPath: outputPath,
		RandomStart: randomStart, GuidanceScale: guidanceScale,
		NumInferenceSteps: int(numInferenceSteps), DisableSuperres: noSuperRes,
		SuperresScale: superresScale, CardKey: cfg.CardKey, TimeoutSec: timeoutSec,
	}
	if multiShotJSON != "" {
		payload, err := loadJSONMap(multiShotJSON)
		if err != nil {
			return output.ErrSystem("read multi-shot json: %v", err)
		}
		opts.MultiShot = payload
	}
	fmt.Printf("Submitting LipSync task...\n  Avatar: %s\n  AudioKey: %s\n  Output: %s\n", avatar, audioKey, outputPath)
	result, err := atom.RunLipSync(opts)
	if err != nil {
		return output.ErrSystem("%v", err)
	}
	fmt.Printf("  Task ID: %s\n", result.TaskID)
	if hashed, err := hashSuffixFile(outputPath); err == nil {
		outputPath = hashed
		result.OutputPath = hashed
	}
	recordStep(proj, "lipsync", audioKey, outputPath)
	fmt.Printf("\nDone! Saved to: %s\n", outputPath)
	return nil
}

// --- ASR ---

func runASRFile(filePath, language string) error {
	if _, err := os.Stat(filePath); err != nil {
		return output.ErrValidation("file not found: %s", filePath)
	}
	cfg, err := requireConfig()
	if err != nil {
		return err
	}
	fmt.Println("Uploading file...\nSubmitting ASR task...")
	result, err := atom.RunASR(atom.ASROptions{FilePath: filePath, Language: language, CardKey: cfg.CardKey})
	if err != nil {
		return output.ErrSystem("%v", err)
	}
	fmt.Printf("  Uploaded: %s\n  Task ID: %s\n", result.ObjectKey, result.TaskID)
	fmt.Printf("\n--- ASR Result ---\nText (%d chars):\n%s\n", len(result.Text), result.Text)
	if len(result.Segments) > 0 {
		fmt.Printf("Segments: %d\n", len(result.Segments))
	}
	return nil
}

// --- Enhance ---

func runEnhance(videoPath string, scale int, outputPath string) error {
	if _, err := os.Stat(videoPath); err != nil {
		return output.ErrValidation("file not found: %s", videoPath)
	}
	cfg, err := requireConfig()
	if err != nil {
		return err
	}
	proj := resolveProjectByName("")
	if outputPath == "" && proj != nil {
		outputPath = filepath.Join(proj.SubDir(project.DirOutput), strings.TrimSuffix(filepath.Base(videoPath), filepath.Ext(videoPath))+"_enhanced"+filepath.Ext(videoPath))
	} else if outputPath == "" {
		ext := filepath.Ext(videoPath)
		outputPath = strings.TrimSuffix(videoPath, ext) + "_enhanced" + ext
	}
	outputPath, err = absoluteOutputPath(outputPath)
	if err != nil {
		return output.ErrValidation("invalid output path: %v", err)
	}
	fmt.Printf("Uploading video...\nSubmitting Enhance task...\n  Output: %s\n", outputPath)
	result, err := atom.RunEnhance(atom.EnhanceOptions{
		VideoPath: videoPath, Scale: scale, CardKey: cfg.CardKey, OutputPath: outputPath,
	})
	if err != nil {
		return output.ErrSystem("%v", err)
	}
	fmt.Printf("  Uploaded: %s\n  Task ID: %s\n\nDone! Saved to: %s\n", result.ObjectKey, result.TaskID, outputPath)
	return nil
}

// --- Subtitle ---

func runSubtitle(input string, isTextMode bool, transcriptPath, outputPath, segmentsOutput, projectName string, maxChars, fontSize, sideMargin, bottomMargin int, fontResource, color, strokeColor, highlightColor string, skipEffects, skipHighlight bool, persona string) error {
	opts := &subtitleOptions{
		input: input, isTextMode: isTextMode, transcriptPath: transcriptPath,
		outputPath: outputPath, segmentsOutput: segmentsOutput, projectName: projectName,
		maxChars: maxChars, fontSize: fontSize, fontResource: fontResource,
		color: color, strokeColor: strokeColor, highlightColor: highlightColor,
		sideMargin: sideMargin, bottomMargin: bottomMargin,
		skipEffects: skipEffects, skipHighlight: skipHighlight, persona: persona,
	}
	if opts.maxChars <= 0 {
		opts.maxChars = 15
	}
	return cmdSubtitleMain(opts)
}
